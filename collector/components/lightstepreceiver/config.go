// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package lightstepreceiver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/lightstepreceiver"

import (
	"errors"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/confmap"
)

const (
	protocolsFieldName = "protocols"
	protoHTTP          = "http"
)

type HTTPConfig struct {
	*confighttp.ServerConfig `mapstructure:",squash"`
}

// Protocols is the configuration for the supported protocols.
type Protocols struct {
	HTTP *HTTPConfig `mapstructure:"http"`
}

// Config defines configuration for the Lightstep receiver.
type Config struct {
	// Protocols is the configuration for the supported protocols, currently HTTP.
	Protocols `mapstructure:"protocols"`
}

var _ component.Config = (*Config)(nil)

// Validate checks the receiver configuration is valid
func (cfg *Config) Validate() error {
	if cfg.HTTP == nil {
		return errors.New("must specify at least one protocol when using the Lightstep receiver")
	}
	return nil
}

// Unmarshal a confmap.Conf into the config struct.
func (cfg *Config) Unmarshal(componentParser *confmap.Conf) error {
	if componentParser == nil {
		return fmt.Errorf("nil config for lightstepreceiver")
	}

	err := componentParser.Unmarshal(cfg)
	if err != nil {
		return err
	}
	protocols, err := componentParser.Sub(protocolsFieldName)
	if err != nil {
		return err
	}

	if !protocols.IsSet(protoHTTP) {
		cfg.HTTP = nil
	}
	return nil
}
