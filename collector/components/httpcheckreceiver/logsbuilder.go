// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package httpcheckreceiver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/httpcheckreceiver"

import (
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/receiver"
)

type LogsBuilder struct {
	logBuffer  plog.Logs
	logRecords plog.LogRecordSlice
	buildInfo  component.BuildInfo
}

func NewLogsBuilder(settings receiver.CreateSettings) *LogsBuilder {
	lb := &LogsBuilder{
		buildInfo: settings.BuildInfo,
	}
	lb.reset()
	return lb
}

func (lb *LogsBuilder) reset() {
	out := plog.NewLogs()
	logs := out.ResourceLogs()
	rls := logs.AppendEmpty()

	ills := rls.ScopeLogs().AppendEmpty()
	ills.Scope().SetName("otelcol/httpcheckreceiver")
	ills.Scope().SetVersion(lb.buildInfo.Version)

	lb.logBuffer = out
	lb.logRecords = ills.LogRecords()
}

func (lb *LogsBuilder) RecordResponse(
	endpoint string,
	body string,
	statusCode int,
	timestamp pcommon.Timestamp,
	elapsed time.Duration) {

	lr := lb.logRecords.AppendEmpty()
	lr.SetTimestamp(timestamp)
	lr.Body().SetStr(body)
	attrs := lr.Attributes()
	attrs.PutStr("http.url", endpoint)
	attrs.PutInt("http.status_code", int64(statusCode))
	attrs.PutInt("http.client.request.duration", elapsed.Milliseconds())
}

func (lb *LogsBuilder) Emit() plog.Logs {
	logs := lb.logBuffer
	lb.reset()
	return logs
}
