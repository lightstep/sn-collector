// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package azuremonitorreceiver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/azuremonitorreceiver"

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/monitor/azquery"

	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/azuremonitorreceiver/internal/azuresdk"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/azuremonitorreceiver/internal/resourcestore"
)

type metricsManager struct {
	conf *Config
	// metricsClient can provide a NewListDefinitionsPager for a resourceURI and metric values
	// via the QueryResource method
	metricsClient *azquery.MetricsClient
	// metricsBatchClient can get metrics by resourceID
	metricsBatchClient *azquery.MetricsBatchClient
}

func newMetricsManager(conf *Config) *metricsManager {
	return &metricsManager{
		conf: conf,
	}
}

func (m *metricsManager) start(ctx context.Context) error {
	cred, err := m.conf.getAzureCredential()
	if err != nil {
		return err
	}
	// TODO: calculate opts based on config - configure retry, logging, telemetry, etc
	metricsOptions := &azquery.MetricsClientOptions{}
	metricsClient, err := azquery.NewMetricsClient(cred, metricsOptions)
	if err != nil {
		return err
	}

	// TODO: calculate opts based on config - configure retry, logging, telemetry, etc
	endpoint := fmt.Sprintf("https://%s.metrics.monitor.azure.com", m.conf.Region)
	metricsBatchOptions := &azquery.MetricsBatchClientOptions{}
	metricsBatchClient, err := azquery.NewMetricsBatchClient(endpoint, cred, metricsBatchOptions)
	if err != nil {
		return err
	}
	m.metricsClient = metricsClient
	m.metricsBatchClient = metricsBatchClient
	return nil
}

// fetchMetricsByResource fetches metrics for a specific resource identified by its resource ID.
// It takes a context.Context and a resourceID string as input parameters.
// It returns a slice of pointers to azquery.Metric and an error.
// The function calculates options based on the configuration to configure retry, logging, telemetry, etc.
func (m *metricsManager) fetchMetricsByResource(ctx context.Context, resourceID string) ([]*azquery.Metric, error) {
	// TODO: calculate opts based on config - configure retry, logging, telemetry, etc
	opts := &azquery.MetricsClientQueryResourceOptions{}
	ms, err := m.metricsClient.QueryResource(ctx, resourceID, opts)
	if err != nil {
		return nil, err
	}
	return ms.Value, nil
}

// fetchDefinitions fetches metric definitions for a specific resource identified by its resource ID.
func (m *metricsManager) fetchDefinitions(ctx context.Context, resourceID string) (map[string]*azquery.MetricDefinition, error) {
	// TODO: calculate opts based on config - configure retry, logging, telemetry, etc
	opts := &azquery.MetricsClientListDefinitionsOptions{}
	pager := m.metricsClient.NewListDefinitionsPager(resourceID, opts)
	mdMap := make(map[string]*azquery.MetricDefinition)
	if err := azuresdk.ProcessItems(
		ctx,
		pager,
		azuresdk.ExtractMetricDefinitions,
		func(_ context.Context, md *azquery.MetricDefinition) error {
			mdMap[*md.Name.Value] = md
			return nil
		},
	); err != nil {
		return nil, err
	}
	return mdMap, nil
}

type namesAndResources struct {
	names    	[]string
	resources 	[]string
}

func (m *metricsManager) fetchMetricValues(ctx context.Context, rb *resourcestore.ResourceBatch) ([]*azquery.MetricValues, error) {
	var (
		mvs = []*azquery.MetricValues{}
		mu = &sync.Mutex{}
	)
	errs := make([]error, 0)
	resourceIDList := azquery.ResourceIDList{ResourceIDs: to.SliceOfPtrs(rb.ResourceIDs()...)}
	wg := &sync.WaitGroup{}
	// TODO: calculate opts based on config - configure retry, logging, telemetry, etc
	opts := &azquery.MetricsBatchClientQueryBatchOptions{}
	for _, namespace := range rb.Namespaces() {
		// build a batch and send it in a goroutine
		wg.Add(1)
		go func() {
			defer wg.Done()

			metricNames, err := rb.NamespaceMetricNames(namespace)
			if err != nil {
				errs = append(errs, err)
				return
			}

			resp, err := m.metricsBatchClient.QueryBatch(ctx, m.conf.SubscriptionID, namespace, metricNames, resourceIDList, opts)
			if err != nil {
				errs = append(errs, err)
				return
			}
			mu.Lock()
			defer mu.Unlock()
			mvs = append(mvs, resp.Values...)
		}()
		wg.Wait()
	}
	return mvs, errors.Join(errs...)
}
