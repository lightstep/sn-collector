// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package azuremonitorreceiver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/azuremonitorreceiver"

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/monitor/azquery"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
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

type void struct{}

type azureResourceData struct {
	resource          *armresources.GenericResourceExpanded
	metricDefinitions map[string]*azquery.MetricDefinition
	// TODO: could we refactor to make these getters, so we don't need to precompute?
	attributes map[string]*string
	tags       map[string]*string
}

func newScraper(conf *Config, settings receiver.CreateSettings) *azureScraper {
	return &azureScraper{
		cfg:                 conf,
		settings:            settings.TelemetrySettings,
		mb:                  metadata.NewMetricsBuilder(conf.MetricsBuilderConfig, settings),
		azIDCredentialsFunc: azidentity.NewClientSecretCredential,
		azIDWorkloadFunc:    azidentity.NewWorkloadIdentityCredential,
		resources:           resourcestore.NewResourceStore(),
	}
}

type azureScraper struct {
	cred                azcore.TokenCredential
	cfg                 *Config
	settings            component.TelemetrySettings
	mb                  *metadata.MetricsBuilder
	azIDCredentialsFunc func(string, string, string, *azidentity.ClientSecretCredentialOptions) (*azidentity.ClientSecretCredential, error)
	azIDWorkloadFunc    func(options *azidentity.WorkloadIdentityCredentialOptions) (*azidentity.WorkloadIdentityCredential, error)
	// metricsClient can get metric definitions and metric values by resourceID
	metricsClient *azquery.MetricsClient
	// metricsBatchClient is currently only in azquery v1.2.0-beta.1. It provides compelling value over alternatives for the Collector use case.
	// I believe previous versions of the SDK provide this functionality, but you need to build the batch client yourself.
	metricsBatchClient *azquery.MetricsBatchClient
	// resourcesClientFactory holds some base configuration for resourcesClient and resourceGroupsClient
	resourcesClient *armresources.Client
	// we only need to use the resource groups client to list resource groups when the client configures the receiver to do so
	resourceGroupsClient *armresources.ResourceGroupsClient
	// resources is a map[resourceType]BatchResourceData{
	// 		resources   map[resourceID]*armresources.GenericResourceExpanded
	// 		definitions map[string]*azquery.MetricDefinition
	// }
	resources *resourcestore.ResourceStore
	// underneath we treat this as a map[resourceType][]*azureResourceData
}

// TODO: see what the race detector wants to protect the resources map

// resourceUpdater is a goroutine that periodically checks the TTL of resource metadata and updates it when necessary.
func (s *azureScraper) runResourcesCacheUpdater(ctx context.Context) {
	// convert float64 to time.Duration
	d := time.Duration(s.cfg.CacheResources) * time.Second
	ticker := time.NewTicker(d)
	defer ticker.Stop()
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.updateResourcesCache(ctx)
			}
		}
	}()
}

// update
// 1. fetch the resources
// 		send a map of IDs (a set) for updating
// 		fetch defintions for each resource (if we don't have them)
// 2. store the resources we fetch

// TODO: store resources by type
func (s *azureScraper) updateResourcesCache(ctx context.Context) {
	// keep a copy of keys in the map
	var currentKeys map[string]map[string]struct{}
	s.resources.Range(func(resourceType any, value any) bool {
		for key := range value.(map[string]*azureResourceData) {
			if _, ok := currentKeys[resourceType.(string)]; !ok {
				currentKeys[resourceType.(string)] = make(map[string]struct{})
			}
			currentKeys[resourceType.(string)][key] = struct{}{}
		}
		return true
	})
	rs, err := s.fetchResources(ctx)
	if err != nil {
		s.settings.Logger.Error("failed to get Azure Resources data", zap.Error(err))
		return
	}
	for _, r := range rs {
		s.resources.Store(r)
	}
	var wg sync.WaitGroup
	for _, r := range rs {
		wg.Add(1)
		go func(r *armresources.GenericResourceExpanded) {
			defer wg.Done()
			defs, err := s.fetchDefinitions(ctx, *r.ID)
			if err != nil {
				s.settings.Logger.Error("failed to get Azure Metrics definitions data", zap.Error(err))
				return
			}
			attrs, err := azuresdk.CalculateResourceAttributes(r)
			if err != nil {
				s.settings.Logger.Error("failed to calculate Azure Resource attributes", zap.Error(err))
				return
			}
			resourceType := *r.Type
			rt, ok := s.resources.Load(resourceType)
			if !ok {
				s.resources.Store(resourceType, make(map[string]*azureResourceData))
			}
			// NOTE: will this work for a sync.Map? I think so, but need to test.
			rt.(map[string]*azureResourceData)[*r.ID] = &azureResourceData{
				resource:          r,
				metricDefinitions: defs,
				attributes:        attrs,
				tags:              r.Tags,
			}
			delete(currentKeys, *r.ID)
		}(r)
	}
	wg.Wait()
	for key := range currentKeys {
		s.resources.Delete(key)
	}
}

func (s *azureScraper) fetchResources(ctx context.Context) ([]*armresources.GenericResourceExpanded, error) {
	// TODO: make opts based on config
	// filter := azuresdk.CalculateResourcesFilter(s.cfg.Services, s.cfg.ResourceGroups)
	// opts := &armresources.ClientListOptions{
	// 	Filter: to.Ptr(""),
	// }
	var results []*armresources.GenericResourceExpanded
	err := azuresdk.ProcessPager(
		ctx,
		s.resourcesClient.NewListPager(nil),
		// TODO: use our Extractor in the AzureSDK package
		func(page armresources.ClientListResponse) []*armresources.GenericResourceExpanded {
			return page.Value
		},
		func(ctx context.Context, resource *armresources.GenericResourceExpanded) error {
			results = append(results, resource)
			return nil
		},
	)
	return nil, err
}

func (s *azureScraper) fetchDefinitions(ctx context.Context, rid string) (map[string]*azquery.MetricDefinition, error) {
	var defs map[string]*azquery.MetricDefinition
	// TODO: make opts based on config
	opts := &azquery.MetricsClientListDefinitionsOptions{}

	err := azuresdk.ProcessPager(
		ctx,
		s.metricsClient.NewListDefinitionsPager(rid, opts),
		func(page azquery.MetricsClientListDefinitionsResponse) []*azquery.MetricDefinition {
			return page.Value
		},
		func(ctx context.Context, v *azquery.MetricDefinition) error {
			defs[*v.ID] = v
			return nil
		},
	)
	if err != nil {
		return nil, err
	}
	return defs, nil
}

func (s *azureScraper) instantiateClients() (err error) {
	if err = s.loadCredentials(); err != nil {
		return
	}

	// TODO: calculate opts based on config - configure retry, logging, telemetry, etc
	opts := &policy.ClientOptions{}
	clientFactory, err := armresources.NewClientFactory(s.cfg.SubscriptionID, s.cred, opts)
	if err != nil {
		return
	}

	s.resourcesClient = clientFactory.NewClient()
	s.resourceGroupsClient = clientFactory.NewResourceGroupsClient()

	// TODO: calculate opts based on config - configure retry, logging, telemetry, etc
	metricsOptions := &azquery.MetricsClientOptions{}
	if s.metricsClient, err = azquery.NewMetricsClient(s.cred, metricsOptions); err != nil {
		s.settings.Logger.Error("failed to create Azure Metrics client", zap.Error(err))
		return
	}

	endpoint := fmt.Sprintf("https://%s.metrics.monitor.azure.com", s.cfg.Region)
	// TODO: calculate opts based on config - configure retry, logging, telemetry, etc
	metricsBatchOptions := &azquery.MetricsBatchClientOptions{}
	if s.metricsBatchClient, err = azquery.NewMetricsBatchClient(endpoint, s.cred, metricsBatchOptions); err != nil {
		s.settings.Logger.Error("failed to create Azure Metrics Batch client", zap.Error(err))
		return
	}

}

func (s *azureScraper) start(ctx context.Context, _ component.Host) (err error) {
	if err = s.instantiateClients(); err != nil {
		return
	}
	// initialize the resource metadata cache and start the updater
	s.updateResourcesCache(ctx)
	s.runResourcesCacheUpdater(ctx)

	return
}

func (s *azureScraper) loadCredentials() (err error) {
	switch s.cfg.Authentication {
	case servicePrincipal:
		if s.cred, err = s.azIDCredentialsFunc(s.cfg.TenantID, s.cfg.ClientID, s.cfg.ClientSecret, nil); err != nil {
			return err
		}
	case workloadIdentity:
		if s.cred, err = s.azIDWorkloadFunc(nil); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown authentication %v", s.cfg.Authentication)
	}
	return nil
}

// scrape will get batches which are by resource type, it will further batch them by other constraints (e.g. time grain),
// and then it will limit by API batch request size. We may need filters for time grain.
func (s *azureScraper) scrape(ctx context.Context) (pmetric.Metrics, error) {
	// TODO: make opts based on config
	opts := &azquery.MetricsClientQueryResourceOptions{}
	wg := sync.WaitGroup{}
	s.resources.Range(func(key any, value interface{}) bool {
		v, ok := value.(azureResourceData)
		if !ok {
			s.settings.Logger.Error("type assertion to azureResourceData")
			return true
		}
		k := key.(string)
		wg.Add(1)
		go func(k string, v azureResourceData) {
			defer wg.Done()
			metrics, err := s.metricsClient.QueryResource(ctx, k, opts)
			// if the operation fails, it returns *azcore.ResponseError
			if err != nil {
				s.settings.Logger.Error("failed to get Azure Metrics data", zap.Error(err))
				return
			}
			for _, m := range metrics.Value {
				d, ok := v.metricDefinitions[*m.ID]
				if !ok {
					s.settings.Logger.Error("metrics definition unavailable " + *m.ID)
					return
				}
				s.processMetric(ctx, d, m)
			}
		}(k, v)
		return true
	})
	wg.Wait()
	// TODO: add resource metrics options where we will include semconv
	return s.mb.Emit(nil), nil
}

var (
	metadataPrefix = "metadata_"
)

// func (s *ResourceStore) Keys() []string {
// 	var keys []string
// 	s.sm.Range(func(key, value interface{}) bool {
// 		keys = append(keys, key.(string))
// 		return true
// 	})
// 	return keys
// }

// TODO: put this in Config, verify the max allowable in the SDK,
// and possibly make sure that config can't exceed the SDK constraint.
const maxResourcesPerCall = 5

// The use case for batching describes the Collector's case perfectly:
// 		https://learn.microsoft.com/en-us/azure/azure-monitor/essentials/migrate-to-batch-api?tabs=individual-response#paging

// for each batch, compose a batch metrics call
// 		- all resources in a batch must be in the same subscription.

func (s *azureScraper) getBatchMetricsValues(ctx context.Context) {
	// the outer function is to range to get all resources in a type from the untyped sync.Map
	// 1. fetch the batches of resources
	// 2. handle the batch of resources
	// var resType map[string][]azureResourceData
	s.resources.Range(func(resourceType any, value any) bool {
		// TODO: get this into a function that returns this type. We don't
		// 		want to return from the Range function, so we need to do this
		resourceBatches, ok := value.(map[string]*resourcestore.ResourceBatch)
		if !ok {
			s.settings.Logger.Error("type assertion to azureResourceData")
			return true
		}
		return true
	})
	for compositeKey, metricsByGrain := range resType.metricsByCompositeKey {
		if time.Since(metricsByGrain.metricsValuesUpdated).Seconds() < float64(timeGrains[compositeKey.timeGrain]) {
			continue
		}
		now := time.Now().UTC()
		metricsByGrain.metricsValuesUpdated = now
		startTime := now.Add(time.Duration(-timeGrains[compositeKey.timeGrain]) * time.Second)
		s.settings.Logger.Info("getBatchMetricsValues", zap.String("resourceType", resourceType), zap.Any("metricNames", metricsByGrain.metrics), zap.Any("startTime", startTime), zap.Any("now", now), zap.String("timeGrain", compositeKey.timeGrain))

		start := 0
		for start < len(metricsByGrain.metrics) {

			end := start + s.cfg.MaximumNumberOfMetricsInACall
			if end > len(metricsByGrain.metrics) {
				end = len(metricsByGrain.metrics)
			}

			// QueryBatch in the SDK uses https://learn.microsoft.com/en-us/azure/azure-monitor/essentials/migrate-to-batch-api
			// So per docs, we'll need to consider the following restrictions when constructing calls:
			// 		- All resources in a batch must be in the same subscription.
			// 		- All resources in a batch must be in the same Azure region.
			// 		- All resources in a batch must be the same resource type.

			// res, err := metricsBatchClient.QueryBatch(
			// 	context.Background(),
			// 	subscriptionID,
			// 	"Microsoft.Storage/storageAccounts",
			// 	[]string{"Ingress"},
			// 	azquery.ResourceIDList{ResourceIDs: to.SliceOfPtrs(resourceURI1, resourceURI2)},
			// 	&azquery.MetricsBatchClientQueryBatchOptions{
			// 		Aggregation: to.SliceOfPtrs(azquery.AggregationTypeAverage),
			// 		StartTime:   to.Ptr("2023-11-15"),
			// 		EndTime:     to.Ptr("2023-11-16"),
			// 		Interval:    to.Ptr("PT5M"),
			// 	},
			// )
			var resourceIDs []string
			for _, res := range resType {
				resourceIDs = append(resourceIDs, *res.resource.ID)
			}
			// TODO: let's test how much we can batch.
			response, err := s.metricsBatchClient.QueryBatch(
				ctx,
				s.cfg.SubscriptionID,
				resourceType,
				// metricsByGrain.metrics[start:end],
				// let's just build a couple of calls for resources

				&azquery.ResourceIDList{ResourceIDs: resType.resourceIds},
				// azquery.ResourceIDList{ResourceIDs: resType.resourceIds},
				&azquery.MetricsBatchClientQueryBatchOptions{
					Aggregation: to.SliceOfPtrs(
						azquery.AggregationTypeAverage,
						azquery.AggregationTypeMaximum,
						azquery.AggregationTypeMinimum,
						azquery.AggregationTypeTotal,
						azquery.AggregationTypeCount,
					),
					StartTime: to.Ptr(startTime.Format(time.RFC3339)),
					EndTime:   to.Ptr(now.Format(time.RFC3339)),
					Interval:  to.Ptr(compositeKey.timeGrain),
					// Top:       to.Ptr(int32(s.cfg.MaximumNumberOfDimensionsInACall)), // Defaults to 10 (may be limiting results)
				},
			)

			if err != nil {
				s.settings.Logger.Error("failed to get Azure Metrics values data", zap.Error(err))
				return
			}

			start = end
			for _, metricValues := range response.Values {
				for _, metric := range metricValues.Values {
					for _, timeseriesElement := range metric.TimeSeries {

						if timeseriesElement.Data != nil {
							res := s.resources[*metricValues.ResourceID]
							attributes := map[string]*string{}
							for name, value := range res.attributes {
								attributes[name] = value
							}
							for _, value := range timeseriesElement.MetadataValues {
								name := metadataPrefix + *value.Name.Value
								attributes[name] = value.Value
							}
							if s.cfg.AppendTagsAsAttributes {
								for tagName, value := range res.tags {
									name := tagPrefix + tagName
									attributes[name] = value
								}
							}
							for _, metricValue := range timeseriesElement.Data {
								s.processQueryTimeseriesData(*metricValues.ResourceID, metric, metricValue, attributes)
							}
						}
					}
				}
			}
		}
	}
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
			// TODO: add tags as attributes
			// if s.cfg.AppendTagsAsAttributes {
			// 	name := tagPrefix + tagName
			// 	attrs[name] = value
			// }
			for _, metricValue := range elem.Data {
				aggValues := azuresdk.GetAggregationValues(d, metricValue)
				for aggName, aggValue := range aggValues {
					s.mb.AddDataPoint(
						*m.ID,
						*m.Name.Value,
						aggName,
						string(*m.Unit),
						attrs,
						// TODO: confirm that Azure SDK always sets .TimeStamp
						pcommon.NewTimestampFromTime(*metricValue.TimeStamp),
						*aggValue,
					)
				}
			}
		}
	}
}
