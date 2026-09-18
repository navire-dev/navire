package deps

import (
	"net/http"
	"time"

	"github.com/navire-dev/navire/core/internal/ingest"
	coremetrics "github.com/navire-dev/navire/core/internal/metrics"
	"github.com/navire-dev/navire/core/internal/provider"
	"github.com/navire-dev/navire/shared/logger"
	templateCatalog "github.com/navire-dev/navire/shared/template/catalog"
	"github.com/navire-dev/navire/shared/transport"
	"github.com/redis/go-redis/v9"
)

type Deps struct {
	Logger               logger.Logger
	StartTime            time.Time
	Version              string
	Commit               string
	BuildDate            string
	GoVersion            string
	TimeNow              func() time.Time // for testing, defaults to time.Now
	AllowedCIDRS         []string         // IPs allowed to access healthz/readyz endpoints
	AllowedProxies       []string         // proxies allowed to provide forwarding headers
	TrustProxy           bool             // true if running behind a trusted reverse proxy (e.g., cloudflared)
	Redis                *redis.Client
	SecretsEncryptionKey []byte
	DispatchPlanKey      []byte
	RegistrationSecret   []byte
	Publisher            transport.ExecutionPlanPublisher
	EndpointStore        ingest.EndpointStore
	TemplatesDir         string
	TemplateCatalog      *templateCatalog.Catalog
	HTTPClient           *http.Client
	ProviderRegistry     *provider.Registry
	Metrics              coremetrics.Recorder
	MetricsExporter      coremetrics.Exporter
}
