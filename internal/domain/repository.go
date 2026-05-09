package domain

import (
	"context"
	"time"
)

type NotificationRepository interface {
	Create(ctx context.Context, n *Notification) error
	CreateBatch(ctx context.Context, notifications []*Notification) error
	GetByID(ctx context.Context, id string) (*Notification, error)
	GetByIdempotencyKey(ctx context.Context, key string) (*Notification, error)
	UpdateStatus(ctx context.Context, id string, status Status, providerMsgID *string) error
	IncrementRetry(ctx context.Context, id string) error
	Cancel(ctx context.Context, id string) error
	List(ctx context.Context, filter ListFilter) ([]*Notification, int, error)
	GetDueScheduled(ctx context.Context, limit int) ([]*Notification, error)
	BulkUpdateStatus(ctx context.Context, ids []string, status Status) error
	Archive(ctx context.Context, olderThan time.Time, batchSize int) (int64, error)
}
