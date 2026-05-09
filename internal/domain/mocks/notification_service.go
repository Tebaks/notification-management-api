package mocks

import (
	"context"

	"github.com/kenanabbak/notification-management-api/internal/domain"
	"github.com/stretchr/testify/mock"
)

type NotificationService struct {
	mock.Mock
}

func (m *NotificationService) Create(ctx context.Context, input domain.CreateNotificationInput) (*domain.Notification, error) {
	args := m.Called(ctx, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Notification), args.Error(1)
}

func (m *NotificationService) CreateBatch(ctx context.Context, input domain.CreateBatchInput) (*domain.BatchResult, error) {
	args := m.Called(ctx, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.BatchResult), args.Error(1)
}

func (m *NotificationService) GetByID(ctx context.Context, id string) (*domain.Notification, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Notification), args.Error(1)
}

func (m *NotificationService) Cancel(ctx context.Context, id string) error {
	return m.Called(ctx, id).Error(0)
}

func (m *NotificationService) List(ctx context.Context, filter domain.ListFilter) ([]*domain.Notification, int, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*domain.Notification), args.Int(1), args.Error(2)
}
