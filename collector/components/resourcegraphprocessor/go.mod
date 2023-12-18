module github.com/lightstep/sn-collector/components/servicenowexporter

go 1.19

require (
	go.opentelemetry.io/collector/component v0.81.0
	go.opentelemetry.io/collector/consumer v0.81.0
	go.opentelemetry.io/collector/pdata v1.0.0-rcv0013
	go.opentelemetry.io/collector/processor v0.81.0
	go.uber.org/zap v1.24.0
    github.com/redis/go-redis/v9 v9.3.0
)