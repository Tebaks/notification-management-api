package mocks

import (
	"context"

	"github.com/kenanabbak/notification-management-api/internal/domain"
	"github.com/stretchr/testify/mock"
)

type QueuePublisher struct {
	mock.Mock
}

func (m *QueuePublisher) Enqueue(ctx context.Context, n *domain.Notification) error {
	return m.Called(ctx, n).Error(0)
}

func (m *QueuePublisher) EnqueueBatch(ctx context.Context, notifications []*domain.Notification) error {
	return m.Called(ctx, notifications).Error(0)
}
