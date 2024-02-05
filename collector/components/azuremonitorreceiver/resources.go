// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package azuremonitorreceiver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/azuremonitorreceiver"

import (
	"context"
	"time"

	// corepolicy "github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"

	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/azuremonitorreceiver/internal/azuresdk"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/azuremonitorreceiver/internal/resourcestore"
)

type resourceManager struct {
	config 					*Config
	// resourcesClientFactory holds some base configuration for resourcesClient and resourceGroupsClient
	clientFactory 			*armresources.ClientFactory
	// resourcesClient can get resources by subscriptionID
	resourcesClient 		*armresources.Client
	// resourceGroupsClient can get resource groups by subscriptionID
	resourceGroupsClient 	*armresources.ResourceGroupsClient
	// resourceStore is a typesafe wrapper on sync.Map for storing resources in batches by resource type
	resourceStore 			*resourcestore.ResourceStore

	updateInterval 			time.Duration
}

func newResourceManager(conf *Config) *resourceManager {
	return &resourceManager{
		config: conf,
	}
}

func (r *resourceManager) start(ctx context.Context) error {
	// TODO: set the ClientListOptions based upon config, as of now we pass a typed nil
	// so we accept the defaults
	opts := &policy.ClientOptions{}
	if cred, err := r.config.getAzureCredential(); err != nil {
		return err
	} else {
		clientFactory, err := armresources.NewClientFactory(r.config.SubscriptionID, cred, opts)
		if err != nil {
			return err
		}
		r.clientFactory = clientFactory
	}
	r.resourcesClient = r.clientFactory.NewClient()
	r.resourceGroupsClient = r.clientFactory.NewResourceGroupsClient()
	r.resourceStore = resourcestore.NewResourceStore()
	// TODO: invoke the controller that will do the resource scrape
	r.runUpdater(ctx)

	return nil
}



func (r *resourceManager) updateResourcesCache(ctx context.Context) error {
	// keep a copy of keys in the map
	// currentKeys := r.currentKeys()

	// TODO: set the ClientListOptions based upon config
	opts := &armresources.ClientListOptions{}
	ctxFunc := func() context.Context {
		// TODO: maybe request timeouts are different, maybe they're just in the clients and that's
		// good enough for KISS
		return ctx
	}
	err := azuresdk.FanoutPages(
		ctxFunc,
		r.resourcesClient.NewListPager(opts),
		func(ctx context.Context, page armresources.ClientListResponse) (err error) {
			if _, err := r.resourceStore.StoreResources(page.Value); err != nil {
				return err
			}
			return nil
		},
	)
	if err != nil {
		return err
	}
	return nil
}

/*
func (r *resourceManager) currentKeys() keyTree {
	var currentKeys keyTree
	r.resourceStore.Range(func(resourceType string, rb *resourcestore.ResourceBatch) bool {
		if rs, ok := rb.GetResources(); !ok {
			return true
		} else {
			for key, _ := range rs {
				currentKeys.add(resourceType, key)
			}
		}
		return true
	})
	return currentKeys
}
*/

// keyTree is a map of resource type to map of resource ID to struct{}
// it's only child keys, so it's not a full tree
type keyTree map[string]map[string]struct{}

// add a child key to a root key
func (kt keyTree) add(rootKey, childKey string) {
	if _, ok := kt[rootKey]; !ok {
		kt[rootKey] = make(map[string]struct{})
	}
	kt[rootKey][childKey] = struct{}{}
}

// delete a child key from a root key
func (kt keyTree) delete(rootKey, childKey string) {
	if _, ok := kt[rootKey]; !ok {
		return
	}
	delete(kt[rootKey], childKey)
}


