package runtime

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/navire-dev/navire/core/internal/api"
	"github.com/navire-dev/navire/core/internal/api/deps"
	"github.com/navire-dev/navire/core/internal/config"
	coredispatcher "github.com/navire-dev/navire/core/internal/dispatcher"
	"github.com/navire-dev/navire/core/internal/dispatcher/sinks"
	"github.com/navire-dev/navire/core/internal/ingest"
	coremetrics "github.com/navire-dev/navire/core/internal/metrics"
	prommetrics "github.com/navire-dev/navire/core/internal/metrics/prometheus"
	"github.com/navire-dev/navire/core/internal/provider"
	"github.com/navire-dev/navire/core/internal/transport/redisstreams"
	"github.com/navire-dev/navire/shared/logger"
	"github.com/navire-dev/navire/shared/models"
	sharedproviders "github.com/navire-dev/navire/shared/providers"
	templateCatalog "github.com/navire-dev/navire/shared/template/catalog"
	"github.com/navire-dev/navire/shared/version"
	"github.com/redis/go-redis/v9"
)

// App holds Core dependencies and lifecycle state.
type App struct {
	cfg                *config.Config
	logger             logger.Logger
	server             *api.Server
	ingester           *ingest.Ingester
	planCleaners       []*redisstreams.StreamCleaner
	lifecycleCleaner   *redisstreams.StreamCleaner
	lifecyclePublisher redisstreams.LifecyclePublisher
	dispatcher         *coredispatcher.Service
	planConsumer       *redisstreams.PlanConsumer
	metrics            coremetrics.Recorder
	templateCatalog    *templateCatalog.Catalog
	coreID             string
}

// New builds a runtime Core from the loaded configuration.
func New(cfg *config.Config) (*App, error) {
	if cfg == nil {
		return nil, fmt.Errorf("core config is nil")
	}
	started := time.Now()
	log := logger.New(cfg.LogLevel, cfg.PrettyLog)
	log.Info("starting Navire Core",
		logger.String("version", version.Version),
		logger.String("commit", version.Commit),
		logger.String("build_date", version.HumanBuildDate(version.BuildDate, time.UTC, "DD-MM-YYYY hh:mm:ss TZ")),
		logger.String("go_version", version.GoVersion),
	)
	keys, err := loadKeys(cfg)
	if err != nil {
		return nil, err
	}
	coreID := resolveCoreID()

	redisClient, err := connectRedis(cfg, log)
	if err != nil {
		return nil, err
	}
	ingester, err := loadIngester(cfg, redisClient, keys.secretsEncryptionKey, log)
	if err != nil {
		_ = redisClient.Close()
		return nil, err
	}
	catalog, err := loadTemplates(cfg, log)
	if err != nil {
		_ = redisClient.Close()
		return nil, err
	}

	providerRegistry, err := provider.NewDefaultRegistry()
	if err != nil {
		_ = redisClient.Close()
		return nil, fmt.Errorf("register Core providers: %w", err)
	}
	sinkRegistry, err := sinks.NewDefault()
	if err != nil {
		_ = redisClient.Close()
		return nil, fmt.Errorf("register dispatcher providers: %w", err)
	}
	httpClient := &http.Client{Timeout: 10 * time.Second}
	dispatcherService, err := coredispatcher.New(httpClient, keys.executionPlanKey, sinkRegistry, log)
	if err != nil {
		_ = redisClient.Close()
		return nil, fmt.Errorf("build dispatcher: %w", err)
	}
	processMetrics := prommetrics.New(started)
	processMetrics.SetRedisConnected(true)
	server := api.New(cfg, log, deps.Deps{
		Logger:               log,
		StartTime:            started,
		Version:              version.Version,
		Commit:               version.Commit,
		BuildDate:            version.BuildDate,
		GoVersion:            version.GoVersion,
		TimeNow:              time.Now,
		AllowedCIDRS:         cfg.AllowedCIDRS,
		AllowedProxies:       cfg.AllowedProxies,
		TrustProxy:           cfg.TrustProxy,
		Redis:                redisClient,
		SecretsEncryptionKey: keys.secretsEncryptionKey,
		ExecutionPlanKey:     keys.executionPlanKey,
		Publisher:            &redisstreams.Publisher{Client: redisClient},
		EndpointStore:        ingest.RedisEndpointStore{Client: redisClient},
		TemplatesDir:         cfg.TemplatesDir,
		TemplateCatalog:      catalog,
		HTTPClient:           httpClient,
		ProviderRegistry:     providerRegistry,
		Metrics:              processMetrics,
		MetricsExporter:      processMetrics,
	})

	return &App{
		cfg:          cfg,
		logger:       log,
		server:       server,
		ingester:     ingester,
		planCleaners: planCleaners(redisClient, cfg.PlanRetention, cfg.PlanCleanupInterval),
		lifecycleCleaner: &redisstreams.StreamCleaner{
			Client:    redisClient,
			Stream:    redisstreams.LifecycleStream,
			Retention: redisstreams.LifecycleRetention,
			Interval:  redisstreams.LifecycleCleanupInterval,
		},
		lifecyclePublisher: redisstreams.LifecyclePublisher{Client: redisClient},
		dispatcher:         dispatcherService,
		planConsumer: &redisstreams.PlanConsumer{
			Client:   redisClient,
			Streams:  redisstreams.NotificationStreams(sharedproviders.ActiveIDs()),
			Group:    redisstreams.NotificationGroup,
			Consumer: "core-" + coreID,
			Workers:  cfg.DispatcherWorkers,
			OnConsume: func(_ context.Context, plan models.ExecutionPlan) {
				processMetrics.StreamConsumed(string(plan.Target.Provider))
				log.Info("execution plan received",
					logger.String("message_id", plan.MessageID),
					logger.String("transport_message_id", plan.TransportMessageID),
					logger.String("target", plan.Target.Name),
					logger.String("provider", string(plan.Target.Provider)))
			},
			OnAck: func(_ context.Context, plan models.ExecutionPlan) {
				processMetrics.StreamAcked(string(plan.Target.Provider))
				log.Info("execution plan acknowledged",
					logger.String("message_id", plan.MessageID),
					logger.String("transport_message_id", plan.TransportMessageID),
					logger.String("target", plan.Target.Name),
					logger.String("provider", string(plan.Target.Provider)))
			},
		},
		metrics:         processMetrics,
		templateCatalog: catalog,
		coreID:          coreID,
	}, nil
}

func planCleaners(client *redis.Client, retention, interval time.Duration) []*redisstreams.StreamCleaner {
	providers := sharedproviders.ActiveIDs()
	cleaners := make([]*redisstreams.StreamCleaner, 0, len(providers))
	for _, provider := range providers {
		cleaners = append(cleaners, &redisstreams.StreamCleaner{
			Client:    client,
			Stream:    redisstreams.NotificationStream(provider),
			Retention: retention,
			Interval:  interval,
		})
	}
	return cleaners
}

func resolveCoreID() string {
	coreID, err := os.Hostname()
	if err != nil || strings.TrimSpace(coreID) == "" {
		return "core"
	}
	return coreID
}
