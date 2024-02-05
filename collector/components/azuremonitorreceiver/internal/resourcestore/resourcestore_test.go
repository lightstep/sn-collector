package resourcestore

import (
	// "errors"
	// "reflect"
	"testing"

	"github.com/google/go-cmp/cmp"

	// "github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/monitor/azquery"
	// "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
	// "github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	// "github.com/lightstep/sn-collector/collector/components/azuremonitorreceiver/internal/resourcestore"
)

func toLocalizableString(s string) *azquery.LocalizableString {
	return &azquery.LocalizableString{Value: to.Ptr(s)}
}

const (
	resourceID_A = "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/{resourceProviderNamespace}/type-a/{resourceName}"
	resourceID_B = "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/{resourceProviderNamespace}/type-b/{resourceName}"
	resourceID_C = "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/{resourceProviderNamespace}/type-c/{resourceName}"
)

func TestResourceStore_StoreDefinitions(t *testing.T) {
	// using table driven tests - which setup cases and call them in a loop
	// calling t.Parallel for test and subtests while preventing variable capture
	// using cmp.Diff

	var testCases = []struct {
		name        *string
		definitions []*azquery.MetricDefinition
		expected    map[string]*azquery.MetricDefinition
		errs 	  bool
	}{
		{
			name:        to.Ptr("empty slice"),
			definitions: []*azquery.MetricDefinition{},
			expected:    map[string]*azquery.MetricDefinition{},
			errs: true,
		},
		{
			name: to.Ptr("one definition"),
			definitions: []*azquery.MetricDefinition{
				{
					ID:         to.Ptr("id1"),
					Name:       toLocalizableString("name"),
					Namespace:  to.Ptr("namespace1"),
					ResourceID: to.Ptr(resourceID_A),
				},
			},
			expected: map[string]*azquery.MetricDefinition{
				"name": {
					ID:         to.Ptr("id1"),
					Name:       toLocalizableString("name"),
					Namespace:  to.Ptr("namespace1"),
					ResourceID: to.Ptr(resourceID_A),
				},
			},
			errs: false,
		},
		{
			name: to.Ptr("multiple definitions"),
			definitions: []*azquery.MetricDefinition{
				{
					ID:         to.Ptr("id1"),
					Name:       toLocalizableString("name"),
					Namespace: to.Ptr("namespace1"),
					ResourceID: to.Ptr(resourceID_A),
				},
				{
					ID:         to.Ptr("id2"),
					Name:       toLocalizableString("name2"),
					Namespace: 	to.Ptr("namespace2"),
					ResourceID: to.Ptr(resourceID_A),
				},
			},
			expected: map[string]*azquery.MetricDefinition{
				"name": {
					ID:         to.Ptr("id1"),
					Name:       toLocalizableString("name"),
					Namespace:  to.Ptr("namespace1"),
					ResourceID: to.Ptr(resourceID_A),
				},
				"name2": {
					ID:         to.Ptr("id2"),
					Name:       toLocalizableString("name2"),
					Namespace:  to.Ptr("namespace2"),
					ResourceID: to.Ptr(resourceID_A),
				},
			},
		},
		{
			// NOTE: this tests that we don't store definitions for resources of a different type
			// but this is a silent failure, so we may want to expect a different behavior
			name: to.Ptr("multiple definitions with one not of the same type"),
			definitions: []*azquery.MetricDefinition{
				{
					ID:         to.Ptr("id1"),
					Name:       toLocalizableString("name"),
					Namespace:  to.Ptr("namespace1"),
					ResourceID: to.Ptr(resourceID_A),
				},
				{
					ID:         to.Ptr("id2"),
					Name:       toLocalizableString("name2"),
					Namespace:  to.Ptr("namespace2"),
					ResourceID: to.Ptr(resourceID_A),
				},
				{
					ID:         to.Ptr("id3"),
					Name:       toLocalizableString("name3"),
					Namespace:  to.Ptr("namespace3"),
					ResourceID: to.Ptr(resourceID_B),
				},
			},
			expected: map[string]*azquery.MetricDefinition{
				"name": {
					ID:         to.Ptr("id1"),
					Name:       toLocalizableString("name"),
					Namespace:  to.Ptr("namespace1"),
					ResourceID: to.Ptr(resourceID_A),
				},
				"name2": {
					ID:         to.Ptr("id2"),
					Name:       toLocalizableString("name2"),
					Namespace:  to.Ptr("namespace2"),
					ResourceID: to.Ptr(resourceID_A),
				},
			},
			errs: true,
		},
		{
			// NOTE: this tests that we don't store definitions for resources of a different type
			// but this is a silent failure, so we may want to expect a different behavior
			name: to.Ptr("multiple definitions with one not of the same type and one not of the same type as the first"),
			definitions: []*azquery.MetricDefinition{
				{
					ID:         to.Ptr("id1"),
					Name:       toLocalizableString("name"),
					Namespace:  to.Ptr("namespace1"),
					ResourceID: to.Ptr(resourceID_A),
				},
				{
					ID:         to.Ptr("id2"),
					Name:       toLocalizableString("name2"),
					Namespace:  to.Ptr("namespace2"),
					ResourceID: to.Ptr(resourceID_A),
				},
				{
					ID:         to.Ptr("id3"),
					Name:       toLocalizableString("name3"),
					Namespace:  to.Ptr("namespace3"),
					ResourceID: to.Ptr(resourceID_B),
				},
				{
					ID:         to.Ptr("id4"),
					Name:       toLocalizableString("name4"),
					Namespace:  to.Ptr("namespace3"),
					ResourceID: to.Ptr(resourceID_C),
				},
			},
			expected: map[string]*azquery.MetricDefinition{
				"name": {
					ID:         to.Ptr("id1"),
					Name:       toLocalizableString("name"),
					ResourceID: to.Ptr(resourceID_A),
				},
				"name2": {
					ID:         to.Ptr("id2"),
					Name:       toLocalizableString("name2"),
					ResourceID: to.Ptr(resourceID_A),
				},
			},
			errs: true,
		},
		{
			name: to.Ptr("multiple definitions with one not of the same type and one not of the same type as the first and one not of the same type as the second"),
			definitions: []*azquery.MetricDefinition{
				{
					ID:         to.Ptr("id1"),
					Name:       toLocalizableString("name"),
					Namespace:  to.Ptr("namespace1"),
					ResourceID: to.Ptr(resourceID_A),
				},
				{
					ID:         to.Ptr("id2"),
					Name:       toLocalizableString("name2"),
					Namespace:  to.Ptr("namespace2"),
					ResourceID: to.Ptr(resourceID_B),
				},
				{
					ID:         to.Ptr("id3"),
					Name:       toLocalizableString("name3"),
					Namespace:  to.Ptr("namespace3"),
					ResourceID: to.Ptr(resourceID_C),
				},
				{
					ID:         to.Ptr("id4"),
					Name:       toLocalizableString("name4"),
					Namespace:  to.Ptr("namespace3"),
					ResourceID: to.Ptr(resourceID_A),
				},
				{
					ID:         to.Ptr("id5"),
					Name:       toLocalizableString("name5"),
					Namespace:  to.Ptr("namespace3"),
					ResourceID: to.Ptr(resourceID_B),
				},
			},
			expected: map[string]*azquery.MetricDefinition{
				"name": {
					ID:         to.Ptr("id1"),
					Name:       toLocalizableString("name"),
					Namespace:  to.Ptr("namespace1"),
					ResourceID: to.Ptr(resourceID_A),
				},
				"name4": {
					ID:         to.Ptr("id4"),
					Name:       toLocalizableString("name4"),
					Namespace:  to.Ptr("namespace3"),
					ResourceID: to.Ptr(resourceID_A),
				},
			},
			errs: true,
		},
	}

	for _, tc := range testCases {
		// NOTE: this is a closure, so we need to pass the tc to the subtest otherwise
		// we'll get the last value of tc for all subtests
		tc := tc

		t.Run(*tc.name, func(t *testing.T) {
			t.Parallel()

			if tc.definitions == nil {
				t.Errorf("tc definitions is nil")
			}

			// init for the type
			rs := NewResourceStore()

			// invoke the SUT
			if err := rs.StoreDefinitions(tc.definitions); err != nil {
				// TODO: we'll check the new error handling later
				if tc.errs {
					// this is all good if we expected the error
					return
				}
				t.Errorf("error storing definitions: %s", err)
			}

			// get the len of the sync map
			var lenOfSyncMap int
			rs.m.Range(func(_, _ interface{}) bool {
				lenOfSyncMap++
				return true
			})

			if lenOfSyncMap == 0 && len(tc.definitions) != 0 {
				t.Errorf("lenOfSyncMap is 0, but tc definitions is not")
			}

			// filter this out, because this is a pass on a test case
			if len(tc.definitions) == 0 && lenOfSyncMap == 0 {
				return
			}

			if tc.definitions[0].ResourceID == nil {
				t.Errorf("tc definitions[0].ResourceID is nil")
			}

			resID, err := arm.ParseResourceID(*tc.definitions[0].ResourceID)
			if err != nil {
				t.Errorf("error parsing resource ID: %s", err)
			}

			defs, ok := rs.GetDefinitions(resID.ResourceType.String())
			if !ok {
				t.Errorf("error getting definitions: %s", err)
			}

			if diff := cmp.Diff(defs, tc.expected); diff != "" {
				t.Errorf("unexpected diff: %s", diff)
			}
		})
	}
}
