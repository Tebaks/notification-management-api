package queue

import (
	"context"
	"encoding/json"

	"github.com/kenanabbak/notification-management-api/internal/config"
	"github.com/kenanabbak/notification-management-api/internal/domain"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

const (
	queueHigh   = "notifications:high"
	queueNormal = "notifications:normal"
	queueLow    = "notifications:low"
	retryQueue  = "notifications:retry"
)

type queueMessage struct {
	TraceCarrier map[string]string    `json:"tc"`
	Notification *domain.Notification `json:"n"`
}

func NewRedisClient(cfg *config.Config) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
}

type publisher struct {
	rdb        *redis.Client
	propagator propagation.TextMapPropagator
}

func NewPublisher(rdb *redis.Client) domain.QueuePublisher {
	return &publisher{
		rdb:        rdb,
		propagator: otel.GetTextMapPropagator(),
	}
}

func (p *publisher) Enqueue(ctx context.Context, n *domain.Notification) error {
	data, err := marshal(ctx, p.propagator, n)
	if err != nil {
		return err
	}
	return p.rdb.LPush(ctx, queueKey(n.Priority), data).Err()
}

func (p *publisher) EnqueueBatch(ctx context.Context, notifications []*domain.Notification) error {
	pipe := p.rdb.Pipeline()
	for _, n := range notifications {
		data, err := marshal(ctx, p.propagator, n)
		if err != nil {
			return err
		}
		pipe.LPush(ctx, queueKey(n.Priority), data)
	}
	_, err := pipe.Exec(ctx)
	return err
}

func marshal(ctx context.Context, prop propagation.TextMapPropagator, n *domain.Notification) ([]byte, error) {
	carrier := propagation.MapCarrier{}
	prop.Inject(ctx, carrier)
	return json.Marshal(queueMessage{TraceCarrier: carrier, Notification: n})
}

func unmarshal(data []byte) (*queueMessage, error) {
	var msg queueMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

func queueKey(priority domain.Priority) string {
	switch priority {
	case domain.PriorityHigh:
		return queueHigh
	case domain.PriorityLow:
		return queueLow
	default:
		return queueNormal
	}
}

func priorityQueues() []string {
	return []string{queueHigh, queueNormal, queueLow}
}
