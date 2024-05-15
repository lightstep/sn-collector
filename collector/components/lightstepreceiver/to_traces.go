// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package lightstepreceiver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/lightstepreceiver"

import (
	"errors"
	"fmt"
	"time"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
	semconv "go.opentelemetry.io/collector/semconv/v1.18.0"

	"github.com/lightstep/sn-collector/collector/lightstepreceiver/internal/collectorpb"
)

const (
	InstrumentationScopeName = "lightstep-receiver"
	InstrumentationScopeVersion = "0.0.1" // TODO: Use the actual internal version?
)

func ToTraces(req *collectorpb.ReportRequest) (ptrace.Traces, error) {
	td := ptrace.NewTraces()
	if req.Reporter == nil {
		return td, errors.New("Reporter in ReportRequest cannot be null.")
	}
	if req.GetSpans() == nil || len(req.GetSpans()) == 0 {
		return td, nil
	}

	reporter := req.GetReporter()
	rss := td.ResourceSpans().AppendEmpty()
	resource := rss.Resource()
	translateTagsToAttrs(reporter.GetTags(), resource.Attributes())

	serviceName := getServiceName(reporter.GetTags())
	resource.Attributes().PutStr(semconv.AttributeServiceName, serviceName)
	sss := rss.ScopeSpans().AppendEmpty()
	scope := sss.Scope()
	scope.SetName(InstrumentationScopeName)
	scope.SetVersion(InstrumentationScopeVersion)

	spans := sss.Spans()
	spans.EnsureCapacity(len(req.GetSpans()))
	tstampOffset, _ := time.ParseDuration(fmt.Sprintf("%dus", req.GetTimestampOffsetMicros()))

	for _, lspan := range req.GetSpans() {
		span := spans.AppendEmpty()
		translateToSpan(lspan, span, tstampOffset)
	}

	return td, nil
}

func translateToSpan(lspan *collectorpb.Span, span ptrace.Span, offset time.Duration) {
	span.SetName(lspan.GetOperationName())
	translateTagsToAttrs(lspan.GetTags(), span.Attributes())

	ts := lspan.GetStartTimestamp()
	startt := time.Unix(ts.GetSeconds(), int64(ts.GetNanos()))

	duration, _ := time.ParseDuration(fmt.Sprintf("%dus", int64(lspan.GetDurationMicros())))
	span.SetStartTimestamp(pcommon.NewTimestampFromTime(startt.Add(offset)))
	span.SetEndTimestamp(pcommon.NewTimestampFromTime(startt.Add(duration).Add(offset)))

	// We store our ids using the left-most part of TraceID.
	span.SetTraceID(UInt64ToTraceID(0, lspan.GetSpanContext().GetTraceId()))
	span.SetSpanID(UInt64ToSpanID(lspan.GetSpanContext().GetSpanId()))
	setSpanParents(span, lspan.GetReferences())

	translateLogsToEvents(span, lspan.GetLogs(), offset)
}

func translateTagsToAttrs(tags []*collectorpb.KeyValue, attrs pcommon.Map) {
	attrs.EnsureCapacity(len(tags))
	for _, kv := range tags {
		key := kv.GetKey()
		value := kv.GetValue()
		switch x := value.(type) {
		case *collectorpb.KeyValue_StringValue:
			attrs.PutStr(key, x.StringValue)
		case *collectorpb.KeyValue_JsonValue:
			attrs.PutStr(key, x.JsonValue)
		case *collectorpb.KeyValue_IntValue:
			attrs.PutInt(key, x.IntValue)
		case *collectorpb.KeyValue_DoubleValue:
			attrs.PutDouble(key, x.DoubleValue)
		case *collectorpb.KeyValue_BoolValue:
			attrs.PutBool(key, x.BoolValue)
		}
	}
}

func getServiceName(tags []*collectorpb.KeyValue) string {
	for _, tag := range tags {
		if tag.GetKey() == "lightstep.component_name" {
			return tag.GetStringValue()
		}
	}

	// Identifier used by the SDKs when no service is specified, so we use it too.
	return "unknown_service"
}

func setSpanParents(span ptrace.Span, refs []*collectorpb.Reference) {
	if len(refs) == 0 {
		return
	}
	if len(refs) == 1 { // Common case, no need to do extra steps.
		span.SetParentSpanID(UInt64ToSpanID(refs[0].GetSpanContext().GetSpanId()))
		return
	}

	links := span.Links()
	links.EnsureCapacity(len(refs))
	is_main_parent_set := false
	for _, ref := range refs {
		if !is_main_parent_set {
			span.SetParentSpanID(UInt64ToSpanID(ref.GetSpanContext().GetSpanId()))
			is_main_parent_set = true
		} else {
			link := links.AppendEmpty()
			link.SetSpanID(UInt64ToSpanID(ref.GetSpanContext().GetSpanId()));
			link.SetTraceID(UInt64ToTraceID(0, ref.GetSpanContext().GetTraceId()))
		}
	}
}

func translateLogsToEvents(span ptrace.Span, logs []*collectorpb.Log, offset time.Duration) {
	if len(logs) == 0 {
		return
	}

	events := span.Events()
	events.EnsureCapacity(len(logs))
	for _, log := range logs {
		tstamp := time.Unix(log.GetTimestamp().GetSeconds(), int64(log.GetTimestamp().GetNanos())).Add(offset)
		event := events.AppendEmpty()
		event.SetTimestamp(pcommon.NewTimestampFromTime(tstamp))
		translateTagsToAttrs(log.GetFields(), event.Attributes())
	}
}
