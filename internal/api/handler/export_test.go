package handler

import (
	"context"

	appmetrics "github.com/kenanabbak/notification-management-api/internal/metrics"
)

func NewHealthHandlerForTest(pingDB, pingRedis func(context.Context) error) *HealthHandler {
	return &HealthHandler{pingDB: pingDB, pingRedis: pingRedis}
}

func NewMetricsHandlerForTest(queues QueueDepthReader, collector *appmetrics.Collector) *MetricsHandler {
	return &MetricsHandler{queues: queues, collector: collector}
}
