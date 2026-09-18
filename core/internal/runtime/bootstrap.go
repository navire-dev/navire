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
	"github.com/navire-dev/navire/core/internal/dispatchers/registration"
	"github.com/navire-dev/navire/core/internal/ingest"
	coremetrics "github.com/navire-dev/navire/core/internal/metrics"
	prommetrics "github.com/navire-dev/navire/core/internal/metrics/prometheus"
	"github.com/navire-dev/navire/core/internal/provider"
	"github.com/navire-dev/navire/shared/logger"
	templateCatalog "github.com/navire-dev/navire/shared/template/catalog"
	"github.com/navire-dev/navire/shared/transport/redisstreams"
	"github.com/navire-dev/navire/shared/version"
)

// App holds Core dependencies and lifecycle state.
type App struct {
	cfg                   *config.Config
	logger                logger.Logger
	server                *api.Server
	ingester              *ingest.Ingester
	reportingConsumer     *redisstreams.ReportingConsumer
	heartbeatConsumer     *redisstreams.HeartbeatConsumer
	planCleaner           *redisstreams.StreamCleaner
	reportingCleaner      *redisstreams.StreamCleaner
	lifecycleCleaner      *redisstreams.StreamCleaner
	heartbeatAckPublisher redisstreams.HeartbeatAckPublisher
	lifecyclePublisher    redisstreams.LifecyclePublisher
	registration          *registration.Service
	metrics               coremetrics.Recorder
	templateCatalog       *templateCatalog.Catalog
	coreID                string
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

	redis, err := connectRedis(cfg, log)
	if err != nil {
		return nil, err
	}
	registrationService := registration.Service{Redis: redis, Secret: keys.registrationSecret}
	if err := registrationService.EnsureSecretHash(context.Background()); err != nil {
		_ = redis.Close()
		return nil, fmt.Errorf("prepare dispatcher registration: %w", err)
	}

	ingester, err := loadIngester(cfg, redis, keys.secretsEncryptionKey, log)
	if err != nil {
		_ = redis.Close()
		return nil, err
	}
	catalog, err := loadTemplates(cfg, log)
	if err != nil {
		_ = redis.Close()
		return nil, err
	}

	providerRegistry, err := provider.NewDefaultRegistry()
	if err != nil {
		_ = redis.Close()
		return nil, fmt.Errorf("register Core providers: %w", err)
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
		Redis:                redis,
		SecretsEncryptionKey: keys.secretsEncryptionKey,
		DispatchPlanKey:      keys.dispatchPlanKey,
		RegistrationSecret:   keys.registrationSecret,
		Publisher:            &redisstreams.Publisher{Client: redis},
		EndpointStore:        ingest.RedisEndpointStore{Client: redis},
		TemplatesDir:         cfg.TemplatesDir,
		TemplateCatalog:      catalog,
		HTTPClient:           &http.Client{Timeout: 10 * time.Second},
		ProviderRegistry:     providerRegistry,
		Metrics:              processMetrics,
		MetricsExporter:      processMetrics,
	})

	return &App{
		cfg:                   cfg,
		logger:                log,
		server:                server,
		ingester:              ingester,
		reportingConsumer:     &redisstreams.ReportingConsumer{Client: redis, Consumer: "core-metrics"},
		heartbeatConsumer:     &redisstreams.HeartbeatConsumer{Client: redis, Consumer: "core-heartbeats"},
		planCleaner:           &redisstreams.StreamCleaner{Client: redis, Stream: redisstreams.NotificationStream, Retention: cfg.PlanRetention, Interval: cfg.PlanCleanupInterval},
		reportingCleaner:      &redisstreams.StreamCleaner{Client: redis, Stream: redisstreams.ReportingStream, Retention: cfg.ReportingRetention, Interval: cfg.ReportingCleanupInterval},
		lifecycleCleaner:      &redisstreams.StreamCleaner{Client: redis, Stream: redisstreams.LifecycleStream, Retention: redisstreams.LifecycleRetention, Interval: redisstreams.LifecycleCleanupInterval},
		heartbeatAckPublisher: redisstreams.HeartbeatAckPublisher{Client: redis},
		lifecyclePublisher:    redisstreams.LifecyclePublisher{Client: redis},
		registration:          &registrationService,
		metrics:               processMetrics,
		templateCatalog:       catalog,
		coreID:                coreID,
	}, nil
}

func resolveCoreID() string {
	coreID, err := os.Hostname()
	if err != nil || strings.TrimSpace(coreID) == "" {
		return "core"
	}
	return coreID
}
