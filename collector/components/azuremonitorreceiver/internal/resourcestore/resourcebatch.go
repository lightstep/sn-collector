package resourcestore

import (
	"errors"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/monitor/azquery"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
)

// ResourceType as the SDK provide it works for now, but we could provide an enum to
// force either a known resource type or explictly allow any string.
// Since batch fetching doesn't work with custom resources we're storing all metric
// definitions for the resource type in the batch. This is more important as a network
// optimization than memory optimization.

//  type MetricDefinitionMap map[string]*azquery.MetricDefinition
// the first string key is a namespace and the second is the metric name
type metricDefinitionMap map[string]map[string]*azquery.MetricDefinition

func (mdm metricDefinitionMap) Add(d *azquery.MetricDefinition) error {
	if d.Name == nil {
		return errors.New("metric definition name missing")
	}
	if d.Namespace == nil {
		return errors.New("metric definition namespace missing")
	}
	if _, ok := mdm[*d.Namespace]; !ok {
		mdm[*d.Namespace] = make(map[string]*azquery.MetricDefinition)
	}
	mdm[*d.Namespace][*d.Name.Value] = d
	return nil
}

// Delete removes the metric definition with the given name from the map.
func (mdm metricDefinitionMap) Delete(name string) {
	for _, ns := range mdm {
		if _, ok := ns[name]; ok {
			delete(ns, name)
			return
		}
	}
}

// Get returns the metric definition with the given namespace and name.
func (mdm metricDefinitionMap) Get(namespace string, name string) (*azquery.MetricDefinition, bool) {
	if ns, ok := mdm[namespace]; ok {
		if d, ok := ns[name]; ok {
			return d, true
		}
	}
	return nil, false
}

// Find examines each namespace until it finds the metric definition with the given name.
func (mdm metricDefinitionMap) Find(name string) (*azquery.MetricDefinition, bool) {
	for _, ns := range mdm {
		if d, ok := ns[name]; ok {
			return d, true
		}
	}
	return nil, false
}

type BatchRequest struct {
	ResourceType string
	Definitions  map[string]*azquery.MetricDefinition
	Resources    map[string]*Resource
}


// ResourceBatch is the value type stored in ResourceStore. It only exports
// only slice parameters and map returns, because it is intended to be called
// only for batches of resources.
type ResourceBatch struct {
	resourceType string
	definitions  map[string]map[string]*azquery.MetricDefinition
	resources    map[string]*Resource
}

func NewResourceBatch(resourceType string) *ResourceBatch {
	return &ResourceBatch{
		resourceType: resourceType,
		// definitions are keyed by namespace, then by metric name - so batch requests can be made
		definitions: 	make(map[string]map[string]*azquery.MetricDefinition),
		resources:    	make(map[string]*Resource),
	}
}

func (rb *ResourceBatch) ResourceType() string {
	return rb.resourceType
}

func (rb *ResourceBatch) Namespaces() []string {
	var ns []string
	for k := range rb.definitions {
		ns = append(ns, k)
	}
	return ns
}

func (rb *ResourceBatch) NamespaceMetricNames(namespace string) ([]string, error) {
	if _, ok := rb.definitions[namespace]; !ok {
		return nil, errors.New("namespace not found: " + namespace)
	}
	var names []string
	for k := range rb.definitions[namespace] {
		names = append(names, k)
	}
	return names, nil
}

func (rb *ResourceBatch) Definitions() map[string]map[string]*azquery.MetricDefinition {
	return rb.definitions
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
	return nil
}

// TODO: make the errors in this method of type validationError

// AddDefinition adds a metric definition to the ResourceBatch. It returns an error if the metric
// definition is not for the correct resource type or namespace.
func (rb *ResourceBatch) AddDefinition(d *azquery.MetricDefinition) error {
	// Validate that the metric definition is for the correct resource type and namespace.
	if rb.resourceType == "" {
		return errResourceTypeMissing
	}
	if d.Namespace == nil {
		return errors.New("namespace missing: " + *d.Name.Value)
	}

	resType, err := arm.ParseResourceType(*d.ResourceID)
	if err != nil {
		return err
	}
	if resType.String() != rb.resourceType {
		return errors.New("resource type mismatch: " + rb.resourceType + " != " + resType.String())
	}

	if _, ok := rb.definitions[*d.Namespace]; !ok {
		rb.definitions[*d.Namespace] = make(map[string]*azquery.MetricDefinition)
	}
	rb.definitions[*d.Namespace][*d.Name.Value] = d
	return nil
}

// StoreDefinitions batches storing the metric definitions in a ResourceBatch. The returned error
// is a joined error for all definitions not of this type. There's a memory optimization available,
// since we're using the batch API and it doesn't support custom metrics. That means we can store
// one set of metric definitions for all resources of a type. Otherwise, we have to use the
// non-batch API for custom metrics, which means we make a request per resource.
func (rb *ResourceBatch) StoreDefinitions(ds []*azquery.MetricDefinition) error {
	var errs []error
	for _, d := range ds {
		if err := rb.AddDefinition(d); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

/*
// GetDefinitionsByResource returns metric definitions for the given resource ID.
func (rb *ResourceBatch) getDefinitionsByResource(resourceID string) (map[string]*azquery.MetricDefinition, bool) {
	// get the resource by ID
	r, ok := rb.resources[resourceID]
	if !ok {
		return nil, false
	}
	return r.metricDefinitions, true
}
*/

// StoreResources provides batch storing of resources. It stores each resource within a batch.
func (rb *ResourceBatch) StoreResources(rs []*armresources.GenericResourceExpanded) (set, error) {
	// ensures: if rs[i].Type != this.resourceType, then append to errs
	// ensures: forall rs[i].Type == this.resourceType, rb.resources[*rs[i].ID] = rs[i] [see NOTE]
	// ensures: notFound := (rb.resources.Keys() - rs[i].ResourceID)
	// NOTE - we relax these guarantees, because we don't dedupe ResourceIDs. Thus the last
	// entry we process is what will be in the map. If this makes it incorrect, the problems aren't
	// solvable in this client code.

	// I would rather simply return an error
	if rb.resourceType == "" {
		return nil, errResourceTypeMissing
	}

	// oldResources is a set of resourceIDs which were not overwritten. Declare it then copy the current
	// data. We'll remove items that are overwritten and for now just provide the client with a delete
	// operation to handle as they see fit.
	oldResources := make(map[string]struct{})
	for _, v := range rb.resources {
		oldResources[*v.resource.ID] = struct{}{}
	}

	for k := range rb.resources {
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
			rb.resources[*r.ID] = NewResource(r)
		}
		rb.resources[*r.ID].resource = r
	}
	return oldResources, errors.Join(errs...)
}

// ResourceIDs returns a slice of resource IDs for the resources in the ResourceBatch.
func (rb *ResourceBatch) ResourceIDs() []string {
	var ids []string
	for k := range rb.resources {
		ids = append(ids, k)
	}
	return ids
}

// MetricNames returns a slice of metric names for the metric definitions in the ResourceBatch.
func (rb *ResourceBatch) MetricNames() []string {
	var names []string
	for k := range rb.definitions {
		names = append(names, k)
	}
	return names
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
