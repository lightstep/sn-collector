package servicenowexporter

import (
	"bytes"
	"context"
	"strconv"
	"strings"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

const (
	// sanitizedRune is used to replace any invalid char per Carbon format.
	sanitizedRune = '_'

	// Tag related constants per Carbon plaintext protocol.
	tagPrefix                = ";"
	tagKeyValueSeparator     = "="
	tagValueEmptyPlaceholder = "<empty>"

	// Constants used when converting from distribution metrics to Carbon format.
	distributionBucketSuffix             = ".bucket"
	distributionUpperBoundTagKey         = "upper_bound"
	distributionUpperBoundTagBeforeValue = tagPrefix + distributionUpperBoundTagKey + tagKeyValueSeparator

	// Constants used when converting from summary metrics to Carbon format.
	summaryQuantileSuffix         = ".quantile"
	summaryQuantileTagKey         = "quantile"
	summaryQuantileTagBeforeValue = tagPrefix + summaryQuantileTagKey + tagKeyValueSeparator

	// Suffix to be added to original metric name for a Carbon metric representing
	// a count metric for either distribution or summary metrics.
	countSuffix = ".count"

	// Textual representation for positive infinity valid in Carbon, ie.:
	// positive infinity as represented in Python.
	infinityCarbonValue = "inf"

	midSource = "sn-otel-collector"
)

type serviceNowProducer struct {
	logger *zap.Logger
	config *Config
	client *midClient
}

func newServiceNowProducer(logger *zap.Logger, config *Config) *serviceNowProducer {
	return &serviceNowProducer{
		logger: logger,
		config: config,
		client: newMidClient(config, logger),
	}
}

func (e *serviceNowProducer) logDataPusher(_ context.Context, md plog.Logs) error {
	snLogs := make([]ServiceNowLog, 0)
	snEvents := make([]ServiceNowEvent, 0)

	useLogs := e.config.PushLogsURL != ""

	for i := 0; i < md.ResourceLogs().Len(); i++ {
		rl := md.ResourceLogs().At(i)
		resourceAttrs := rl.Resource().Attributes()
		for j := 0; j < rl.ScopeLogs().Len(); j++ {
			sl := rl.ScopeLogs().At(j)
			scope := sl.Scope().Name()

			for k := 0; k < sl.LogRecords().Len(); k++ {
				log := sl.LogRecords().At(k)
				if useLogs {
					newLog := ServiceNowLog{
						Body:         log.Body().AsString(),
						ResourcePath: buildPath("", log.Attributes()),
						Ci2LogID:     ci2metricAttrs(resourceAttrs),
						Timestamp:    formatTimestamp(log.Timestamp()),
						Severity:     log.SeverityText(),
						Node:         formatNode(ci2metricAttrs(resourceAttrs)),
						Source:       midSource,
					}
					snLogs = append(snLogs, newLog)
				} else {
					additionalInfo, err := formatAdditionalInfo(ci2metricAttrs(log.Attributes()), ci2metricAttrs(resourceAttrs))
					if err != nil {
						e.logger.Error("Failed to format additional info", zap.Error(err))
						continue
					}

					newEvent := ServiceNowEvent{
						Type:           scope,
						Description:    log.Body().AsString(),
						Resource:       buildPath("", log.Attributes()),
						Severity:       "5", // TODO: figure out this mapping
						Timestamp:      formatEventTimestamp(log.Timestamp()),
						Node:           formatNode(ci2metricAttrs(resourceAttrs)),
						Source:         midSource,
						AdditionalInfo: additionalInfo,
					}
					snEvents = append(snEvents, newEvent)
				}
			}
		}
	}

	if useLogs {
		e.logger.Info("Sending logs to MID Server...", zap.Any("logs", snLogs))
		err := e.client.sendLogs(snLogs)
		if err != nil {
			e.logger.Error("Failed to send logs to MID Server", zap.Int("logCount", len(snLogs)), zap.Error(err))
			return err
		}

		return nil
	}

	e.logger.Info("Sending events to instance...", zap.Any("events", snEvents))
	err := e.client.sendEvents(snEvents)
	if err != nil {
		e.logger.Error("Failed to send events to MID Server", zap.Int("logCount", len(snEvents)), zap.Error(err))
		return err
	}

	return nil
}

// based on: https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/exporter/carbonexporter/metricdata_to_plaintext.go#L82
func (e *serviceNowProducer) metricsDataPusher(_ context.Context, md pmetric.Metrics) error {
	snMetrics := make([]ServiceNowMetric, 0)

	for i := 0; i < md.ResourceMetrics().Len(); i++ {
		rm := md.ResourceMetrics().At(i)
		resourceAttrs := rm.Resource().Attributes()
		for j := 0; j < rm.ScopeMetrics().Len(); j++ {
			sm := rm.ScopeMetrics().At(j)
			scope := sm.Scope().Name()

			for k := 0; k < sm.Metrics().Len(); k++ {
				metric := sm.Metrics().At(k)
				if metric.Name() == "" {
					// TODO: log error info
					continue
				}
				switch metric.Type() {
				case pmetric.MetricTypeGauge:
					snMetrics = append(snMetrics, e.writeNumberDataPoints(metric.Name(), scope, resourceAttrs, metric.Gauge().DataPoints())...)
				case pmetric.MetricTypeSum:
					snMetrics = append(snMetrics, e.writeNumberDataPoints(metric.Name(), scope, resourceAttrs, metric.Sum().DataPoints())...)
				case pmetric.MetricTypeHistogram:
					snMetrics = append(snMetrics, e.formatHistogramDataPoints(metric.Name(), scope, resourceAttrs, metric.Histogram().DataPoints())...)
				case pmetric.MetricTypeSummary:
					snMetrics = append(snMetrics, e.formatSummaryDataPoints(metric.Name(), scope, resourceAttrs, metric.Summary().DataPoints())...)
				}
			}
		}
	}

	e.logger.Info("Sending metrics to MID Server...", zap.Any("metrics", len(snMetrics)))
	err := e.client.sendMetrics(snMetrics)

	if err != nil {
		e.logger.Error("Failed to send metric to MID Server", zap.Int("metricCount", len(snMetrics)), zap.Error(err))
		return err
	}
	return nil
}

func (e *serviceNowProducer) Close(context.Context) error {
	e.client.Close()
	return nil
}

func (e *serviceNowProducer) writeNumberDataPoints(metricName string, scope string, rAttrs pcommon.Map, dps pmetric.NumberDataPointSlice) []ServiceNowMetric {
	snm := make([]ServiceNowMetric, 0)
	for i := 0; i < dps.Len(); i++ {
		dp := dps.At(i)
		var val float64
		switch dp.ValueType() {
		case pmetric.NumberDataPointValueTypeEmpty:
			continue // skip this data point - otherwise an empty string will be used as the value and the backend will use the timestamp as the metric value
		case pmetric.NumberDataPointValueTypeInt:
			val = float64(dp.IntValue())
		case pmetric.NumberDataPointValueTypeDouble:
			val = float64(dp.DoubleValue())
		}
		snm = append(snm, e.createMetric(
			metricName,
			scope,
			ci2metricAttrs(rAttrs),
			buildPath(metricName, dp.Attributes()),
			val,
			formatTimestamp(dp.Timestamp())))
	}
	return snm
}

// Converts resource attributes to a map of string key/value pairs
// for use in ci2metric_id in the push metric API
func ci2metricAttrs(rAttrs pcommon.Map) map[string]string {
	attrs := make(map[string]string)
	rAttrs.Range(func(k string, v pcommon.Value) bool {
		attrs[k] = v.AsString()
		return true
	})
	return attrs
}

// formatHistogramDataPoints transforms a slice of histogram data points into a series
// of Carbon metrics and injects them into the string builder.
//
// Carbon doesn't have direct support to distribution metrics they will be
// translated into a series of Carbon metrics:
//
// 1. The total count will be represented by a metric named "<metricName>.count".
//
// 2. The total sum will be represented by a metric with the original "<metricName>".
//
// 3. Each histogram bucket is represented by a metric named "<metricName>.bucket"
// and will include a dimension "upper_bound" that specifies the maximum value in
// that bucket. This metric specifies the number of events with a value that is
// less than or equal to the upper bound.
func (e *serviceNowProducer) formatHistogramDataPoints(
	metricName string,
	scope string,
	rAttrs pcommon.Map,
	dps pmetric.HistogramDataPointSlice,
) []ServiceNowMetric {
	snm := make([]ServiceNowMetric, 0)

	for i := 0; i < dps.Len(); i++ {
		dp := dps.At(i)

		e.formatCountAndSum(metricName, scope, rAttrs, dp.Attributes(), dp.Count(), dp.Sum(), dp.Timestamp())
		if dp.ExplicitBounds().Len() == 0 {
			continue
		}

		bounds := dp.ExplicitBounds().AsRaw()
		carbonBounds := make([]string, len(bounds)+1)
		for i := 0; i < len(bounds); i++ {
			carbonBounds[i] = formatFloatForLabel(bounds[i])
		}
		carbonBounds[len(carbonBounds)-1] = infinityCarbonValue

		bucketPath := buildPath(metricName+distributionBucketSuffix, dp.Attributes())
		for j := 0; j < dp.BucketCounts().Len(); j++ {
			snm = append(snm, e.createMetric(
				metricName+distributionBucketSuffix,
				scope,
				ci2metricAttrs(rAttrs),
				bucketPath+distributionUpperBoundTagBeforeValue+carbonBounds[j],
				float64(dp.BucketCounts().At(j)),
				formatTimestamp(dp.Timestamp())))
		}
	}
	return snm
}

// formatSummaryDataPoints transforms a slice of summary data points into a series
// of Carbon metrics and injects them into the string builder.
//
// Carbon doesn't have direct support to summary metrics they will be
// translated into a series of Carbon metrics:
//
// 1. The total count will be represented by a metric named "<metricName>.count".
//
// 2. The total sum will be represented by a metric with the original "<metricName>".
//
// 3. Each quantile is represented by a metric named "<metricName>.quantile"
// and will include a tag key "quantile" that specifies the quantile value.
func (e *serviceNowProducer) formatSummaryDataPoints(
	metricName string,
	scope string,
	rAttrs pcommon.Map,
	dps pmetric.SummaryDataPointSlice,
) []ServiceNowMetric {
	snm := make([]ServiceNowMetric, 0)
	for i := 0; i < dps.Len(); i++ {
		dp := dps.At(i)

		e.formatCountAndSum(metricName, scope, rAttrs, dp.Attributes(), dp.Count(), dp.Sum(), dp.Timestamp())

		if dp.QuantileValues().Len() == 0 {
			continue
		}

		quantilePath := buildPath(metricName+summaryQuantileSuffix, dp.Attributes())
		for j := 0; j < dp.QuantileValues().Len(); j++ {
			snm = append(snm, e.createMetric(
				metricName+summaryQuantileSuffix,
				scope,
				ci2metricAttrs(rAttrs),
				quantilePath+summaryQuantileTagBeforeValue+formatFloatForLabel(dp.QuantileValues().At(j).Quantile()*100),
				dp.QuantileValues().At(j).Value(),
				formatTimestamp(dp.Timestamp())))
		}
	}
	return snm
}

// Carbon doesn't have direct support to distribution or summary metrics in both
// cases it needs to create a "count" and a "sum" metric. This function creates
// both, as follows:
//
// 1. The total count will be represented by a metric named "<metricName>.count".
//
// 2. The total sum will be represented by a metruc with the original "<metricName>".
func (e *serviceNowProducer) formatCountAndSum(
	metricName string,
	scope string,
	rAttrs pcommon.Map,
	attributes pcommon.Map,
	count uint64,
	sum float64,
	timestamp pcommon.Timestamp,
) []ServiceNowMetric {
	snm := make([]ServiceNowMetric, 0, 2)
	// Write count and sum metrics.
	snm = append(snm, e.createMetric(
		metricName,
		scope,
		ci2metricAttrs(rAttrs),
		buildPath(metricName+countSuffix, attributes),
		float64(count),
		formatTimestamp(timestamp)))

	snm = append(snm, e.createMetric(
		metricName,
		scope,
		ci2metricAttrs(rAttrs),
		buildPath(metricName, attributes),
		sum,
		formatTimestamp(timestamp)))
	return snm
}

// buildPath is used to build the <metric_path> per description above.
func buildPath(name string, attributes pcommon.Map) string {
	if attributes.Len() == 0 {
		return name
	}

	buf := new(bytes.Buffer)

	buf.WriteString(name)
	attributes.Range(func(k string, v pcommon.Value) bool {
		value := v.AsString()
		if value == "" {
			value = tagValueEmptyPlaceholder
		}
		buf.WriteString(tagPrefix)
		buf.WriteString(sanitizeTagKey(k))
		buf.WriteString(tagKeyValueSeparator)
		buf.WriteString(value)
		return true
	})

	return buf.String()
}

func formatAdditionalInfo(attrs map[string]string, resourceAttrs map[string]string) (string, error) {
	// merge attrs + resource attrs
	newAttrs := make(map[string]string)
	for k, v := range resourceAttrs {
		newAttrs[k] = v
	}

	for k, v := range attrs {
		newAttrs[k] = v
	}

	// convert resourceAttrs
	// key1=value2,key2=value2
	additionalInfoString := ""
	for k, v := range newAttrs {
		additionalInfoString += k + "=" + v + ","
	}
	return additionalInfoString, nil
}

func formatNode(resourceAttrs map[string]string) string {
	// TODO: make this mapping support more than host.name
	return resourceAttrs["host.name"]
}

func (e *serviceNowProducer) createMetric(name string, scope string, resourceAttrs map[string]string, path string, value float64, timestamp uint64) ServiceNowMetric {
	if scope != "" {
		resourceAttrs["otel.scope"] = scope
	}

	snm := ServiceNowMetric{
		MetricType:   name,
		ResourcePath: path,
		Value:        value,
		Timestamp:    timestamp,
		Source:       midSource,
		Ci2MetricID:  resourceAttrs,
	}

	// set by a processor (does not exist yet)
	ciClass := resourceAttrs["servicenow.ci.sys_id"]
	if ciClass != "" {
		snm.CiSysId = ciClass
		snm.Ci2MetricID = nil
	}

	snm.Node = formatNode(resourceAttrs)

	return snm
}

// sanitizeTagKey removes any invalid character from the tag key, the invalid
// characters are ";!^=".
func sanitizeTagKey(key string) string {
	mapRune := func(r rune) rune {
		switch r {
		case ';', '!', '^', '=':
			return sanitizedRune
		default:
			return r
		}
	}

	return strings.Map(mapRune, key)
}

// sanitizeTagValue removes any invalid character from the tag value, the invalid
// characters are ";~".
func sanitizeTagValue(value string) string {
	mapRune := func(r rune) rune {
		switch r {
		case ';', '~':
			return sanitizedRune
		default:
			return r
		}
	}

	return strings.Map(mapRune, value)
}

// Formats a float64 per Prometheus label value. This is an attempt to keep other
// the label values with different formats of metrics.
func formatFloatForLabel(f float64) string {
	return strconv.FormatFloat(f, 'g', -1, 64)
}

// Formats a float64 per Carbon plaintext format.
func formatFloatForValue(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}

func formatUint64(i uint64) string {
	return strconv.FormatUint(i, 10)
}

func formatInt64(i int64) string {
	return strconv.FormatInt(i, 10)
}

func formatEventTimestamp(timestamp pcommon.Timestamp) string {
	ts := timestamp.AsTime()
	return ts.Format("2006-01-02 15:04:05")
}

func formatTimestamp(timestamp pcommon.Timestamp) uint64 {
	return uint64(timestamp) / 1e6
}
