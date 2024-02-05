// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package azuremonitorreceiver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/azuremonitorreceiver"

import (
	"context"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/monitor/azquery"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/azuremonitorreceiver/internal/azuresdk"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/azuremonitorreceiver/internal/metadata"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/azuremonitorreceiver/internal/resourcestore"
)

var (
	timeGrains = map[string]int64{
		"PT1M":  60,
		"PT5M":  300,
		"PT15M": 900,
		"PT30M": 1800,
		"PT1H":  3600,
		"PT6H":  21600,
		"PT12H": 43200,
		"P1D":   86400,
	}
)

var (
	metadataPrefix = "metadata_"
)

// TODO: put this in Config, verify the max allowable in the SDK,
// and possibly make sure that config can't exceed the SDK constraint.
const maxResourcesPerCall = 5

type azureScraper struct {
	cfg                 *Config
	settings            component.TelemetrySettings
	mb                  *metadata.MetricsBuilder

	metricsManager *metricsManager
	resourceManager *resourceManager
}

func newAzureScraper(conf *Config, settings receiver.CreateSettings) *azureScraper {
	// TODO: make opts for client factory based on config
	resourceManager := newResourceManager(conf)
	return &azureScraper{
		cfg:                 conf,
		settings:            settings.TelemetrySettings,
		mb:                  metadata.NewMetricsBuilder(conf.MetricsBuilderConfig, settings),
		resourceManager: 	 resourceManager,
	}
}

// resourceManager is a goroutine that periodically checks the TTL of resource metadata and updates it when necessary.
func (r *resourceManager) runUpdater(ctx context.Context) {
	// TODO: convert this to take parseable duration from config
	d := time.Duration(r.updateInterval) * time.Second
	ticker := time.NewTicker(d)
	defer ticker.Stop()
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				r.updateResourcesCache(ctx)
			}
		}
	}()
}

func (s *azureScraper) start(ctx context.Context, _ component.Host) error {
	// initialize the resource metadata cache and start the updater
	if err := s.resourceManager.start(ctx); err != nil {
		return err
	}
	// s.resourceManager.start(ctx)
	if err := s.metricsManager.start(ctx); err != nil {
		return err
	}
	return nil
}

// scrape will get batches which are by resource type, it will further batch them by other constraints (e.g. time grain),
// and then it will limit by API batch request size. We may need filters for time grain.
func (s *azureScraper) scrape(ctx context.Context) (pmetric.Metrics, error) {
	// TODO: make opts based on config
	wg := sync.WaitGroup{}
	// 1. we want to range over batches keyed by resource type
	s.resourceManager.resourceStore.Range(
		func(resourceType string, rb *resourcestore.ResourceBatch) bool {
			vals, err := s.metricsManager.fetchMetricValues(ctx, rb)
			if err != nil {
				s.settings.Logger.Error("failed to get Azure Metrics data", zap.Error(err))
			}

			for _, v := range vals {
				mdefs, ok := rb.Definitions()[*v.Namespace]
				if !ok {
					s.settings.Logger.Error("metric definitions unavailable for namespace: " + *v.Namespace)
				}
				for _, m := range v.Values {
					d, ok := mdefs[*m.Name.Value]
					if !ok {
						s.settings.Logger.Error("metrics definition unavailable: " + *m.Name.Value)
					}
					s.processMetric(ctx, d, m)
				}
			}
			return true
		},
	)
	wg.Wait()
	// TODO: add back the semconv resource attributes here
	return s.mb.Emit(nil), nil
}

func (s *azureScraper) processMetric(ctx context.Context, d *azquery.MetricDefinition, m *azquery.Metric) {
	attrs := map[string]*string{}
	for _, elem := range m.TimeSeries {
		if elem.Data != nil {
			for _, value := range elem.MetadataValues {
				name := metadataPrefix + *value.Name.Value
				attrs[name] = value.Value
			}
			for _, value := range elem.MetadataValues {
				name := metadataPrefix + *value.Name.Value
				attrs[name] = value.Value
			}

			// TODO: add other resource attrs per semconv
			attrs["resource_id"] = d.ResourceID
			attrs["resource_name"] = d.Name.Value

			// TODO: add tags as attributes
			for _, metricValue := range elem.Data {
				// TODO: look at rewriting this function for something that fits better
				// with how it's used in this context.
				aggValues := azuresdk.GetAggregationValues(d, metricValue)
				// add a data point for each aggregation.
				// This could look like metric_name_max, metric_name_min, etc.
				for aggName, val := range aggValues {
					s.mb.AddDataPoint(
						*m.ID,
						*m.Name.Value,
						aggName,
						string(*m.Unit),
						attrs,
						// TODO: confirm that Azure SDK always sets .TimeStamp in metricValue
						// and that it's set to an expected value for context.
						pcommon.NewTimestampFromTime(*metricValue.TimeStamp),
						*val,
					)
				}
			}
		}
	}
}
