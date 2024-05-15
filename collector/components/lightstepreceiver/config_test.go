// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package lightstepreceiver

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/config/configtls"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/confmaptest"
)

func TestUnmarshalDefaultConfig(t *testing.T) {
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "default.yaml"))
	require.NoError(t, err)
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	assert.NoError(t, component.UnmarshalConfig(cm, cfg))
	assert.Equal(t, factory.CreateDefaultConfig(), cfg)
}

func TestUnmarshalConfigOnlyHTTP(t *testing.T) {
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "only_http.yaml"))
	require.NoError(t, err)
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	assert.NoError(t, component.UnmarshalConfig(cm, cfg))

	defaultOnlyHTTP := factory.CreateDefaultConfig().(*Config)
	assert.Equal(t, defaultOnlyHTTP, cfg)
}

func TestUnmarshalConfigOnlyHTTPNull(t *testing.T) {
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "only_http_null.yaml"))
	require.NoError(t, err)
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	assert.NoError(t, component.UnmarshalConfig(cm, cfg))

	defaultOnlyHTTP := factory.CreateDefaultConfig().(*Config)
	assert.Equal(t, defaultOnlyHTTP, cfg)
}

func TestUnmarshalConfigOnlyHTTPEmptyMap(t *testing.T) {
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "only_http_empty_map.yaml"))
	require.NoError(t, err)
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	assert.NoError(t, component.UnmarshalConfig(cm, cfg))

	defaultOnlyHTTP := factory.CreateDefaultConfig().(*Config)
	assert.Equal(t, defaultOnlyHTTP, cfg)
}

func TestUnmarshalConfig(t *testing.T) {
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "config.yaml"))
	require.NoError(t, err)
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	assert.NoError(t, component.UnmarshalConfig(cm, cfg))
	assert.Equal(t,
		&Config{
			Protocols: Protocols{
				HTTP: &HTTPConfig{
					ServerConfig: &confighttp.ServerConfig{
						Endpoint: "0.0.0.0:443",
						TLSSetting: &configtls.ServerConfig{
							Config: configtls.Config{
								CertFile: "test.crt",
								KeyFile:  "test.key",
							},
						},
						CORS: &confighttp.CORSConfig{
							AllowedOrigins: []string{"https://*.test.com", "https://test.com"},
							MaxAge:         7200,
						},
					},
				},
			},
		}, cfg)

}

func TestUnmarshalConfigTypoDefaultProtocol(t *testing.T) {
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "typo_default_proto_config.yaml"))
	require.NoError(t, err)
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	assert.EqualError(t, component.UnmarshalConfig(cm, cfg), "1 error(s) decoding:\n\n* 'protocols' has invalid keys: htttp")
}

func TestUnmarshalConfigInvalidProtocol(t *testing.T) {
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "bad_proto_config.yaml"))
	require.NoError(t, err)
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	assert.EqualError(t, component.UnmarshalConfig(cm, cfg), "1 error(s) decoding:\n\n* 'protocols' has invalid keys: thrift")
}

func TestUnmarshalConfigEmptyProtocols(t *testing.T) {
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "bad_no_proto_config.yaml"))
	require.NoError(t, err)
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	assert.NoError(t, component.UnmarshalConfig(cm, cfg))
	assert.EqualError(t, component.ValidateConfig(cfg), "must specify at least one protocol when using the Lightstep receiver")
}

func TestUnmarshalConfigEmpty(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	assert.NoError(t, component.UnmarshalConfig(confmap.New(), cfg))
	assert.EqualError(t, component.ValidateConfig(cfg), "must specify at least one protocol when using the Lightstep receiver")
}
