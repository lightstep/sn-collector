package servicenowexporter

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFactory(t *testing.T) {
	f := NewFactory()
	assert.EqualValues(t, "servicenow", f.Type().String())
	cfg := f.CreateDefaultConfig()
	assert.NotNil(t, cfg)
}
