// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package lightstepreceiver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/lightstepreceiver"

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/lightstep/sn-collector/collector/lightstepreceiver/internal/collectorpb"
)

const (
	TraceID1 = 5208512171318403364
	TraceID2 = 1645381947485011451
	SpanID1  = 4819382048639779717
	SpanID2  = 5060571933882717101
	SpanID3  = 1174406194215929934
)

func TestTranslateAllMembersNil(t *testing.T) {
	req := &collectorpb.ReportRequest{
		Reporter:              nil,
		Spans:                 nil,
		Auth:                  nil,
		TimestampOffsetMicros: 0,
		InternalMetrics:       nil,
	}
	traces, err := ToTraces(req)
	assert.Equal(t, errors.New("Reporter in ReportRequest cannot be null."), err)
	assert.Equal(t, ptrace.NewTraces(), traces)
}

func TestTranslateEmptySpans(t *testing.T) {
	req := &collectorpb.ReportRequest{
		Reporter: &collectorpb.Reporter{
			Tags: []*collectorpb.KeyValue{
				{
					Key: "lightstep.component_name",
					Value: &collectorpb.KeyValue_StringValue{
						StringValue: "GatewayService",
					},
				},
			},
		},
		Spans: []*collectorpb.Span{},
	}
	traces, err := ToTraces(req)
	assert.NoError(t, err)
	assert.Equal(t, traces, ptrace.NewTraces())
}

func TestTranslatNoComponentName(t *testing.T) {
	req := &collectorpb.ReportRequest{
		Reporter: &collectorpb.Reporter{},
		Spans: []*collectorpb.Span{
			{
				OperationName: "span1",
			},
		},
	}
	traces, err := ToTraces(req)
	assert.NoError(t, err)
	assert.Equal(t, traces, func() ptrace.Traces {
		td := ptrace.NewTraces()
		rs := td.ResourceSpans().AppendEmpty()

		r := rs.Resource()
		rattrs := r.Attributes()
		rattrs.PutStr("service.name", "unknown_service") // fallback

		sss := rs.ScopeSpans().AppendEmpty()
		scope := sss.Scope()
		scope.SetName("lightstep-receiver")
		scope.SetVersion("0.0.1")

		spans := sss.Spans()
		span1 := spans.AppendEmpty()
		span1.SetName("span1")

		return td
	}())

}

func TestAttributes(t *testing.T) {
	req := &collectorpb.ReportRequest{
		Reporter: &collectorpb.Reporter{
			Tags: []*collectorpb.KeyValue{
				{
					Key: "lightstep.component_name",
					Value: &collectorpb.KeyValue_StringValue{
						StringValue: "GatewayService",
					},
				},
				{
					Key: "strTag",
					Value: &collectorpb.KeyValue_StringValue{
						StringValue: "strValue",
					},
				},
				{
					Key: "intTag",
					Value: &collectorpb.KeyValue_IntValue{
						IntValue: 123456789,
					},
				},
				{
					Key: "doubleTag",
					Value: &collectorpb.KeyValue_DoubleValue{
						DoubleValue: 123456.789,
					},
				},
				{
					Key: "boolTag",
					Value: &collectorpb.KeyValue_BoolValue{
						BoolValue: true,
					},
				},
				{
					Key: "jsonTag",
					Value: &collectorpb.KeyValue_JsonValue{
						JsonValue: "{\"foo\": \"bar\"}",
					},
				},
			},
		},
		Spans: []*collectorpb.Span{
			{
				OperationName: "span1",
				Tags: []*collectorpb.KeyValue{
					{
						Key: "spanStrTag",
						Value: &collectorpb.KeyValue_StringValue{
							StringValue: "spanStrValue",
						},
					},
					{
						Key: "spanIntTag",
						Value: &collectorpb.KeyValue_IntValue{
							IntValue: 123456789,
						},
					},
					{
						Key: "spanDoubleTag",
						Value: &collectorpb.KeyValue_DoubleValue{
							DoubleValue: 123456.789,
						},
					},
					{
						Key: "spanBoolTag",
						Value: &collectorpb.KeyValue_BoolValue{
							BoolValue: false,
						},
					},
					{
						Key: "spanJsonTag",
						Value: &collectorpb.KeyValue_JsonValue{
							JsonValue: "{\"foo\": \"bar\"}",
						},
					},
				},
				Logs: []*collectorpb.Log{
					{
						Fields: []*collectorpb.KeyValue{
							{
								Key: "logStrTag",
								Value: &collectorpb.KeyValue_StringValue{
									StringValue: "logStrValue",
								},
							},
							{
								Key: "logIntTag",
								Value: &collectorpb.KeyValue_IntValue{
									IntValue: 123456789,
								},
							},
							{
								Key: "logDoubleTag",
								Value: &collectorpb.KeyValue_DoubleValue{
									DoubleValue: 123456.789,
								},
							},
							{
								Key: "logBoolTag",
								Value: &collectorpb.KeyValue_BoolValue{
									BoolValue: false,
								},
							},
							{
								Key: "logJsonTag",
								Value: &collectorpb.KeyValue_JsonValue{
									JsonValue: "{\"foo\": \"bar\"}",
								},
							},
						},
					},
				},
			},
		},
	}
	traces, err := ToTraces(req)
	assert.NoError(t, err)
	assert.Equal(t, traces, func() ptrace.Traces {
		td := ptrace.NewTraces()
		rs := td.ResourceSpans().AppendEmpty()

		r := rs.Resource()
		rattrs := r.Attributes()
		rattrs.PutStr("lightstep.component_name", "GatewayService")
		rattrs.PutStr("strTag", "strValue")
		rattrs.PutInt("intTag", 123456789)
		rattrs.PutDouble("doubleTag", 123456.789)
		rattrs.PutBool("boolTag", true)
		rattrs.PutStr("jsonTag", "{\"foo\": \"bar\"}")
		rattrs.PutStr("service.name", "GatewayService") // derived

		sss := rs.ScopeSpans().AppendEmpty()
		scope := sss.Scope()
		scope.SetName("lightstep-receiver")
		scope.SetVersion("0.0.1")

		spans := sss.Spans()
		span1 := spans.AppendEmpty()
		span1.SetName("span1")
		s1attrs := span1.Attributes()
		s1attrs.PutStr("spanStrTag", "spanStrValue")
		s1attrs.PutInt("spanIntTag", 123456789)
		s1attrs.PutDouble("spanDoubleTag", 123456.789)
		s1attrs.PutBool("spanBoolTag", false)
		s1attrs.PutStr("spanJsonTag", "{\"foo\": \"bar\"}")
		ev := span1.Events().AppendEmpty()
		evattrs := ev.Attributes()
		evattrs.PutStr("logStrTag", "logStrValue")
		evattrs.PutInt("logIntTag", 123456789)
		evattrs.PutDouble("logDoubleTag", 123456.789)
		evattrs.PutBool("logBoolTag", false)
		evattrs.PutStr("logJsonTag", "{\"foo\": \"bar\"}")

		return td
	}())
}

func TestReferences(t *testing.T) {
	req := &collectorpb.ReportRequest{
		Reporter: &collectorpb.Reporter{
			Tags: []*collectorpb.KeyValue{
				{
					Key: "lightstep.component_name",
					Value: &collectorpb.KeyValue_StringValue{
						StringValue: "GatewayService",
					},
				},
			},
		},
		Spans: []*collectorpb.Span{
			{
				SpanContext: &collectorpb.SpanContext{
					TraceId: TraceID1,
					SpanId:  SpanID1,
				},
				OperationName: "span1",
			},
			{
				SpanContext: &collectorpb.SpanContext{
					TraceId: TraceID1,
					SpanId:  SpanID2,
				},
				OperationName: "span2",
				References: []*collectorpb.Reference{
					{ // Parent
						Relationship: collectorpb.Reference_CHILD_OF,
						SpanContext: &collectorpb.SpanContext{
							TraceId: TraceID1,
							SpanId:  SpanID1,
						},
					},
				},
			},
			{
				SpanContext: &collectorpb.SpanContext{
					TraceId: TraceID1,
					SpanId:  SpanID3,
				},
				OperationName: "span3",
				References: []*collectorpb.Reference{
					{ // Parent
						Relationship: collectorpb.Reference_CHILD_OF,
						SpanContext: &collectorpb.SpanContext{
							TraceId: TraceID1,
							SpanId:  SpanID2,
						},
					},
					{ // Link
						Relationship: collectorpb.Reference_FOLLOWS_FROM,
						SpanContext: &collectorpb.SpanContext{
							TraceId: TraceID1,
							SpanId:  SpanID1,
						},
					},
					{ // Link outside the Trace
						Relationship: collectorpb.Reference_FOLLOWS_FROM,
						SpanContext: &collectorpb.SpanContext{
							TraceId: TraceID2,
						},
					},
				},
			},
		},
	}
	traces, err := ToTraces(req)
	assert.NoError(t, err)
	assert.Equal(t, traces, func() ptrace.Traces {
		td := ptrace.NewTraces()
		rs := td.ResourceSpans().AppendEmpty()

		r := rs.Resource()
		rattrs := r.Attributes()
		rattrs.PutStr("lightstep.component_name", "GatewayService")
		rattrs.PutStr("service.name", "GatewayService") // derived

		sss := rs.ScopeSpans().AppendEmpty()
		scope := sss.Scope()
		scope.SetName("lightstep-receiver")
		scope.SetVersion("0.0.1")

		spans := sss.Spans()
		span1 := spans.AppendEmpty()
		span1.SetName("span1")
		span1.SetTraceID(UInt64ToTraceID(0, TraceID1))
		span1.SetSpanID(UInt64ToSpanID(SpanID1))

		span2 := spans.AppendEmpty()
		span2.SetName("span2")
		span2.SetTraceID(UInt64ToTraceID(0, TraceID1))
		span2.SetSpanID(UInt64ToSpanID(SpanID2))
		span2.SetParentSpanID(UInt64ToSpanID(SpanID1))

		span3 := spans.AppendEmpty()
		span3.SetName("span3")
		span3.SetTraceID(UInt64ToTraceID(0, TraceID1))
		span3.SetSpanID(UInt64ToSpanID(SpanID3))
		span3.SetParentSpanID(UInt64ToSpanID(SpanID2))
		link := span3.Links().AppendEmpty()
		link.SetTraceID(UInt64ToTraceID(0, TraceID1))
		link.SetSpanID(UInt64ToSpanID(SpanID1))
		link2 := span3.Links().AppendEmpty()
		link2.SetTraceID(UInt64ToTraceID(0, TraceID2))

		return td
	}())
}

// ReportRequest has a TimestampOffsetMicros field
// which needs to be added to all the timestamps here,
// if defined.
func TestTimestampOffset(t *testing.T) {
	duration, _ := time.ParseDuration(fmt.Sprintf("%dus", 743100000))
	offset, _ := time.ParseDuration(fmt.Sprintf("%dus", 13571113))
	start_t := time.Now()
	end_t := start_t.Add(duration)
	req := &collectorpb.ReportRequest{
		Reporter: &collectorpb.Reporter{
			ReporterId: 0000000000001,
			Tags: []*collectorpb.KeyValue{
				{
					Key: "lightstep.component_name",
					Value: &collectorpb.KeyValue_StringValue{
						StringValue: "GatewayService",
					},
				},
			},
		},
		// Important parameter.
		TimestampOffsetMicros: int32(offset.Microseconds()),
		Spans: []*collectorpb.Span{
			{
				OperationName:  "span1",
				StartTimestamp: timestamppb.New(start_t),
				DurationMicros: uint64(duration.Microseconds()),
				Logs: []*collectorpb.Log{
					{
						Timestamp: timestamppb.New(start_t),
						Fields: []*collectorpb.KeyValue{
							{
								Key: "event.name",
								Value: &collectorpb.KeyValue_StringValue{
									StringValue: "requestStarted",
								},
							},
						},
					},
					{
						Timestamp: timestamppb.New(end_t),
						Fields: []*collectorpb.KeyValue{
							{
								Key: "event.name",
								Value: &collectorpb.KeyValue_StringValue{
									StringValue: "requestEnded",
								},
							},
						},
					},
				},
			},
		},
	}
	traces, err := ToTraces(req)
	assert.NoError(t, err)
	assert.Equal(t, traces, func() ptrace.Traces {
		td := ptrace.NewTraces()
		rs := td.ResourceSpans().AppendEmpty()

		r := rs.Resource()
		rattrs := r.Attributes()
		rattrs.PutStr("lightstep.component_name", "GatewayService")
		rattrs.PutStr("service.name", "GatewayService") // derived

		sss := rs.ScopeSpans().AppendEmpty()
		scope := sss.Scope()
		scope.SetName("lightstep-receiver")
		scope.SetVersion("0.0.1")

		spans := sss.Spans()
		span1 := spans.AppendEmpty()
		span1.SetName("span1")
		span1.SetStartTimestamp(pcommon.NewTimestampFromTime(start_t.Add(offset)))
		span1.SetEndTimestamp(pcommon.NewTimestampFromTime(end_t.Add(offset)))
		ev1 := span1.Events().AppendEmpty()
		ev1.SetTimestamp(pcommon.NewTimestampFromTime(start_t.Add(offset)))
		ev1.Attributes().PutStr("event.name", "requestStarted")
		ev2 := span1.Events().AppendEmpty()
		ev2.SetTimestamp(pcommon.NewTimestampFromTime(end_t.Add(offset)))
		ev2.Attributes().PutStr("event.name", "requestEnded")

		return td
	}())
}

func TestTranslateFullSimple(t *testing.T) {
	duration, err := time.ParseDuration(fmt.Sprintf("%dus", 743100000))
	start_t := time.Now()
	end_t := start_t.Add(duration)
	req := &collectorpb.ReportRequest{
		Reporter: &collectorpb.Reporter{
			Tags: []*collectorpb.KeyValue{
				{
					Key: "lightstep.component_name",
					Value: &collectorpb.KeyValue_StringValue{
						StringValue: "GatewayService",
					},
				},
				{
					Key: "custom.id",
					Value: &collectorpb.KeyValue_StringValue{
						StringValue: "ac2ec213-4251-4ca5-93a9-14a94827c51e",
					},
				},
			},
		},
		Spans: []*collectorpb.Span{
			{
				SpanContext: &collectorpb.SpanContext{
					TraceId: TraceID1,
					SpanId:  SpanID1,
				},
				OperationName:  "parent",
				StartTimestamp: timestamppb.New(start_t),
				DurationMicros: uint64(duration.Microseconds()),
				Tags: []*collectorpb.KeyValue{
					{
						Key: "status.code",
						Value: &collectorpb.KeyValue_IntValue{
							IntValue: 201,
						},
					},
				},
				Logs: []*collectorpb.Log{
					{
						Timestamp: timestamppb.New(end_t),
						Fields: []*collectorpb.KeyValue{
							{
								Key: "flush.count",
								Value: &collectorpb.KeyValue_IntValue{
									IntValue: 13,
								},
							},
						},
					},
				},
			},
			{
				SpanContext: &collectorpb.SpanContext{
					TraceId: TraceID1,
					SpanId:  SpanID2,
				},
				OperationName: "child",
				References: []*collectorpb.Reference{
					{ // Parent
						Relationship: collectorpb.Reference_CHILD_OF,
						SpanContext: &collectorpb.SpanContext{
							TraceId: TraceID1,
							SpanId:  SpanID1,
						},
					},
					{ // Link
						Relationship: collectorpb.Reference_FOLLOWS_FROM,
						SpanContext: &collectorpb.SpanContext{
							TraceId: TraceID1,
							SpanId:  SpanID3,
						},
					},
				},
				StartTimestamp: timestamppb.New(start_t),
				DurationMicros: uint64(duration.Microseconds()),
				Tags: []*collectorpb.KeyValue{
					{
						Key: "retry.success",
						Value: &collectorpb.KeyValue_BoolValue{
							BoolValue: true,
						},
					},
				},
				Logs: []*collectorpb.Log{
					{
						Timestamp: timestamppb.New(end_t),
						Fields: []*collectorpb.KeyValue{
							{
								Key: "gc.count",
								Value: &collectorpb.KeyValue_IntValue{
									IntValue: 71,
								},
							},
						},
					},
				},
			},
		},
	}
	traces, err := ToTraces(req)
	assert.NoError(t, err)
	assert.Equal(t, traces, func() ptrace.Traces {
		td := ptrace.NewTraces()
		rs := td.ResourceSpans().AppendEmpty()

		r := rs.Resource()
		rattrs := r.Attributes()
		rattrs.PutStr("lightstep.component_name", "GatewayService")
		rattrs.PutStr("custom.id", "ac2ec213-4251-4ca5-93a9-14a94827c51e")
		rattrs.PutStr("service.name", "GatewayService") // derived

		sss := rs.ScopeSpans().AppendEmpty()
		scope := sss.Scope()
		scope.SetName("lightstep-receiver")
		scope.SetVersion("0.0.1")

		spans := sss.Spans()
		span1 := spans.AppendEmpty()
		span1.SetName("parent")
		span1.SetTraceID(UInt64ToTraceID(0, TraceID1))
		span1.SetSpanID(UInt64ToSpanID(SpanID1))
		span1.SetStartTimestamp(pcommon.NewTimestampFromTime(start_t))
		span1.SetEndTimestamp(pcommon.NewTimestampFromTime(end_t))
		span1.Attributes().PutInt("status.code", 201)
		ev1 := span1.Events().AppendEmpty()
		ev1.SetTimestamp(pcommon.NewTimestampFromTime(end_t))
		ev1.Attributes().PutInt("flush.count", 13)

		span2 := spans.AppendEmpty()
		span2.SetName("child")
		span2.SetTraceID(UInt64ToTraceID(0, TraceID1))
		span2.SetSpanID(UInt64ToSpanID(SpanID2))
		span2.SetParentSpanID(UInt64ToSpanID(SpanID1))
		span2.SetStartTimestamp(pcommon.NewTimestampFromTime(start_t))
		span2.SetEndTimestamp(pcommon.NewTimestampFromTime(end_t))
		span2.Attributes().PutBool("retry.success", true)
		link := span2.Links().AppendEmpty()
		link.SetTraceID(UInt64ToTraceID(0, TraceID1))
		link.SetSpanID(UInt64ToSpanID(SpanID3))
		ev2 := span2.Events().AppendEmpty()
		ev2.SetTimestamp(pcommon.NewTimestampFromTime(end_t))
		ev2.Attributes().PutInt("gc.count", 71)

		return td
	}())
}
