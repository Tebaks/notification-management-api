package service

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/kenanabbak/notification-management-api/internal/domain"
	"go.uber.org/zap"
)

type templateService struct {
	repo domain.TemplateRepository
	log  *zap.Logger
}

func NewTemplateService(repo domain.TemplateRepository, log *zap.Logger) domain.TemplateService {
	return &templateService{repo: repo, log: log}
}

func (s *templateService) Create(ctx context.Context, input domain.CreateTemplateInput) (*domain.Template, error) {
	t := &domain.Template{
		ID:        uuid.NewString(),
		Name:      input.Name,
		Channel:   input.Channel,
		Body:      input.Body,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := s.repo.Create(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}

func (s *templateService) GetByID(ctx context.Context, id string) (*domain.Template, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *templateService) List(ctx context.Context) ([]*domain.Template, error) {
	return s.repo.List(ctx)
}

func (s *templateService) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

// RenderTemplate replaces {{key}} placeholders in body with values from variables.
func RenderTemplate(body string, variables map[string]string) string {
	if len(variables) == 0 {
		return body
	}
	pairs := make([]string, 0, len(variables)*2)
	for k, v := range variables {
		pairs = append(pairs, "{{"+k+"}}", v)
	}
	return strings.NewReplacer(pairs...).Replace(body)
}
