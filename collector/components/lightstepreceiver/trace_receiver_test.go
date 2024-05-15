// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package lightstepreceiver

import (
	"context"
	"net"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

func TestNew(t *testing.T) {
	type args struct {
		address      string
		nextConsumer consumer.Traces
	}
	tests := []struct {
		name    string
		args    args
		wantErr error
	}{
		{
			name: "happy path",
			args: args{
				nextConsumer: consumertest.NewNop(),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Protocols: Protocols{
					HTTP: &HTTPConfig{
						ServerConfig: &confighttp.ServerConfig{
							Endpoint: "0.0.0.0:443",
						},
					},
				},
			}

			got, err := newReceiver(cfg, tt.args.nextConsumer, receivertest.NewNopCreateSettings())
			require.Equal(t, tt.wantErr, err)
			if tt.wantErr == nil {
				require.NotNil(t, got)
			} else {
				require.Nil(t, got)
			}
		})
	}
}

func TestReceiverPortAlreadyInUse(t *testing.T) {
	l, err := net.Listen("tcp", "localhost:")
	require.NoError(t, err, "failed to open a port: %v", err)
	defer l.Close()
	_, portStr, err := net.SplitHostPort(l.Addr().String())
	require.NoError(t, err, "failed to split listener address: %v", err)
	cfg := &Config{
		Protocols: Protocols{
			HTTP: &HTTPConfig{
				ServerConfig: &confighttp.ServerConfig{
					Endpoint: "localhost:" + portStr,
				},
			},
		},
	}
	traceReceiver, err := newReceiver(cfg, consumertest.NewNop(), receivertest.NewNopCreateSettings())
	require.NoError(t, err, "Failed to create receiver: %v", err)
	err = traceReceiver.Start(context.Background(), componenttest.NewNopHost())
	require.Error(t, err)
}

