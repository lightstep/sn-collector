// Package resourcestore provides a threadsafe store for Azure resources and their metadata.
// It's intended to work with the Azure Monitor batch API.
//
// Based on how the batch API works, think the essence of the data structure we want
// is something like this:
//
//	map[resourceType]struct{
//		metricsDefinitions map[metricID]metricDefinition
//		resources map[resourceID]ResourceBatch
//	}
//
// But the implementation uses sync.Map for the maps. The risk of dirty reads between starting
// the resource metadata fetch and evicting old key with DeleteUnusedResources is that we could
// fail a batch request for including a resource that is no longer in the subscription.
//
// Note also that the batch API does not presently include custom metrics. This means that
// we can store one set of metric definitions for all resources of a type. Otherwise, we
// have to use the non-batch API for custom metrics, which means we have to make a request
// per resource.
//
// See https://learn.microsoft.com/en-us/azure/azure-monitor/essentials/migrate-to-batch-api?tabs=individual-response#troubleshooting
package resourcestore

// TODO: consider making ResourceType into a type. Seems we know the strings at this time.
// And/Or make ResourceID into a type, so we can guarantee it's a valid resource ID after
// we build it.

import (
	"errors"
	// "strings"
	"sync"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/monitor/azquery"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
)

// ResourceStore is a typesafe wrapper on sync.Map
type ResourceStore struct {
	// think of the underlying type as map[resourceType]ResourceBatch
	m sync.Map
}

// StoreDefinitions(ds []*azquery.MetricDefinition)
// GetDefinitions(resourceType string) (map[string]*azquery.MetricDefinition, bool)
// StoreResources(rs []*armresources.GenericResourceExpanded)
// GetResources(resourceType string) (map[string]*armresources.GenericResourceExpanded, bool)

// NewResourceStore creates a new ResourceStore.
func NewResourceStore() *ResourceStore {
	return &ResourceStore{}
}

func (rs *ResourceStore) getDefinitions(resourceID string) (map[string]*azquery.MetricDefinition, error) {
	resType, err := arm.ParseResourceType(resourceID)
	if err != nil {
		return nil, err
	}
	v, ok := rs.m.Load(resType.String())
	if !ok {
		return nil, errors.New("resource type not found")
	}
	// NOTE: we can return a copy of the map, but we don't need to for a couple reasons:
	rb, ok := v.(*ResourceBatch)
	if !ok {
		return nil, errors.New("error asserting to resource batch")
	}
	resDefs, ok := rb.getDefinitionsByResource(resourceID)
	if !ok {
		return nil, errors.New("resource not found")
	}
	return resDefs, nil
}

func (rs *ResourceStore) StoreDefinitions(ds []*azquery.MetricDefinition) error {
    // avoids possible panic on index out of range and nil dereference
	if len(ds) == 0 {
		return errors.New("no definitions to store")
	}
    if ds[0].ResourceID == nil {
        return errors.New("first definition resource ID is nil")
    }

	resID := *ds[0].ResourceID
	resType, err := arm.ParseResourceType(resID)
	if err != nil {
		return err
	}
	// 1. load value or create one
    v, _ := rs.m.LoadOrStore(resType.String(), newResourceBatch(resType.String()))
	// we could do something to check this, but it's not necessary
	// 2. type assert the value to a ResourceBatch
    rb, ok := v.(*ResourceBatch)
    if !ok {
        return errors.New("error asserting to resource batch")
    }
	// 3. store the definitions in the ResourceBatch
    if err := rb.StoreDefinitions(resID, ds); err != nil {
        return err
    }
	// 4. store the ResourceBatch in the ResourceStore_sync.Map
	rs.m.Store(resType.String(), rb)
    return nil
}

func (rs *ResourceStore) storeResources(resourceOrType string, rss []*armresources.GenericResourceExpanded) (set, error) {
	resType, err := arm.ParseResourceType(resourceOrType)
	if err != nil {
		return nil, err
	}

    v, _ := rs.m.LoadOrStore(resType.String(), newResourceBatch(resType.String()))

    rb, ok := v.(*ResourceBatch)
    if !ok {
        return nil, errors.New("error asserting to resource batch")
    }

	// store the resources in the ResourceBatch
    nf, err := rb.storeResources(rss)
	if err != nil {
		return nil, err
	}
	// store the batch in the ResourceStore by resource type
	rs.m.Store(resType.String(), rb)
	return nf, nil
}

// StoreResources batches resources for storing by ResourceType.
func (rs *ResourceStore) StoreResources(rss []*armresources.GenericResourceExpanded) (map[string]struct{}, error) {
	// don't attempt to store empty resources
	var errs []error
    if len(rss) == 0 {
        return nil, errors.New("no resources to store")
    }

	batchMap := make(map[string][]*armresources.GenericResourceExpanded)
	for _, r := range rss {
		rtype, err := arm.ParseResourceType(*r.ID)
		if err != nil {
			return nil, err
		}
		batchMap[rtype.String()] = append(batchMap[rtype.String()], r)
	}

	var notFound set
	for k, v := range batchMap {
		if nf, err := rs.storeResources(k, v); err != nil {
			errs = append(errs, err)
		} else {
			notFound = notFound.union(nf)
		}
	}
	return notFound, errors.Join(errs...)
}

func (rs *ResourceStore) DeleteResources(resourceIDs []string) error {
    /*
    var errs []error
	resID, err := arm.ParseResourceID(resourceIDs[0])
	if err != nil {
		return err
	}
	v, ok := rs.m.Load(resID.ResourceType.String())
	if !ok {
		return errors.New("resource type not found")
	}
	rb := v.(*ResourceBatch)

	// notFound := rb.DeleteResources(resourceIDs)
	if len(notFound) > 0 {
		return errors.New("resources not found: " + strings.Join(notFound, ", "))
	}
    */
	return nil
}

// Range is a typesafe wrapper for ranging over resource batches, which are keyed by ResourceType.
// It provides the same facility as sync.Map.Range, but with the ResourceBatch type.
func (rs *ResourceStore) Range(f func(resourceType string, rb *ResourceBatch) bool) {
	rs.m.Range(func(key, value interface{}) bool {
		return f(key.(string), value.(*ResourceBatch))
	})
}
