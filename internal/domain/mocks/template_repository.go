package mocks

import (
	"context"

	"github.com/kenanabbak/notification-management-api/internal/domain"
	"github.com/stretchr/testify/mock"
)

type TemplateRepository struct {
	mock.Mock
}

func (m *TemplateRepository) Create(ctx context.Context, t *domain.Template) error {
	return m.Called(ctx, t).Error(0)
}

func (m *TemplateRepository) GetByID(ctx context.Context, id string) (*domain.Template, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Template), args.Error(1)
}

func (m *TemplateRepository) List(ctx context.Context) ([]*domain.Template, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Template), args.Error(1)
}

func (m *TemplateRepository) Delete(ctx context.Context, id string) error {
	return m.Called(ctx, id).Error(0)
}
