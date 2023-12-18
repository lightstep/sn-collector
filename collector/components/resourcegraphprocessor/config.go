package resourcegraphprocessor

import (
	"go.opentelemetry.io/collector/config/confignet"
	"go.opentelemetry.io/collector/config/configopaque"
	"go.opentelemetry.io/collector/config/configtls"
)

type Config struct {
	confignet.NetAddr `mapstructure:",squash"`
	// Optional username. Use the specified Username to authenticate the current connection
	// with one of the connections defined in the ACL list when connecting
	// to a Redis 6.0 instance, or greater, that is using the Redis ACL system.
	Username string `mapstructure:"username"`

	// Optional password. Must match the password specified in the
	// requirepass server configuration option, or the user's password when connecting
	// to a Redis 6.0 instance, or greater, that is using the Redis ACL system.
	Password configopaque.String `mapstructure:"password"`

	TLS configtls.TLSClientSetting `mapstructure:"tls,omitempty"`
}
