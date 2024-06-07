// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package lightstepreceiver

import (
	"bytes"
	"context"
	"net"
	"net/http"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/receiver/receivertest"

	"github.com/lightstep/sn-collector/collector/lightstepreceiver/internal/collectorpb"
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

func TestSimpleRequest(t *testing.T) {
	addr := findAvailableAddress(t)
	cfg := &Config{
		Protocols: Protocols{
			HTTP: &HTTPConfig{
				ServerConfig: &confighttp.ServerConfig{
					Endpoint: addr,
				},
			},
		},
	}
	sink := new(consumertest.TracesSink)

	traceReceiver, err := newReceiver(cfg, sink, receivertest.NewNopCreateSettings())
	require.NoError(t, err, "Failed to create receiver: %v", err)
	err = traceReceiver.Start(context.Background(), componenttest.NewNopHost())
	require.NoError(t, err, "Failed to start receiver: %v", err)
	t.Cleanup(func() { require.NoError(t, traceReceiver.Shutdown(context.Background())) })

	httpReq, err := createHttpRequest(addr, createSimpleRequest())
	require.NoError(t, err)

	client := http.Client{}
	httpResp, err := client.Do(httpReq)
	require.NoError(t, err)
	require.Equal(t, httpResp.StatusCode, 202)

	traces := sink.AllTraces()
	assert.Equal(t, len(traces), 1)
	assert.Equal(t, traces[0], func() ptrace.Traces {
		td := ptrace.NewTraces()
		rs := td.ResourceSpans().AppendEmpty()

		r := rs.Resource()
		rattrs := r.Attributes()
		rattrs.PutStr("lightstep.component_name", "GatewayService")
		rattrs.PutStr("service.name", "GatewayService") // derived

		sss := rs.ScopeSpans().AppendEmpty()
		scope := sss.Scope()
		scope.SetName("lightstep-receiver")
		scope.SetVersion("0.0.1")

		spans := sss.Spans()
		span1 := spans.AppendEmpty()
		span1.SetName("span1")
		return td
	}())
}

func createHttpRequest(addr string, req *collectorpb.ReportRequest) (*http.Request, error) {
	buff, err := proto.Marshal(req)
	if err != nil {
		return nil, err
	}

	requestBody := bytes.NewReader(buff)
	request, err := http.NewRequest("POST", "http://"+addr, requestBody)
	if err != nil {
		return nil, err
	}

	request.Header.Set(ContentType, ContentTypeOctetStream)
	request.Header.Set("Accept", ContentTypeOctetStream)

	return request, nil
}

func createSimpleRequest() *collectorpb.ReportRequest {
	return &collectorpb.ReportRequest{
		Reporter: &collectorpb.Reporter{
			Tags: []*collectorpb.KeyValue{
				{
					Key: "lightstep.component_name",
					Value: &collectorpb.KeyValue_StringValue{
						StringValue: "GatewayService",
					},
				},
			},
		},
		Spans: []*collectorpb.Span{
			{
				OperationName: "span1",
			},
		},
	}
}

// Copied from the testutils package in the main collector repo.
func findAvailableAddress(t testing.TB) string {
	ln, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err, "Failed to get a free local port")
	// There is a possible race if something else takes this same port before
	// the test uses it, however, that is unlikely in practice.
	defer func() {
		assert.NoError(t, ln.Close())
	}()
	return ln.Addr().String()
}
