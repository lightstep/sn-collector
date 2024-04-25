// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package httpcheckreceiver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/httpcheckreceiver"

import (
	"context"
	"errors"
	"io"
	"net/http"
	"sync"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/multierr"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/httpcheckreceiver/internal/metadata"
)

var (
	errClientNotInit    = errors.New("client not initialized")
	httpResponseClasses = map[string]int{"1xx": 1, "2xx": 2, "3xx": 3, "4xx": 4, "5xx": 5}
)

type httpcheckScraper struct {
	clients  []*http.Client
	cfg      *Config
	settings component.TelemetrySettings
	version  string
	mb       *metadata.MetricsBuilder
	logs     consumer.Logs
}

// start starts the scraper by creating a new HTTP Client on the scraper
func (h *httpcheckScraper) start(ctx context.Context, host component.Host) (err error) {
	for _, target := range h.cfg.Targets {
		client, clentErr := target.ToClient(ctx, host, h.settings)
		if clentErr != nil {
			err = multierr.Append(err, clentErr)
		}
		h.clients = append(h.clients, client)
	}
	return
}

// scrape connects to the endpoint and produces metrics based on the response
func (h *httpcheckScraper) scrape(ctx context.Context) (pmetric.Metrics, error) {
	if h.clients == nil || len(h.clients) == 0 {
		return pmetric.NewMetrics(), errClientNotInit
	}

	var wg sync.WaitGroup
	wg.Add(len(h.clients))
	var mux sync.Mutex

	for idx, client := range h.clients {
		go func(targetClient *http.Client, targetIndex int) {
			defer wg.Done()

			now := pcommon.NewTimestampFromTime(time.Now())

			req, err := http.NewRequestWithContext(ctx, h.cfg.Targets[targetIndex].Method, h.cfg.Targets[targetIndex].Endpoint, http.NoBody)
			if err != nil {
				h.settings.Logger.Error("failed to create request", zap.Error(err))
				return
			}

			start := time.Now()
			resp, err := targetClient.Do(req)
			duration := time.Since(start)
			mux.Lock()
			h.mb.RecordHttpcheckDurationDataPoint(now, time.Since(start).Milliseconds(), h.cfg.Targets[targetIndex].Endpoint)

			statusCode := 0
			if err != nil {
				h.mb.RecordHttpcheckErrorDataPoint(now, int64(1), h.cfg.Targets[targetIndex].Endpoint, err.Error())
			} else {
				statusCode = resp.StatusCode
			}

			for class, intVal := range httpResponseClasses {
				if statusCode/100 == intVal {
					h.mb.RecordHttpcheckStatusDataPoint(now, int64(1), h.cfg.Targets[targetIndex].Endpoint, int64(statusCode), req.Method, class)
				} else {
					h.mb.RecordHttpcheckStatusDataPoint(now, int64(0), h.cfg.Targets[targetIndex].Endpoint, int64(statusCode), req.Method, class)
				}
			}
			mux.Unlock()

			// Report logging *outside* the lock.
			if statusCode != 0 {
				err = h.logResponse(ctx, h.cfg.Targets[targetIndex].Endpoint, resp, now, duration)
				if err != nil {
					h.settings.Logger.Error("failed to log response", zap.Error(err))
				}

			}
		}(client, idx)
	}

	wg.Wait()

	return h.mb.Emit(), nil
}

func (h *httpcheckScraper) logResponse(
	ctx context.Context,
	endpoint string,
	resp *http.Response,
	timestamp pcommon.Timestamp,
	elapsed time.Duration) error {

	// Return if logs reporting was NOT enabled.
	if h.logs == nil {
		return nil
	}

	defer resp.Body.Close()
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	out := plog.NewLogs()
	logs := out.ResourceLogs()
	rls := logs.AppendEmpty()

	ills := rls.ScopeLogs().AppendEmpty()
	ills.Scope().SetName("otelcol/httpcheckreceiver")
	ills.Scope().SetVersion(h.version)

	lr := ills.LogRecords().AppendEmpty()
	lr.SetTimestamp(timestamp)
	lr.Body().SetStr(string(bodyBytes))
	attrs := lr.Attributes()
	attrs.PutStr("http.url", endpoint)
	attrs.PutInt("http.response.status_code", int64(resp.StatusCode))
	attrs.PutInt("http.client.request.duration", int64(elapsed))

	return h.logs.ConsumeLogs(ctx, out)
}

func newScraper(conf *Config, settings receiver.CreateSettings) *httpcheckScraper {
	return &httpcheckScraper{
		cfg:      conf,
		settings: settings.TelemetrySettings,
		version:  settings.BuildInfo.Version,
		mb:       metadata.NewMetricsBuilder(conf.MetricsBuilderConfig, settings),
	}
}
