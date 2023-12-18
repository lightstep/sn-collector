package osqueryreceiver

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	osquery "github.com/osquery/osquery-go"
	"github.com/osquery/osquery-go/plugin/logger"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/receiver"
)

const (
	typeStr   = "osqueryreceiver"
	stability = component.StabilityLevelAlpha
)

var errInvalidConfig = errors.New("invalid config for osqueryreceiver")

type Config struct {
}

func createDefaultConfig() component.Config {
	return &Config{}
}

type osQueryReceiver struct {
	config       *Config
	logsConsumer consumer.Logs
}

func newLog(body string) plog.Logs {
	ld := plog.NewLogs()
	rl := ld.ResourceLogs().AppendEmpty()
	sl := rl.ScopeLogs().AppendEmpty()
	lr := sl.LogRecords().AppendEmpty()

	resourceAttrs := rl.Resource().Attributes()
	resourceAttrs.PutStr("foo", "bar")
	lr.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))

	// The Message field contains description about the event,
	// which is best suited for the "Body" of the LogRecordSlice.
	lr.Body().SetStr(body)
	return ld
}

func (or *osQueryReceiver) Start(ctx context.Context, _ component.Host) error {
	server, err := osquery.NewExtensionManagerServer("foobar", "/Users/clay.smith/.osquery/shell.em")
	if err != nil {
		log.Fatalf("Error creating extension: %s\n", err)
	}
	server.RegisterPlugin(logger.NewPlugin("example_logger", createLoggerFunction(or.logsConsumer)))
	or.logsConsumer.ConsumeLogs(ctx, newLog("test"))
	return nil
}

func createLoggerFunction(lc consumer.Logs) func(context.Context, logger.LogType, string) error {
	return func(ctx context.Context, typ logger.LogType, logText string) error {
		lc.ConsumeLogs(ctx, newLog(logText))
		//log.Printf("%s: %s\n", typ, logText)
		return nil
	}
}

func LogString(ctx context.Context, typ logger.LogType, logText string) error {
	log.Printf("%s: %s\n", typ, logText)
	return nil
}

func (or *osQueryReceiver) Shutdown(context.Context) error {
	return nil
}

func createLogsReceiver(
	ctx context.Context,
	set receiver.CreateSettings,
	cfg component.Config,
	consumer consumer.Logs,
) (receiver.Logs, error) {
	if err := component.ValidateConfig(cfg); err != nil {
		return nil, fmt.Errorf("cannot configure servicenow logs exporter: %w", err)
	}
	return &osQueryReceiver{
		config:       cfg.(*Config),
		logsConsumer: consumer,
	}, nil
}

func NewFactory() receiver.Factory {
	return receiver.NewFactory(
		typeStr,
		createDefaultConfig,
		receiver.WithLogs(createLogsReceiver, stability),
	)
}
