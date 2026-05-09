package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	appmetrics "github.com/kenanabbak/notification-management-api/internal/metrics"
)

type QueueDepthReader interface {
	QueueDepths(ctx context.Context) (high, normal, low int64)
}

type redisQueueReader struct{ rdb *redis.Client }

func (r *redisQueueReader) QueueDepths(ctx context.Context) (int64, int64, int64) {
	high, _ := r.rdb.LLen(ctx, "notifications:high").Result()
	normal, _ := r.rdb.LLen(ctx, "notifications:normal").Result()
	low, _ := r.rdb.LLen(ctx, "notifications:low").Result()
	return high, normal, low
}

type MetricsHandler struct {
	queues    QueueDepthReader
	collector *appmetrics.Collector
}

func NewMetricsHandler(rdb *redis.Client, collector *appmetrics.Collector) *MetricsHandler {
	return &MetricsHandler{queues: &redisQueueReader{rdb: rdb}, collector: collector}
}

// Metrics godoc
// @Summary     Get real-time metrics
// @Tags        observability
// @Produce     json
// @Success     200  {object}  map[string]any
// @Router      /metrics [get]
func (h *MetricsHandler) Metrics(c *gin.Context) {
	high, normal, low := h.queues.QueueDepths(c.Request.Context())
	snap := h.collector.Snapshot()

	c.JSON(http.StatusOK, gin.H{
		"queue_depth": gin.H{
			"high":   high,
			"normal": normal,
			"low":    low,
			"total":  high + normal + low,
		},
		"delivery": snap,
	})
}
