package osqueryreceiver

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

func TestNewFactory(t *testing.T) {
	f := NewFactory()
	_, err := f.CreateMetricsReceiver(
		context.Background(),
		receivertest.NewNopCreateSettings(),
		f.CreateDefaultConfig(),
		consumertest.NewNop(),
	)
	require.NoError(t, err)
}
