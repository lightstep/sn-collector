// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package newazureeventhubreceiver // "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/newazureeventhubreceiver"

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/receiver/receivertest"

	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/newazureeventhubreceiver/internal/metadata"
)

func Test_NewFactory(t *testing.T) {
	f := NewFactory()
	assert.Equal(t, metadata.Type, f.Type())
}

func Test_NewLogsReceiver(t *testing.T) {
	f := NewFactory()
	receiver, err := f.CreateLogsReceiver(context.Background(), receivertest.NewNopCreateSettings(), f.CreateDefaultConfig(), consumertest.NewNop())
	assert.NoError(t, err)
	assert.NotNil(t, receiver)
}

func Test_NewMetricsReceiver(t *testing.T) {
	f := NewFactory()
	receiver, err := f.CreateMetricsReceiver(context.Background(), receivertest.NewNopCreateSettings(), f.CreateDefaultConfig(), consumertest.NewNop())
	assert.NoError(t, err)
	assert.NotNil(t, receiver)
}
