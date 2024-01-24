package resourcestore

import (
	"errors"

	// "github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/monitor/azquery"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
)

// ResourceType works for now, but in the future it could provide stronger type
// safety. We could check for valid resource types. For now it just provides
// readability in showing that it's a finite set of known strings.
// type ResourceType string

// type MetricDefinitionMap map[string]*azquery.MetricDefinition

// Store the definitions with the Resources.
type Resource struct {
	metricDefinitions map[string]*azquery.MetricDefinition
	resource          *armresources.GenericResourceExpanded
}

// GetDefinition returns the metric definition for the given name and true if it exists.
// Otherwise it returns nil and false.
func (r *Resource) GetDefinition(name string) (*azquery.MetricDefinition, bool) {
	d, ok := r.metricDefinitions[name]
	return d, ok
}

func (r *Resource) StoreDefinitions(ds []*azquery.MetricDefinition) error {
	// TODO: consider providing the notFound members, but I don't see a good use for this with the current code
	if r.resource == nil && r.resource.ID == nil {
		return errors.New("resource ID missing")
	}
	if r.metricDefinitions == nil {
		r.metricDefinitions = make(map[string]*azquery.MetricDefinition)
	}

	var (
		errs []error
		key string
	)

	for _, d := range ds {
		if d.Name != nil {
			key = *d.Name.Value
		} else if d.ID != nil {
			key = *d.ID
		} else {
			errs = append(errs, errors.New("metric definition name missing: " + *d.ID))
			continue
		}
		r.metricDefinitions[key] = d
	}

	return errors.Join(errs...)
}

// func (r *Resource) GetDefinitionsMap() (map[string]*azquery.MetricDefinition, bool) {
// 	if r.metricDefinitions == nil {
// 		return nil, false
// 	}
// 	return r.metricDefinitions, true
// }

// NewResource creates a new Resource.
func NewResource(r *armresources.GenericResourceExpanded, mds map[string]*azquery.MetricDefinition) *Resource {
	return &Resource{
		resource: 			r,
		metricDefinitions: mds,
	}
}

// ResourceBatch is the value type stored in ResourceStore. It only exports
// only slice parameters and map returns, because it is intended to be called
// only for batches of resources.
type ResourceBatch struct {
	resourceType string
	resources    map[string]*Resource
}

func NewResourceBatch(resourceOrType string) (*ResourceBatch, error) {
	rt, err := arm.ParseResourceType(resourceOrType)
	if err != nil {
		return nil, err
	}

	return newResourceBatch(rt.String()), nil
}

// only use the unexported after you confirm
func newResourceBatch(resourceType string ) *ResourceBatch {
	return &ResourceBatch{
		resourceType: resourceType,
		resources:    make(map[string]*Resource),
	}
}

func (rb *ResourceBatch) AddResource(r *armresources.GenericResourceExpanded) error {
	if rb.resourceType == "" {
		return errResourceTypeMissing
	}
	if r.Type == nil {
		return errors.New("resource type missing")
	}
	if *r.Type != rb.resourceType {
		return errResourceTypeMismatch(rb.resourceType, *r.Type)
	}
	if _, ok := rb.resources[*r.ID]; ok {
		return errors.New("resource already exists")
	}
	rb.resources[*r.ID] = NewResource(r, nil)
	return nil
}

func (rb *ResourceBatch) AddDefinition(d *azquery.MetricDefinition) error {
	if rb.resourceType == "" {
		return errResourceTypeMissing
	}
	if d.ResourceID == nil {
		return errors.New("resource ID missing")
	}
	if *d.ResourceID == "" {
		return errors.New("resource ID empty")
	}
	if _, ok := rb.resources[*d.ResourceID]; !ok {
		return errors.New("resource not found")
	}
	// if _, ok := rb.resources[*d.ResourceID].metricDefinitions[*d.Name.Value]; ok {
	// 	return errors.New("metric definition already exists")
	// }
	rb.resources[*d.ResourceID].metricDefinitions[*d.Name.Value] = d
	return nil
}

// StoreDefinitions batches storing the metric definitions in a ResourceBatch. The returned error
// is a joined error for all definitions not of this type. There's a memory optimization available,
// since we're using the batch API and it doesn't support custom metrics. That means we can store
// one set of metric definitions for all resources of a type. Otherwise, we have to use the
// non-batch API for custom metrics, which means we make a request per resource.
func (rb *ResourceBatch) StoreDefinitions(resourceID string, ds []*azquery.MetricDefinition) error {
	var errs []error
	if resourceID == "" {
		return errors.New("resource type missing")
	}
	if _, ok := rb.resources[resourceID]; !ok {
		rb.resources[resourceID] = NewResource(nil, make(map[string]*azquery.MetricDefinition))
	}
	for _, d := range ds {
		if d.ResourceID == nil || *d.ResourceID != resourceID {
			errs = append(errs, errors.New("resource ID mismatch: " + *d.ResourceID + " != " + resourceID))
			continue
		}
		rb.resources[resourceID].metricDefinitions[*d.Name.Value] = d
	}
	// Join returns an error who's Error() method is new line delimited concatenation of Error()
	// of joined errors in depth first order. It also has plural variation of Unwrap
	// which returns []error.
	return errors.Join(errs...)
}

// GetDefinitionsByResource returns metric definitions for the given resource ID.
func (rb *ResourceBatch) getDefinitionsByResource(resourceID string) (map[string]*azquery.MetricDefinition, bool) {
	// get the resource by ID
	r, ok := rb.resources[resourceID]
	if !ok {
		return nil, false
	}
	return r.metricDefinitions, true
}

// StoreResources provides batch storing of resources. It stores each resource within a batch.
func (rb *ResourceBatch) storeResources(rs []*armresources.GenericResourceExpanded) (set, error) {
	// ensures: if rs[i].Type != this.resourceType, then append to errs
	// ensures: forall rs[i].Type == this.resourceType, rb.resources[*rs[i].ID] = rs[i] [see NOTE]
	// ensures: notFound := (rb.resources.Keys() - rs[i].ResourceID)
	// NOTE - we relax these guarantees, because we don't dedupe ResourceIDs. Thus the last
	// entry we process is what will be in the map. If this makes it incorrect, the problems aren't
	// solvable in client code.

	// I would rather simply return an error
	if rb.resourceType == "" {
		return nil, errResourceTypeMissing
	}

	// oldResources is a set of resourceIDs which were not overwritten. Declare it then copy the data.
	oldResources := make(map[string]struct{})
	for _, v := range rb.resources {
		oldResources[*v.resource.ID] = struct{}{}
	}

	for k, _ := range rb.resources {
		oldResources[k] = struct{}{}
	}

	var errs []error
	for _, r := range rs {
		// NOTE: the ordering of checks is important to prevent possibile nil dereference
		if r.Type == nil {
			errs = append(errs, errResourceTypeMissing)
			continue
		}
		if *r.Type != rb.resourceType {
			errs = append(errs, errResourceTypeMismatch(rb.resourceType, *r.Type))
			continue
		}
		if _, ok := rb.resources[*r.ID]; ok {
			delete(oldResources, *r.ID)
		} else {
			rb.resources[*r.ID] = NewResource(r, nil)
		}
		rb.resources[*r.ID].resource = r
	}
	return oldResources, errors.Join(errs...)
}

func (rb *ResourceBatch) GetResources() (map[string]*Resource, bool) {
	return rb.resources, true
}

// DeleteResources takes a slice of resource IDs and deletes the corresponding resources
// from the ResourceBatch. It returns a slice of resource IDs which were not found.
func (rb *ResourceBatch) DeleteResources(resourceIDs []string) error {
	var errs []error
	for _, id := range resourceIDs {
		if _, ok := rb.resources[id]; ok {
			delete(rb.resources, id)
		} else {
			errs = append(errs, errors.New("resource not found: " + id))
		}
	}
	return errors.Join(errs...)
}

// Range iterates over the resources in the ResourceBatch.
func (rb *ResourceBatch) Range(f func(string, *Resource) bool) {
	for k, v := range rb.resources {
		if !f(k, v) {
			break
		}
	}
}

var (
	errResourceTypeMissing = errors.New("resource type missing")
)

func errResourceTypeMismatch(batchType string, resourceType string) error {
	return errors.New("resource type mismatch: " + string(batchType) + " != " + resourceType)
}
