package runtime

import (
	"context"
	"fmt"

	"github.com/navire-dev/navire/core/internal/config"
	sharedcrypto "github.com/navire-dev/navire/core/internal/crypto"
	"github.com/navire-dev/navire/core/internal/ingest"
	redisClient "github.com/navire-dev/navire/core/internal/redis"
	"github.com/navire-dev/navire/shared/logger"
	templateCatalog "github.com/navire-dev/navire/shared/template/catalog"
	"github.com/redis/go-redis/v9"
)

type runtimeKeys struct {
	secretsEncryptionKey []byte
	executionPlanKey     []byte
}

func loadKeys(cfg *config.Config) (runtimeKeys, error) {
	secretsEncryptionKey, err := validateSecretsKey(cfg)
	if err != nil {
		return runtimeKeys{}, err
	}
	executionPlanKey, err := parseExecutionPlanKey(cfg)
	if err != nil {
		return runtimeKeys{}, err
	}
	return runtimeKeys{
		secretsEncryptionKey: secretsEncryptionKey,
		executionPlanKey:     executionPlanKey,
	}, nil
}

func validateSecretsKey(cfg *config.Config) ([]byte, error) {
	key, err := sharedcrypto.ValidateSecretsEncryptionKey(cfg.SecretsEncryptionKey)
	if err != nil {
		return nil, err
	}
	return key, nil
}

func parseExecutionPlanKey(cfg *config.Config) ([]byte, error) {
	return sharedcrypto.ParseKey(cfg.ExecutionPlanKey, "NAVIRE_EXECUTION_PLAN_KEY")
}

func connectRedis(cfg *config.Config, log logger.Logger) (*redis.Client, error) {
	client, err := redisClient.NewClient(cfg.Redis)
	if err != nil {
		return nil, fmt.Errorf("connect redis: %w", err)
	}
	if err := redisClient.WaitForReady(context.Background(), client, cfg.Redis, log); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("connect redis: %w", err)
	}
	return client, nil
}

func loadIngester(cfg *config.Config, client *redis.Client, encryptionKey []byte, log logger.Logger) (*ingest.Ingester, error) {
	ingester := ingest.New(cfg.TargetsDir, client, encryptionKey, log)
	if err := ingester.LoadAll(context.Background()); err != nil {
		return nil, fmt.Errorf("load provider configs: %w", err)
	}
	log.Info("✅ Provider configs loaded successfully")
	return ingester, nil
}

func loadTemplates(cfg *config.Config, log logger.Logger) (*templateCatalog.Catalog, error) {
	catalog := templateCatalog.NewCatalog(cfg.TemplatesDir)
	issues, err := catalog.Reload()
	if err != nil {
		return nil, fmt.Errorf("load templates: %w", err)
	}
	for _, issue := range issues {
		log.Warn("template discarded", logger.String("file", issue.File), logger.Error(issue.Err))
	}
	log.Info("✅ Templates loaded",
		logger.String("dir", cfg.TemplatesDir),
		logger.Int("loaded", catalog.Count()),
		logger.Int("discarded", len(issues)))
	return catalog, nil
}
