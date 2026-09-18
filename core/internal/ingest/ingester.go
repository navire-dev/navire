package ingest

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"sync"

	"github.com/navire-dev/navire/shared/crypto"
	"github.com/navire-dev/navire/shared/logger"
	"github.com/navire-dev/navire/shared/models"
	"github.com/redis/go-redis/v9"
)

const EndpointsHashKey = "navire:config:endpoints"

// Ingester manages loading and watching target endpoint configs.
type Ingester struct {
	targetsDir           string        // Directory containing targets/*.yml files
	redis                *redis.Client // Redis client for storing endpoints
	secretsEncryptionKey []byte        // Key for encrypting auth tokens
	log                  logger.Logger // Logger
	mu                   sync.Mutex    // Prevent overlapping reconciliations
}

// New creates a new Ingester for loading target endpoints.
func New(targetsDir string, redisClient *redis.Client, secretsEncryptionKey []byte, log logger.Logger) *Ingester {
	if log == nil {
		log = logger.New("info", true)
	}

	return &Ingester{
		targetsDir:           targetsDir,
		redis:                redisClient,
		secretsEncryptionKey: secretsEncryptionKey,
		log:                  log,
	}
}

// LoadAll loads all target endpoint configs at startup.
// An empty directory is valid: it clears the runtime endpoint projection.
// Loads targets/*.yml files, encrypts auth tokens, and stores in Redis.
// Also cleans up stale endpoints (endpoints that were removed or disabled).
func (ing *Ingester) LoadAll(ctx context.Context) error {
	return ing.loadAll(ctx, false)
}

// reconcile reloads runtime configuration. Unlike startup, an intentionally
// empty configuration is valid and removes all previously active endpoints.
func (ing *Ingester) reconcile(ctx context.Context) error {
	return ing.loadAll(ctx, false)
}

func (ing *Ingester) loadAll(ctx context.Context, requireEndpoint bool) error {
	ing.mu.Lock()
	defer ing.mu.Unlock()

	ing.log.Info("loading target endpoints", logger.String("dir", ing.targetsDir))

	// Load all targets
	endpoints, err := loadAllTargets(ing.targetsDir, requireEndpoint)
	if err != nil {
		return fmt.Errorf("load targets: %w", err)
	}

	if len(endpoints) == 0 {
		ing.log.Warn("no enabled endpoints found", logger.String("dir", ing.targetsDir))
	} else {
		ing.log.Info("loaded endpoints", logger.Int("count", len(endpoints)))
	}

	// Prepare the complete snapshot before touching Redis. If parsing,
	// encryption or marshaling fails, the last valid snapshot stays active.
	prepared := make(map[string][]byte, len(endpoints))
	for fullName, ep := range endpoints {
		if ep.Auth.Value != "" && ep.Auth.Type != models.AuthNone {
			encrypted, err := crypto.Encrypt(ep.Auth.Value, ing.secretsEncryptionKey)
			if err != nil {
				return fmt.Errorf("encrypt token for endpoint %s: %w", fullName, err)
			}
			ep.Auth.Value = encrypted
		}

		data, err := json.Marshal(ep)
		if err != nil {
			return fmt.Errorf("marshal endpoint %s: %w", fullName, err)
		}
		prepared[fullName] = data
	}

	fields := make([]string, 0, len(prepared))
	for field := range prepared {
		fields = append(fields, field)
	}
	sort.Strings(fields)

	// Replace the collection in one Redis transaction. A hash keeps discovery
	// scoped to endpoints and avoids scanning the complete Redis keyspace.
	if _, err := ing.redis.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
		pipe.Del(ctx, EndpointsHashKey)
		for _, field := range fields {
			pipe.HSet(ctx, EndpointsHashKey, field, prepared[field])
		}
		return nil
	}); err != nil {
		return fmt.Errorf("replace endpoint snapshot: %w", err)
	}

	ing.log.Info("✅ endpoints synced", logger.Int("stored", len(endpoints)))
	return nil
}

// Watch starts the file watcher (background goroutine).
func (ing *Ingester) Watch(ctx context.Context) error {
	watcher := NewWatcher(ing.targetsDir, ing.reconcile, ing.log)
	return watcher.Start(ctx)
}
