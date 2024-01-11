// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package azuresdk_test // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/azuremonitorreceiver/internal/azuresdk"

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"

	azfake "github.com/Azure/azure-sdk-for-go/sdk/azcore/fake"

	"github.com/Azure/azure-sdk-for-go/sdk/monitor/azquery"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources/fake"

	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/azuremonitorreceiver/internal/azuresdk"
)

func fakeListByResourceGroupResponder() azfake.PagerResponder[armresources.ClientListByResourceGroupResponse] {
	r := azfake.PagerResponder[armresources.ClientListByResourceGroupResponse]{}
	r.AddPage(
		200,
		armresources.ClientListByResourceGroupResponse{
			ResourceListResult: armresources.ResourceListResult{
				Value: []*armresources.GenericResourceExpanded{
					{
						ID:   to.Ptr("fakeID1"),
						Name: to.Ptr("fakeName1"),
					},
					{
						ID:   to.Ptr("fakeID2"),
						Name: to.Ptr("fakeName2"),
					},
				},
			},
		},
		nil,
	)

	return r
}

// Azure SDK provides fakes for testing. All of our workflows rely on the same pattern with
// a *runtime.Pager[T], so we build fake Pagers in terms of the pattern we use to process them.
func fakeListResourcesResponder() azfake.PagerResponder[armresources.ClientListResponse] {
	r := azfake.PagerResponder[armresources.ClientListResponse]{}
	r.AddPage(
		200,
		armresources.ClientListResponse{
			ResourceListResult: armresources.ResourceListResult{
				Value: []*armresources.GenericResourceExpanded{
					{
						ID:   to.Ptr("fakeID1"),
						Name: to.Ptr("fakeName1"),
					},
					{
						ID:   to.Ptr("fakeID2"),
						Name: to.Ptr("fakeName2"),
					},
					{
						ID:   to.Ptr("fakeID3"),
						Name: to.Ptr("fakeName3"),
					},
				},
			},
		},
		nil,
	)

	return r
}

func fakeMetricDefinitionResponder() azfake.PagerResponder[azquery.MetricsClientListDefinitionsResponse] {
	r := azfake.PagerResponder[azquery.MetricsClientListDefinitionsResponse]{}
	r.AddPage(
		200,
		azquery.MetricsClientListDefinitionsResponse{
			MetricDefinitionCollection: azquery.MetricDefinitionCollection{
				Value: []*azquery.MetricDefinition{
					{
						ID:         to.Ptr("fakeID1"),
						ResourceID: to.Ptr("fakeResourceID1"),
						Name:       toLocalizableString("fakeName1"),
					},
					{
						ID:         to.Ptr("fakeID2"),
						ResourceID: to.Ptr("fakeResourceID2"),
						Name:       toLocalizableString("fakeName2"),
					},
				},
			},
		},
		nil,
	)

	return r
}

func fakeQueryResourceResponder() azfake.PagerResponder[azquery.MetricsClientQueryResourceResponse] {
	r := azfake.PagerResponder[azquery.MetricsClientQueryResourceResponse]{}
	r.AddPage(
		200,
		azquery.MetricsClientQueryResourceResponse{
			Response: azquery.Response{
				Value: []*azquery.Metric{
					{
						ID:   to.Ptr("fakeID1"),
						Name: toLocalizableString("fakeName1"),
					},
					{
						ID:   to.Ptr("fakeID2"),
						Name: toLocalizableString("fakeName2"),
					},
					{
						ID:   to.Ptr("fakeID3"),
						Name: toLocalizableString("fakeName3"),
					},
				},
			},
		},
		nil,
	)
	return r
}

type responderValues struct {
	StatusCode int
	ID1, Name1 string
	ID2, Name2 string
}

func toLocalizableString(s string) *azquery.LocalizableString {
	return &azquery.LocalizableString{
		Value: to.Ptr(s),
	}
}

func fakeServerFactory() fake.ServerFactory {
	return fake.ServerFactory{
		Server: fake.Server{
			NewListPager: func(_ *armresources.ClientListOptions) azfake.PagerResponder[armresources.ClientListResponse] {
				return fakeListResourcesResponder()
			},
			NewListByResourceGroupPager: func(resourceGroupName string, _ *armresources.ClientListByResourceGroupOptions) azfake.PagerResponder[armresources.ClientListByResourceGroupResponse] {
				return fakeListByResourceGroupResponder()
			},
		},
	}
}

var testCases = []struct {
	name      string
	data      responderValues
	expectErr bool
}{
	{
		name: "successful processing",
		data: responderValues{
			StatusCode: 200,
			ID1:        "fakeID1",
			Name1:      "fakeName1",
			ID2:        "fakeID2",
			Name2:      "fakeName2",
		},
		expectErr: false,
	},
	{
		name: "error during page extraction",
		data: responderValues{
			StatusCode: 200,
			ID1:        "fakeID1",
			Name1:      "fakeName1",
			ID2:        "fakeID2",
			Name2:      "fakeName2",
		},
	},
}

// At this time the purpose of the test is to show use of the ProcessPager function for
// handling multiple pages with multiple items.
// TODO: Add cases for error handling - potentially providing for injection of error handlers
//
//	in the ProcessPager function.
func TestProcessPagerClient(t *testing.T) {
	ctx := context.Background()
	f := fakeServerFactory()
	c, _ := armresources.NewClient(
		"fake-subscription-id",
		&azfake.TokenCredential{},
		&arm.ClientOptions{
			ClientOptions: azcore.ClientOptions{
				Transport: fake.NewServerTransport(&f.Server),
			},
		},
	)

	// run all cases in parallel
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			pager := c.NewListPager(&armresources.ClientListOptions{})
			extract := func(page armresources.ClientListResponse) []*armresources.GenericResourceExpanded {
				return page.Value
			}
			process := func(ctx context.Context, v *armresources.GenericResourceExpanded) error {
				t.Log(*v.Name)
				return nil
			}

			err := azuresdk.ProcessPager[armresources.ClientListResponse, *armresources.GenericResourceExpanded](ctx, pager, extract, process)
			assert.NoError(t, err)
		})
	}
}

func TestProcessPagerByResourceGroupClient(t *testing.T) {
	ctx := context.Background()
	f := fakeServerFactory()

	c, _ := armresources.NewClient(
		"fake-subscription-id",
		&azfake.TokenCredential{},
		&arm.ClientOptions{
			ClientOptions: azcore.ClientOptions{
				Transport: fake.NewServerTransport(&f.Server),
			},
		},
	)

	// run all cases in parallel
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			pager := c.NewListByResourceGroupPager("fake-resource-group-name", &armresources.ClientListByResourceGroupOptions{})
			extract := func(page armresources.ClientListByResourceGroupResponse) []*armresources.GenericResourceExpanded {
				return page.Value
			}

			process := func(ctx context.Context, v *armresources.GenericResourceExpanded) error {
				t.Log(*v.Name)
				return nil
			}

			err := azuresdk.ProcessPager[armresources.ClientListByResourceGroupResponse, *armresources.GenericResourceExpanded](ctx, pager, extract, process)
			assert.NoError(t, err)
		})
	}
}
