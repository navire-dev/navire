package runtime

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/navire-dev/navire/dispatcher/internal/config"
	"github.com/navire-dev/navire/dispatcher/internal/sender"
	"github.com/navire-dev/navire/dispatcher/internal/sinks"
	"github.com/navire-dev/navire/shared/lifecycle"
	"github.com/navire-dev/navire/shared/logger"
	"github.com/navire-dev/navire/shared/models"
	redisClient "github.com/navire-dev/navire/shared/redis"
	"github.com/navire-dev/navire/shared/reporting"
	"github.com/navire-dev/navire/shared/transport/redisstreams"
	"github.com/navire-dev/navire/shared/version"
	"github.com/redis/go-redis/v9"
)

const (
	registrationRetryInitial = 10 * time.Second
	registrationRetryMax     = 30 * time.Second
	redisRetryInitial        = 10 * time.Second
	redisRetryMax            = 30 * time.Second
)

type App struct {
	cfg                  *config.Config
	logger               logger.Logger
	startedAt            time.Time
	redis                *redis.Client
	consumer             *redisstreams.Consumer
	lifecycleConsumer    *redisstreams.LifecycleConsumer
	heartbeatAckConsumer *redisstreams.HeartbeatAckConsumer
	sender               sender.Service
	reportingPublisher   redisstreams.ReportingPublisher
	heartbeatPublisher   redisstreams.HeartbeatPublisher
	registrationSecret   []byte
	registrationClient   *http.Client
	registrationToken    string
	registrationMu       sync.RWMutex
	registrationWake     chan struct{}
	heartbeatMu          sync.Mutex
	pendingHeartbeats    map[string]time.Time
	coreAvailable        bool
}

func New(cfg *config.Config) (*App, error) {
	if cfg == nil {
		return nil, fmt.Errorf("dispatcher config is nil")
	}
	log := logger.New(cfg.LogLevel, cfg.PrettyLog)
	log.Info("starting Navire DSPC",
		logger.String("version", version.Version),
		logger.String("commit", version.Commit),
		logger.String("build_date", version.HumanBuildDate(version.BuildDate, time.UTC, "DD-MM-YYYY hh:mm:ss TZ")),
		logger.String("go_version", version.GoVersion),
	)
	dispatchPlanKey, registrationSecret, err := loadSecurity(cfg)
	if err != nil {
		return nil, err
	}
	client, err := redisClient.NewClient(cfg.Redis)
	if err != nil {
		return nil, fmt.Errorf("configure redis: %w", err)
	}
	providerRegistry, err := sinks.NewDefault()
	if err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("register sender providers: %w", err)
	}

	consumer := &redisstreams.Consumer{Client: client, Consumer: cfg.ConsumerID}
	lifecycleConsumer := &redisstreams.LifecycleConsumer{
		Client:   client,
		Consumer: cfg.ConsumerID,
		Group:    redisstreams.LifecycleGroup(cfg.ConsumerID),
	}
	heartbeatAckConsumer := &redisstreams.HeartbeatAckConsumer{
		Client:       client,
		DispatcherID: cfg.ConsumerID,
	}
	reportingPublisher := redisstreams.ReportingPublisher{Client: client}
	heartbeatPublisher := redisstreams.HeartbeatPublisher{Client: client}
	app := &App{
		cfg:                  cfg,
		logger:               log,
		startedAt:            time.Now().UTC(),
		redis:                client,
		consumer:             consumer,
		lifecycleConsumer:    lifecycleConsumer,
		heartbeatAckConsumer: heartbeatAckConsumer,
		sender:               sender.Service{HTTPClient: newHTTPClient(), DispatchPlanKey: dispatchPlanKey, Registry: providerRegistry, Logger: log},
		reportingPublisher:   reportingPublisher,
		heartbeatPublisher:   heartbeatPublisher,
		registrationSecret:   registrationSecret,
		registrationClient:   newHTTPClient(),
		registrationWake:     make(chan struct{}, 1),
		pendingHeartbeats:    make(map[string]time.Time),
	}
	configureAckHandlers(app)
	return app, nil
}

func newHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 10 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

func configureAckHandlers(app *App) {
	app.lifecycleConsumer.OnAck = func(_ context.Context, event lifecycle.Event) {
		app.logger.Debug("Core lifecycle event acknowledged",
			logger.String("kind", string(event.Kind)),
			logger.String("core_id", event.CoreID),
		)
	}
	app.consumer.OnAck = func(ctx context.Context, plan models.ExecutionPlan) {
		event := reporting.Event{
			ID:                plan.TransportMessageID + ":delivery-ack",
			Kind:              reporting.KindDeliveryAck,
			OccurredAt:        time.Now().UTC(),
			DispatcherID:      app.cfg.ConsumerID,
			RegistrationToken: app.currentRegistrationToken(),
			MessageID:         plan.MessageID,
			TargetName:        plan.Target.Name,
		}
		if err := app.reportingPublisher.Publish(ctx, event); err != nil {
			app.logger.Errorf("publish acknowledgement reporting: %v", err)
		}
	}
}
