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
		resourceMetadata: sync.Map{},
	}
}

type azureScraper struct {
	cred azcore.TokenCredential
	cfg                    *Config
	settings               component.TelemetrySettings
	mb                     *metadata.MetricsBuilder
	azIDCredentialsFunc    func(string, string, string, *azidentity.ClientSecretCredentialOptions) (*azidentity.ClientSecretCredential, error)
	azIDWorkloadFunc       func(options *azidentity.WorkloadIdentityCredentialOptions) (*azidentity.WorkloadIdentityCredential, error)
	// metricsClient can get metric definitions and metric values by resourceID
	metricsClient          *azquery.MetricsClient
	// resourcesClientFactory holds some base configuration for resourcesClient and resourceGroupsClient
	resourcesClientFactory *armresources.ClientFactory
	resourcesClient        *armresources.Client
	// we only need to use the resource groups client to list resource groups when the client configures the receiver to do so
	resourceGroupsClient   *armresources.ResourceGroupsClient
	// resourceMetadata holds the resource structs and their related metric definitions
	resourceMetadata       sync.Map
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

func (s *azureScraper) updateResourcesCache(ctx context.Context) {
	// keep a copy of keys in the map
	var currentKeys map[string]void
	s.resourceMetadata.Range(func(key any, value any) bool {
		currentKeys[key.(string)] = struct{}{}
		return true
	})
	rs, err := s.fetchResources(ctx)
	if err != nil {
		s.settings.Logger.Error("failed to get Azure Resources data", zap.Error(err))
		return
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
			s.resourceMetadata.Store(*r.ID, &azureResourceData{
				resource:          r,
				metricDefinitions: defs,
				attributes:        attrs,
				tags:              r.Tags,
			})
			delete(currentKeys, *r.ID)
		}(r)
	}
	wg.Wait()
	for key := range currentKeys {
		s.resourceMetadata.Delete(key)
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

func (s *azureScraper) start(ctx context.Context, _ component.Host) (err error) {
	if err = s.loadCredentials(); err != nil {
		return err
	}
	// TODO: calculate opts based on config
	opts := &policy.ClientOptions{}
	s.resourcesClientFactory, err = armresources.NewClientFactory(s.cfg.SubscriptionID, s.cred, opts)
	// begin by initalizing
	s.updateResourcesCache(ctx)
	// set to run periodically based on config
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

func (s *azureScraper) scrape(ctx context.Context) (pmetric.Metrics, error) {
	// TODO: make opts based on config
	opts := &azquery.MetricsClientQueryResourceOptions{}
	wg := sync.WaitGroup{}
	s.resourceMetadata.Range(func(key any, value interface{}) bool {
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
