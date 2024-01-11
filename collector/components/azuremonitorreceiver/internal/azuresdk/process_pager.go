// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package azuresdk // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/azuremonitorreceiver/internal/azuresdk"

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"

	"github.com/Azure/azure-sdk-for-go/sdk/monitor/azquery"
)

// See the SDK documentation on Common Usage Patterns at
// https://learn.microsoft.com/en-us/azure/developer/go/azure-sdk-core-concepts#pagination-methods-that-return-collections
// for examples like the pattern used here.

// Client code can provide concurrency within the ProcessorFunc.

// T represents the type of data that the ProcessorFunc operates on. U represents the type of data that the ExtractorFunc returns.
// ExtractorFunc exists to help infer concrete types in Go, which doesn'generalize over value fields.
type ProcessorFunc[T any] func(context.Context, T) error
type ExtractorFunc[T any, U any] func(T) []U

// Extractor functions match a semantic like func(p *runtime.Pager[T]) []U {return p.Value}, we write it out because Go's type parameters don't
// generalize over value fields.
func ExtractResourceGroups(p armresources.ResourceGroupsClientListResponse) []*armresources.ResourceGroup { return p.Value}
func ExtractResources(p armresources.ClientListResponse) []*armresources.GenericResourceExpanded { return p.Value }
func ExtractMetricDefinitions(p azquery.MetricsClientListDefinitionsResponse) []*azquery.MetricDefinition { return p.Value }
func ExtractMetrics(p azquery.MetricsClientQueryResourceResponse) []*azquery.Metric { return p.Value }

// ProcessSlice applies an operation taking a context to a slice of values. It only forwards the context to the operation.
// This is useful for handling values in the AzureSDK, but not specialized for it.
func ProcessSlice[T any](ctx context.Context, slice []T, process ProcessorFunc[T]) error {
	for _, v := range slice {
		if err := process(ctx, v); err != nil {
			return err
		}
	}
	return nil
}

// ProcessPager is a generic function to process AzureSDK's Pager. It generalizes handling of the AzureSDK patterns described
// at https://learn.microsoft.com/en-us/azure/developer/go/azure-sdk-core-concepts.
func ProcessPager[T, U any](ctx context.Context, pager *runtime.Pager[T], extract ExtractorFunc[T, U], process ProcessorFunc[U]) error {
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
		 	return err
		}
		if err := ProcessSlice(ctx, extract(page), process); err != nil {
			return err
		}
	}
	return nil
}
