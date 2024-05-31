// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package lightstepreceiver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/lightstepreceiver"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"

	"github.com/lightstep/sn-collector/collector/lightstepreceiver/internal/metadata"
)

// This file implements factory bits for the Lightstep receiver.

const (
	// TODO: Define a new port for us to use.
	defaultBindEndpoint = "0.0.0.0:443"
)

// NewFactory creates a new Lightstep receiver factory
func NewFactory() receiver.Factory {
	return receiver.NewFactory(
		component.MustNewType(metadata.Type),
		createDefaultConfig,
		receiver.WithTraces(createTracesReceiver, metadata.TracesStability),
	)
}

// createDefaultConfig creates the default configuration for Lightstep receiver.
func createDefaultConfig() component.Config {
	return &Config{
		Protocols: Protocols{
			HTTP: &HTTPConfig{
				ServerConfig: &confighttp.ServerConfig{
					Endpoint: defaultBindEndpoint,
				},
			},
		},
	}
}

// createTracesReceiver creates a trace receiver based on provided config.
func createTracesReceiver(
	_ context.Context,
	set receiver.CreateSettings,
	cfg component.Config,
	consumer consumer.Traces,
) (receiver.Traces, error) {
	rCfg := cfg.(*Config)
	return newReceiver(rCfg, consumer, set)
}
