package resourcestore

import (
	// "errors"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
)

// Store the definitions with the Resources.
type Resource struct {
	// metricDefinitions map[string]*azquery.MetricDefinition
	resource          *armresources.GenericResourceExpanded
}

// NewResource creates a new Resource.
func NewResource(r *armresources.GenericResourceExpanded) *Resource {
	// for now it's as simple as wrapping the resource in our type
	return &Resource{
		resource: 			r,
		// metricDefinitions: mds,
	}
}
