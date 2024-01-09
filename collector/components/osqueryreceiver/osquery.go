package osqueryreceiver

import (
	"context"
	"time"

	osquery "github.com/osquery/osquery-go"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap"
)

type osQueryReceiver struct {
	config       *Config
	logger       *zap.Logger
	client       *osquery.ExtensionManagerClient
	logsConsumer consumer.Logs
}

func newLog(ld plog.Logs, query string, row map[string]string) plog.Logs {
	//ld := plog.NewLogs()
	rl := ld.ResourceLogs().AppendEmpty()
	sl := rl.ScopeLogs().AppendEmpty()
	lr := sl.LogRecords().AppendEmpty()

	resourceAttrs := rl.Resource().Attributes()
	resourceAttrs.PutStr("instrumentation.name", "otelcol/osqueryreciever")
	for k, v := range row {
		resourceAttrs.PutStr(k, v)
	}
	lr.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	lr.SetSeverityNumber(plog.SeverityNumberInfo)
	lr.SetSeverityText("INFO")
	lr.Body().SetStr(query)
	return ld
}

func (or *osQueryReceiver) runQuery(ctx context.Context, query string) error {
	rows, err := or.client.QueryRows(query)
	if err != nil {
		or.logger.Error("Error running query", zap.Error(err))
	}
	ld := plog.NewLogs()
	for _, row := range rows {
		newLog(ld, query, row)
	}
	return or.logsConsumer.ConsumeLogs(ctx, ld)
}

func (or *osQueryReceiver) Start(ctx context.Context, _ component.Host) error {

	client, err := osquery.NewClient(or.config.ExtensionsSocket, 10*time.Second)
	if err != nil {
		return err
	}
	or.client = client

	ticker := time.NewTicker(15 * time.Second)
	quit := make(chan struct{})
	go func() {
		for {
			select {
			case <-ticker.C:
				for _, query := range or.config.Queries {
					go or.runQuery(ctx, query)
				}
			case <-quit:
				ticker.Stop()
				return
			}
		}
	}()

	//or.logsConsumer.ConsumeLogs(ctx, newLog("test"))
	return nil
}

func (or *osQueryReceiver) Shutdown(context.Context) error {
	if or.client != nil {
		or.client.Close()
		return nil
	}
	return nil
}
