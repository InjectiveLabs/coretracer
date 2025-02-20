package uptrace_client

import (
	"github.com/InjectiveLabs/coretracer"
	"github.com/uptrace/uptrace-go/uptrace"
	"go.opentelemetry.io/otel/attribute"
)

func InitExporter(cfg *coretracer.Config) coretracer.ExporterShutdownFn {
	uptrace.ConfigureOpentelemetry(
		uptrace.WithDSN(cfg.CollectorDSN), // or use UPTRACE_DSN env var
		uptrace.WithServiceName(cfg.ServiceName),
		uptrace.WithServiceVersion(cfg.ServiceVersion),
		uptrace.WithDeploymentEnvironment(cfg.EnvName),
		uptrace.WithResourceAttributes(attribute.String("deployment.chain_id", cfg.ChainID)),
	)

	return uptrace.Shutdown
}
