package queue

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"golang.org/x/time/rate"

	"github.com/kenanabbak/notification-management-api/internal/config"
	"github.com/kenanabbak/notification-management-api/internal/domain"
	"github.com/kenanabbak/notification-management-api/internal/metrics"
)

const tracerName = "notification.worker"

type Worker struct {
	rdb        *redis.Client
	repo       domain.NotificationRepository
	cfg        *config.Config
	log        *zap.Logger
	metrics    *metrics.Collector
	tracer     trace.Tracer
	propagator propagation.TextMapPropagator
	limiters   map[domain.Channel]*rate.Limiter
	client     *http.Client
	notifier   domain.StatusNotifier
	cancel     context.CancelFunc
	wg         sync.WaitGroup
}

type webhookRequest struct {
	To      string `json:"to"`
	Channel string `json:"channel"`
	Content string `json:"content"`
}

type webhookResponse struct {
	MessageID string `json:"messageId"`
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
}

func NewWorker(rdb *redis.Client, repo domain.NotificationRepository, cfg *config.Config, log *zap.Logger, m *metrics.Collector, notifier domain.StatusNotifier) *Worker {
	limiters := map[domain.Channel]*rate.Limiter{
		domain.ChannelSMS:   rate.NewLimiter(rate.Limit(cfg.Worker.RateLimitPerSec), cfg.Worker.RateLimitPerSec),
		domain.ChannelEmail: rate.NewLimiter(rate.Limit(cfg.Worker.RateLimitPerSec), cfg.Worker.RateLimitPerSec),
		domain.ChannelPush:  rate.NewLimiter(rate.Limit(cfg.Worker.RateLimitPerSec), cfg.Worker.RateLimitPerSec),
	}
	return &Worker{
		rdb:        rdb,
		repo:       repo,
		cfg:        cfg,
		log:        log,
		metrics:    m,
		tracer:     otel.Tracer(tracerName),
		propagator: otel.GetTextMapPropagator(),
		limiters:   limiters,
		client:     &http.Client{Timeout: cfg.Webhook.Timeout},
		notifier:   notifier,
	}
}

func RegisterWorkerLifecycle(lc fx.Lifecycle, w *Worker) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			workerCtx, cancel := context.WithCancel(context.Background())
			w.cancel = cancel
			for i := 0; i < w.cfg.Worker.Concurrency; i++ {
				w.wg.Add(1)
				go w.run(workerCtx)
			}
			return nil
		},
		OnStop: func(ctx context.Context) error {
			w.log.Info("stopping workers, draining in-flight notifications")
			w.cancel()
			w.wg.Wait()
			w.log.Info("all workers stopped")
			return nil
		},
	})
}

func (w *Worker) run(ctx context.Context) {
	defer w.wg.Done()
	queues := priorityQueues()
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		result, err := w.rdb.BRPop(ctx, time.Second, queues...).Result()
		if err != nil {
			continue
		}

		msg, err := unmarshal([]byte(result[1]))
		if err != nil {
			w.log.Error("failed to unmarshal queue message", zap.Error(err))
			continue
		}

		carrier := propagation.MapCarrier(msg.TraceCarrier)
		parentCtx := w.propagator.Extract(context.Background(), carrier)

		w.process(parentCtx, msg.Notification)
	}
}

func (w *Worker) process(ctx context.Context, n *domain.Notification) {
	ctx, span := w.tracer.Start(ctx, "worker.process",
		trace.WithAttributes(
			attribute.String("notification.id", n.ID),
			attribute.String("notification.channel", string(n.Channel)),
			attribute.String("notification.priority", string(n.Priority)),
		),
	)
	defer span.End()

	limiter, ok := w.limiters[n.Channel]
	if ok {
		if err := limiter.Wait(ctx); err != nil {
			return
		}
	}

	if err := w.repo.UpdateStatus(ctx, n.ID, domain.StatusProcessing, nil); err != nil {
		w.log.Error("failed to update status to processing", zap.String("id", n.ID), zap.Error(err))
	}

	start := time.Now()
	msgID, err := w.deliver(ctx, n)
	latency := time.Since(start)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		w.metrics.RecordFailed(n.Channel)
		w.handleFailure(ctx, n, err)
		return
	}

	span.SetStatus(codes.Ok, "")
	span.SetAttributes(attribute.String("provider.message_id", msgID))
	w.metrics.RecordDelivered(n.Channel, latency)

	if err := w.repo.UpdateStatus(ctx, n.ID, domain.StatusDelivered, &msgID); err != nil {
		w.log.Error("failed to update status to delivered", zap.String("id", n.ID), zap.Error(err))
	}
	w.notifier.Notify(n.ID, domain.StatusDelivered)

	w.log.Info("notification delivered",
		zap.String("id", n.ID),
		zap.String("channel", string(n.Channel)),
		zap.String("provider_msg_id", msgID),
		zap.Duration("latency", latency),
		zap.String("trace_id", span.SpanContext().TraceID().String()),
	)
}

func (w *Worker) deliver(ctx context.Context, n *domain.Notification) (string, error) {
	ctx, span := w.tracer.Start(ctx, "webhook.deliver",
		trace.WithAttributes(
			attribute.String("notification.id", n.ID),
			attribute.String("http.url", w.cfg.Webhook.URL),
		),
	)
	defer span.End()

	payload, err := json.Marshal(webhookRequest{
		To:      n.Recipient,
		Channel: string(n.Channel),
		Content: n.Content,
	})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, w.cfg.Webhook.URL, bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := w.client.Do(req)
	if err != nil {
		span.RecordError(err)
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read provider response: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		err := fmt.Errorf("provider returned %d: %s", resp.StatusCode, string(body))
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return "", err
	}

	var webhookResp webhookResponse
	if err := json.Unmarshal(body, &webhookResp); err != nil {
		return "", err
	}

	span.SetAttributes(attribute.String("provider.message_id", webhookResp.MessageID))
	return webhookResp.MessageID, nil
}

func (w *Worker) handleFailure(ctx context.Context, n *domain.Notification, deliveryErr error) {
	w.log.Warn("delivery failed",
		zap.String("id", n.ID),
		zap.Int("retry_count", n.RetryCount),
		zap.Error(deliveryErr),
		zap.String("trace_id", trace.SpanFromContext(ctx).SpanContext().TraceID().String()),
	)

	if n.RetryCount >= w.cfg.Worker.MaxRetries {
		if err := w.repo.UpdateStatus(ctx, n.ID, domain.StatusFailed, nil); err != nil {
			w.log.Error("failed to mark as failed", zap.String("id", n.ID), zap.Error(err))
		}
		w.notifier.Notify(n.ID, domain.StatusFailed)
		return
	}

	if err := w.repo.IncrementRetry(ctx, n.ID); err != nil {
		w.log.Error("failed to increment retry", zap.String("id", n.ID), zap.Error(err))
	}

	delay := time.Duration(math.Pow(2, float64(n.RetryCount))) * w.cfg.Worker.RetryBaseDelay
	deliverAt := time.Now().Add(delay).Unix()
	n.RetryCount++

	if err := w.repo.UpdateStatus(ctx, n.ID, domain.StatusQueued, nil); err != nil {
		w.log.Error("failed to update status for retry", zap.String("id", n.ID), zap.Error(err))
		return
	}

	data, err := marshal(ctx, w.propagator, n)
	if err != nil {
		w.log.Error("failed to marshal retry message", zap.String("id", n.ID), zap.Error(err))
		return
	}

	if err := w.rdb.ZAdd(ctx, retryQueue, redis.Z{
		Score:  float64(deliverAt),
		Member: string(data),
	}).Err(); err != nil {
		w.log.Error("failed to schedule retry", zap.String("id", n.ID), zap.Error(err))
	}

	w.log.Info("notification scheduled for retry",
		zap.String("id", n.ID),
		zap.Duration("delay", delay),
	)
}
