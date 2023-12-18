package resourceapiextension

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"go.opentelemetry.io/collector/component"
	"go.uber.org/zap"
)

type resourceApiExtension struct {
	config   Config
	logger   *zap.Logger
	server   *http.Server
	settings component.TelemetrySettings
	stopCh   chan struct{}
}

func (re *resourceApiExtension) Start(_ context.Context, host component.Host) error {
	re.logger.Info("Starting resource_api extension", zap.Any("config", re.config))
	ln, err := re.config.ToListener()
	if err != nil {
		return fmt.Errorf("failed to bind to address %s: %w", re.config.Endpoint, err)
	}

	re.server, err = re.config.ToServer(host, re.settings, nil)
	if err != nil {
		return err
	}

	mux := http.NewServeMux()
	mux.Handle("/", re.baseHandler())
	re.server.Handler = mux
	re.stopCh = make(chan struct{})
	go func() {
		defer close(re.stopCh)

		// The listener ownership goes to the server.
		if err = re.server.Serve(ln); !errors.Is(err, http.ErrServerClosed) && err != nil {
			_ = re.settings.ReportComponentStatus(component.NewFatalErrorEvent(err))
		}
	}()

	return nil
}

func (re *resourceApiExtension) Shutdown(context.Context) error {
	if re.server == nil {
		return nil
	}
	err := re.server.Close()
	if re.stopCh != nil {
		<-re.stopCh
	}
	return err
}

func (re *resourceApiExtension) baseHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("TODO return resources"))
	})
}

func newServer(config Config, settings component.TelemetrySettings) *resourceApiExtension {
	hc := &resourceApiExtension{
		config:   config,
		logger:   settings.Logger,
		settings: settings,
	}

	hc.logger = settings.Logger

	return hc
}
