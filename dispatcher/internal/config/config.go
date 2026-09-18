package config

import (
	"os"

	sharedconfig "github.com/navire-dev/navire/shared/config"
	redisClient "github.com/navire-dev/navire/shared/redis"
)

type Config struct {
	LogLevel           string
	PrettyLog          bool
	DispatchPlanKey    string
	RegistrationSecret string
	CoreURL            string
	Redis              redisClient.Config
	ConsumerID         string
}

func Load() (*Config, error) {
	loader := sharedconfig.NewLoader()
	consumerID := loader.String("DISPATCHER_ID", "")
	if consumerID == "" {
		consumerID, _ = os.Hostname()
	}
	if consumerID == "" {
		consumerID = "dispatcher"
	}
	cfg := &Config{
		LogLevel:           loader.String("DISPATCHER_LOG_LEVEL", "info"),
		PrettyLog:          loader.Bool("DISPATCHER_PRETTY_LOG", true),
		DispatchPlanKey:    loader.RequiredString("NAVIRE_DISPATCH_PLAN_KEY"),
		RegistrationSecret: loader.RequiredString("NAVIRE_DSPC_REGISTRATION_SECRET"),
		CoreURL:            loader.String("NAVIRE_CORE_URL", "http://localhost:8080"),
		Redis:              redisClient.Load(loader),
		ConsumerID:         consumerID,
	}
	if err := loader.Err(); err != nil {
		return nil, err
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) Validate() error {
	if err := sharedconfig.ValidateLogLevel("DISPATCHER_LOG_LEVEL", c.LogLevel); err != nil {
		return err
	}
	return c.Redis.Validate()
}
