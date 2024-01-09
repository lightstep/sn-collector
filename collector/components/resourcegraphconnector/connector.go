package resourcegraphconnector

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

const (
	scopeName        = "otelcol/resourcegraphconnector"
	scopeVersion     = "v0001dev"
	ciClassAttribute = "servicenow.cmdb.ci.name"
)

type resource struct {
	metricsConsumer consumer.Metrics
	startTime       pcommon.Timestamp // start timestamp that will be applied to all recorded datapoints
	logger          *zap.Logger
	config          *Config
	resourceSchema  *ResourceSchema
}

func (r *resource) Start(_ context.Context, _ component.Host) error {
	r.logger.Info("using schema file", zap.String("path", r.config.SchemaPath), zap.String("schema version", r.resourceSchema.APIVersion))
	return nil
}

func (r *resource) Shutdown(_ context.Context) error {
	return nil
}

func (r *resource) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

type cmdbResourceMetric struct {
	name               string
	resourceAttributes map[string]string
	ciClass            string
}

func (r *resource) detectResourceMetric(tr TelemetryResource, attrs pcommon.Map) (*cmdbResourceMetric, bool) {
	// TODO: scope to metric name

	// Check if instrumentation name matches (if specified in schema)
	if tr.InstrumentationName != "" {
		val, exists := attrs.Get("instrumentation.name")
		if exists {
			if val.AsString() != tr.InstrumentationName {
				return nil, false
			}
		}
	}

	// Check if all identifying attributes exist
	exists := len(tr.IDAttributes) > 0
	for _, v := range tr.IDAttributes {
		_, hasAttribute := attrs.Get(v)
		exists = exists && hasAttribute
	}

	// Create a new map with the appropriate attributes
	if exists {
		newAttrs := make(map[string]string)

		for _, attr := range tr.IDAttributes {
			attrVal, hasAttribute := attrs.Get(attr)
			if hasAttribute {
				newAttrs[attr] = attrVal.AsString()
			}
		}
		for _, attr := range tr.Attributes {
			attrVal, hasAttribute := attrs.Get(attr)
			if hasAttribute {
				newAttrs[attr] = attrVal.AsString()
			}
		}

		return &cmdbResourceMetric{
			name:               fmt.Sprintf("servicenow.resource.%s.%s", tr.Name, scopeVersion),
			resourceAttributes: newAttrs,
			ciClass:            tr.CI,
		}, true
	}
	return nil, false
}

func (r *resource) addResourceMetric(m pmetric.ResourceMetrics, cm *cmdbResourceMetric) {
	m.Resource().Attributes().PutStr(ciClassAttribute, cm.ciClass)
	for k, v := range cm.resourceAttributes {
		m.Resource().Attributes().PutStr(k, v)
	}

	ilm := m.ScopeMetrics().AppendEmpty()
	ilm.Scope().SetName(scopeName)

	metric := m.ScopeMetrics().AppendEmpty().Metrics().AppendEmpty()
	metric.SetName(cm.name)
	metric.SetUnit("1")

	sum := metric.SetEmptySum()
	sum.SetAggregationTemporality(pmetric.AggregationTemporalityCumulative)
	sum.SetIsMonotonic(false)

	dataPoint := sum.DataPoints().AppendEmpty()
	dataPoint.SetIntValue(int64(1))
	dataPoint.SetStartTimestamp(r.startTime)
	dataPoint.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
}

func (r *resource) hasSource(sourceType string, sources []string) bool {
	for _, source := range sources {
		if source == sourceType {
			return true
		}
	}
	return false
}

func (r *resource) ConsumeLogs(ctx context.Context, ld plog.Logs) error {
	countMetrics := pmetric.NewMetrics()
	countMetrics.ResourceMetrics().EnsureCapacity(ld.ResourceLogs().Len())
	for i := 0; i < ld.ResourceLogs().Len(); i++ {
		rl := ld.ResourceLogs().At(i)

		for _, resource := range r.resourceSchema.TelemetryResources {
			if !r.hasSource("logs", resource.Sources) {
				continue
			}

			cm, exists := r.detectResourceMetric(resource, rl.Resource().Attributes())
			if exists {
				r.logger.Info("found resource via log", zap.String("resource", cm.name))
				r.addResourceMetric(countMetrics.ResourceMetrics().AppendEmpty(), cm)
			}
		}
	}
	return r.metricsConsumer.ConsumeMetrics(ctx, countMetrics)
}

func (r *resource) ConsumeMetrics(ctx context.Context, md pmetric.Metrics) error {
	countMetrics := pmetric.NewMetrics()
	countMetrics.ResourceMetrics().EnsureCapacity(md.ResourceMetrics().Len())

	for i := 0; i < md.ResourceMetrics().Len(); i++ {
		rm := md.ResourceMetrics().At(i)
		for _, resource := range r.resourceSchema.TelemetryResources {
			if !r.hasSource("metrics", resource.Sources) {
				continue
			}

			cm, exists := r.detectResourceMetric(resource, rm.Resource().Attributes())
			if exists {
				r.logger.Info("found resource via metrics", zap.String("resource", cm.name))
				r.addResourceMetric(countMetrics.ResourceMetrics().AppendEmpty(), cm)
			}
		}
	}
	return r.metricsConsumer.ConsumeMetrics(ctx, countMetrics)
}
