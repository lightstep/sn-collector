// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package lightstepreceiver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/lightstepreceiver"

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/receiver"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/lightstep/sn-collector/collector/lightstepreceiver/internal/collectorpb"
)

const (
	ContentType            = "Content-Type"
	ContentTypeOctetStream = "application/octet-stream"
)

var errNextConsumerRespBody = []byte(`"Internal Server Error"`)

// lightstepReceiver type is used to handle spans received in the Lightstep format.
type lightstepReceiver struct {
	consumer consumer.Traces

	shutdownWG sync.WaitGroup
	server     *http.Server
	config     *Config

	settings receiver.CreateSettings
}

var _ http.Handler = (*lightstepReceiver)(nil)

// newReceiver creates a new lightstepReceiver reference.
func newReceiver(config *Config, consumer consumer.Traces, settings receiver.CreateSettings) (*lightstepReceiver, error) {
	lr := &lightstepReceiver{
		consumer: consumer,
		config:   config,
		settings: settings,
	}
	return lr, nil
}

// Start spins up the receiver's HTTP server and makes the receiver start its processing.
func (lr *lightstepReceiver) Start(_ context.Context, host component.Host) error {
	if host == nil {
		return errors.New("nil host")
	}

	var err error
	lr.server, err = lr.config.HTTP.ToServer(host, lr.settings.TelemetrySettings, lr)
	if err != nil {
		return err
	}

	var listener net.Listener
	listener, err = lr.config.HTTP.ToListener()
	if err != nil {
		return err
	}
	lr.shutdownWG.Add(1)
	go func() {
		defer lr.shutdownWG.Done()

		if errHTTP := lr.server.Serve(listener); !errors.Is(errHTTP, http.ErrServerClosed) && errHTTP != nil {
			lr.settings.TelemetrySettings.ReportStatus(component.NewFatalErrorEvent(errHTTP))
		}
	}()

	return nil
}

// Shutdown tells the receiver that should stop reception,
// giving it a chance to perform any necessary clean-up and shutting down
// its HTTP server.
func (lr *lightstepReceiver) Shutdown(context.Context) error {
	var err error
	if lr.server != nil {
		err = lr.server.Close()
	}
	lr.shutdownWG.Wait()
	return err
}

// The lightstepReceiver receives spans from endpoint /api/v2/reports
// unmarshalls them and sends them along to `consumer`.
// Observe we don't actually check for the endpoint path here.
func (lr *lightstepReceiver) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	receive := time.Now()
	ctx := r.Context()

	// Now deserialize and process the spans.
	pr := r.Body
	slurp, _ := io.ReadAll(pr)
	if c, ok := pr.(io.Closer); ok {
		_ = c.Close()
	}
	_ = r.Body.Close()

	var reportRequest = &collectorpb.ReportRequest{}
	err := proto.Unmarshal(slurp, reportRequest)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var td ptrace.Traces
	td, err = ToTraces(reportRequest)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	consumerErr := lr.consumer.ConsumeTraces(ctx, td)

	if consumerErr != nil {
		// Transient error, due to some internal condition.
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write(errNextConsumerRespBody)
		return
	}

	resp := &collectorpb.ReportResponse{
		ReceiveTimestamp:  timestamppb.New(receive),
		TransmitTimestamp: timestamppb.New(time.Now()),
	}
	bytes, err := proto.Marshal(resp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Finally send back the response "Accepted"
	w.WriteHeader(http.StatusAccepted)
	w.Header().Set(ContentType, ContentTypeOctetStream)
	w.Write(bytes)
}
