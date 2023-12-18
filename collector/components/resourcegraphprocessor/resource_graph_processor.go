package resourcegraphprocessor

import (
	"context"

	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"

	"go.opentelemetry.io/collector/processor"
	"go.uber.org/zap"
)

type resourceGraphProcessor struct {
	logger *zap.Logger
	client *redis.Client
}

func newResourceGraphProcessor(cfg *Config, set processor.CreateSettings) (*resourceGraphProcessor, error) {
	opts := &redis.Options{
		Addr:     cfg.Endpoint,
		Username: cfg.Username,
		Password: string(cfg.Password),
		Network:  cfg.Transport,
	}

	var err error
	if opts.TLSConfig, err = cfg.TLS.LoadTLSConfig(); err != nil {
		return nil, err
	}

	return &resourceGraphProcessor{
		logger: set.Logger,
		client: redis.NewClient(opts),
	}, nil
}

// redis sets
// resources:project-id
// services:project-id
// hosts:project-id
// service-dependencies:project-id
// k8s-clusters:project-id
// k8s-workloads:project-id
// k8s-namespaces:project-id
// k8s-nodes:project-id
// k8s-pods:project-id
// k8s-containers:project-id
// software:project-id

func (rpg *resourceGraphProcessor) processLogs(ctx context.Context, ld plog.Logs) (plog.Logs, error) {
	return ld, nil
}

func (rpg *resourceGraphProcessor) processMetrics(ctx context.Context, md pmetric.Metrics) (pmetric.Metrics, error) {
	// Batches redis commands into a single request
	pipe := rpg.client.Pipeline()
	for i := 0; i < md.ResourceMetrics().Len(); i++ {
		rm := md.ResourceMetrics().At(i)
		v, exists := rm.Resource().Attributes().Get("service.name")
		if exists {
			rpg.logger.Info("found service name", zap.String("value", v.AsString()))
			rpg.client.SAdd(ctx, "resources:project-id", "services:project-id")
			rpg.client.SAdd(ctx, "services:project-id", v.AsString())
		}

		v, exists = rm.Resource().Attributes().Get("host.name")
		if exists {
			rpg.logger.Info("found host name", zap.String("value", v.AsString()))
			rpg.client.SAdd(ctx, "resources:project-id", "hosts:project-id")
			rpg.client.SAdd(ctx, "hosts:project-id", v.AsString())
		}
	}
	_, err := pipe.Exec(ctx)
	if err != nil {
		return md, err
	}

	return md, nil
}

func (rpg *resourceGraphProcessor) Start(ctx context.Context, host component.Host) error {
	status := rpg.client.Ping(ctx)
	if status.Err() != nil {
		rpg.logger.Error("could not connect to redis", zap.Any("status", status))
	} else {
		rpg.logger.Info("connected to redis", zap.Any("status", status))
	}
	return nil
}

func (rpg *resourceGraphProcessor) Shutdown(_ context.Context) error {
	if rpg.client != nil {
		return rpg.client.Close()
	}
	return nil
}
