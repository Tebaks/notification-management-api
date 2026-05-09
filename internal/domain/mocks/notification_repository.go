package mocks

import (
	"context"
	"time"

	"github.com/kenanabbak/notification-management-api/internal/domain"
	"github.com/stretchr/testify/mock"
)

type NotificationRepository struct {
	mock.Mock
}

func (m *NotificationRepository) Create(ctx context.Context, n *domain.Notification) error {
	return m.Called(ctx, n).Error(0)
}

func (m *NotificationRepository) CreateBatch(ctx context.Context, notifications []*domain.Notification) error {
	return m.Called(ctx, notifications).Error(0)
}

func (m *NotificationRepository) GetByID(ctx context.Context, id string) (*domain.Notification, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Notification), args.Error(1)
}

func (m *NotificationRepository) GetByIdempotencyKey(ctx context.Context, key string) (*domain.Notification, error) {
	args := m.Called(ctx, key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Notification), args.Error(1)
}

func (m *NotificationRepository) UpdateStatus(ctx context.Context, id string, status domain.Status, providerMsgID *string) error {
	return m.Called(ctx, id, status, providerMsgID).Error(0)
}

func (m *NotificationRepository) IncrementRetry(ctx context.Context, id string) error {
	return m.Called(ctx, id).Error(0)
}

func (m *NotificationRepository) Cancel(ctx context.Context, id string) error {
	return m.Called(ctx, id).Error(0)
}

func (m *NotificationRepository) List(ctx context.Context, filter domain.ListFilter) ([]*domain.Notification, int, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*domain.Notification), args.Int(1), args.Error(2)
}

func (m *NotificationRepository) GetDueScheduled(ctx context.Context, limit int) ([]*domain.Notification, error) {
	args := m.Called(ctx, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Notification), args.Error(1)
}

func (m *NotificationRepository) BulkUpdateStatus(ctx context.Context, ids []string, status domain.Status) error {
	return m.Called(ctx, ids, status).Error(0)
}

func (m *NotificationRepository) Archive(ctx context.Context, olderThan time.Time, batchSize int) (int64, error) {
	args := m.Called(ctx, olderThan, batchSize)
	return args.Get(0).(int64), args.Error(1)
}
