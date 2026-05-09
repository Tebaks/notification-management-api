package mocks

import (
	"context"

	"github.com/kenanabbak/notification-management-api/internal/domain"
	"github.com/stretchr/testify/mock"
)

type TemplateService struct {
	mock.Mock
}

func (m *TemplateService) Create(ctx context.Context, input domain.CreateTemplateInput) (*domain.Template, error) {
	args := m.Called(ctx, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Template), args.Error(1)
}

func (m *TemplateService) GetByID(ctx context.Context, id string) (*domain.Template, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Template), args.Error(1)
}

func (m *TemplateService) List(ctx context.Context) ([]*domain.Template, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Template), args.Error(1)
}

func (m *TemplateService) Delete(ctx context.Context, id string) error {
	return m.Called(ctx, id).Error(0)
}
