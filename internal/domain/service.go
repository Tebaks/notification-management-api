package domain

import "context"

type NotificationService interface {
	Create(ctx context.Context, input CreateNotificationInput) (*Notification, error)
	CreateBatch(ctx context.Context, input CreateBatchInput) (*BatchResult, error)
	GetByID(ctx context.Context, id string) (*Notification, error)
	Cancel(ctx context.Context, id string) error
	List(ctx context.Context, filter ListFilter) ([]*Notification, int, error)
}

type QueuePublisher interface {
	Enqueue(ctx context.Context, n *Notification) error
	EnqueueBatch(ctx context.Context, notifications []*Notification) error
}

type StatusNotifier interface {
	Notify(notificationID string, status Status)
}
