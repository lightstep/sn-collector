package azuresdk

import (
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/monitor/azquery"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
)

const (
	AttributeID 			= "id"
	AttributeLocation       = "location"
	AttributeName           = "name"
	AttributeResourceGroup  = "resource_group"
	AttributeResourceType   = "type"
	MetadataPrefix          = "metadata_"
	TagPrefix               = "tags_"
)

// CalculateResourceAttributes
func CalculateResourceAttributes(r *armresources.GenericResourceExpanded) (map[string]*string, error) {
	// we only need this, because ResourceGroupName isn't a field in the resource
	rid, err := arm.ParseResourceID(*r.ID)
	if err != nil {
		return nil, err
	}
	attributes := map[string]*string{
		AttributeID: 			to.Ptr(*r.ID),
		AttributeName:          r.Name,
		AttributeResourceGroup: to.Ptr(rid.ResourceGroupName),
		AttributeResourceType:  to.Ptr(*r.Type),
	}
	// check Location because it can be empty
	if r.Location != nil {
		attributes[AttributeLocation] = r.Location
	}
	return attributes, nil
}

func CalculateResourceFilter(types []string, groups []string) string {
	// TODO: switch to parsing services from
	// https://learn.microsoft.com/en-us/azure/azure-monitor/essentials/metrics-supported
	var builder strings.Builder
	builder.WriteString("(resourceType eq '")
	builder.WriteString(strings.Join(types, "' or resourceType eq '"))
	builder.WriteString("')")

	if len(groups) > 0 {
		builder.WriteString(" and (resourceGroup eq '")
		builder.WriteString(strings.Join(groups, "' or resourceGroup eq  '"))
		builder.WriteString("')")
	}
	return builder.String()
}

func LocalizableValuesToStringSlice(lss []*azquery.LocalizableString) []string {
	s := make([]string, 0, len(lss))
	for _, v := range lss {
		if v != nil && len(strings.TrimSpace(*v.Value)) > 0 {
			s = append(s, *v.Value)
		}
	}
	return s
}

// NOTE: I initially factored out this logic to a separate function, but it turns out
// that the purpose was just to batch requests, but with azquery we already get per
// resource batching.
func CalculateDimensionsFilter(dims []string) string {
	var builder strings.Builder
	for _, v := range dims[:len(dims)-1] {
		builder.WriteString(v)
		builder.WriteString(" eq '*' and ")
	}
	builder.WriteString(dims[len(dims)-1])
	builder.WriteString(" eq '*'")
	return builder.String()
}
