package resourceapiextension

import (
	"errors"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/confighttp"
)

// Config has the configuration for the extension enabling the health check
// extension, used to report the health status of the service.
type Config struct {
	confighttp.HTTPServerSettings `mapstructure:",squash"`
}

var _ component.Config = (*Config)(nil)
var (
	errNoEndpointProvided = errors.New("bad config: endpoint must be specified")
)

// Validate checks if the extension configuration is valid
func (cfg *Config) Validate() error {
	if cfg.Endpoint == "" {
		return errNoEndpointProvided
	}
	return nil
}
