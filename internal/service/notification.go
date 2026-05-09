package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/kenanabbak/notification-management-api/internal/domain"
)

type notificationService struct {
	repo      domain.NotificationRepository
	publisher domain.QueuePublisher
	tmplSvc   domain.TemplateService
	log       *zap.Logger
}

func NewNotificationService(repo domain.NotificationRepository, publisher domain.QueuePublisher, tmplSvc domain.TemplateService, log *zap.Logger) domain.NotificationService {
	return &notificationService{repo: repo, publisher: publisher, tmplSvc: tmplSvc, log: log}
}

func (s *notificationService) Create(ctx context.Context, input domain.CreateNotificationInput) (*domain.Notification, error) {
	if err := domain.ValidateRecipient(input.Channel, input.Recipient); err != nil {
		return nil, err
	}

	if input.TemplateID != nil {
		tmpl, err := s.tmplSvc.GetByID(ctx, *input.TemplateID)
		if err != nil {
			return nil, err
		}
		input.Content = RenderTemplate(tmpl.Body, input.Variables)
	}

	if input.Content == "" {
		return nil, domain.ErrContentRequired
	}

	if err := domain.ValidateContent(input.Channel, input.Content); err != nil {
		return nil, err
	}

	if input.IdempotencyKey != nil {
		existing, err := s.repo.GetByIdempotencyKey(ctx, *input.IdempotencyKey)
		if err == nil {
			return existing, nil
		}
		if err != domain.ErrNotFound {
			return nil, err
		}
	}

	priority := input.Priority
	if priority == "" {
		priority = domain.PriorityNormal
	}

	status := domain.StatusQueued
	if input.ScheduledAt != nil {
		status = domain.StatusPending
	}

	n := &domain.Notification{
		ID:             uuid.NewString(),
		Recipient:      input.Recipient,
		Channel:        input.Channel,
		Content:        input.Content,
		Priority:       priority,
		Status:         status,
		IdempotencyKey: input.IdempotencyKey,
		ScheduledAt:    input.ScheduledAt,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := s.repo.Create(ctx, n); err != nil {
		return nil, err
	}

	if status == domain.StatusQueued {
		if err := s.publisher.Enqueue(ctx, n); err != nil {
			s.log.Error("failed to enqueue notification", zap.String("id", n.ID), zap.Error(err))
		}
	}

	return n, nil
}

func (s *notificationService) CreateBatch(ctx context.Context, input domain.CreateBatchInput) (*domain.BatchResult, error) {
	batchID := uuid.NewString()
	now := time.Now()
	notifications := make([]*domain.Notification, 0, len(input.Notifications))

	for _, item := range input.Notifications {
		if err := domain.ValidateRecipient(item.Channel, item.Recipient); err != nil {
			return nil, err
		}

		if item.TemplateID != nil {
			tmpl, err := s.tmplSvc.GetByID(ctx, *item.TemplateID)
			if err != nil {
				return nil, err
			}
			item.Content = RenderTemplate(tmpl.Body, item.Variables)
		}

		if item.Content == "" {
			return nil, domain.ErrContentRequired
		}

		if err := domain.ValidateContent(item.Channel, item.Content); err != nil {
			return nil, err
		}

		priority := item.Priority
		if priority == "" {
			priority = domain.PriorityNormal
		}
		status := domain.StatusQueued
		if item.ScheduledAt != nil {
			status = domain.StatusPending
		}
		n := &domain.Notification{
			ID:             uuid.NewString(),
			BatchID:        &batchID,
			Recipient:      item.Recipient,
			Channel:        item.Channel,
			Content:        item.Content,
			Priority:       priority,
			Status:         status,
			IdempotencyKey: item.IdempotencyKey,
			ScheduledAt:    item.ScheduledAt,
			CreatedAt:      now,
			UpdatedAt:      now,
		}
		notifications = append(notifications, n)
	}

	if err := s.repo.CreateBatch(ctx, notifications); err != nil {
		return nil, err
	}

	toQueue := make([]*domain.Notification, 0)
	for _, n := range notifications {
		if n.Status == domain.StatusQueued {
			toQueue = append(toQueue, n)
		}
	}
	if len(toQueue) > 0 {
		if err := s.publisher.EnqueueBatch(ctx, toQueue); err != nil {
			s.log.Error("failed to enqueue batch", zap.String("batch_id", batchID), zap.Error(err))
		}
	}

	return &domain.BatchResult{
		BatchID: batchID,
		Total:   len(notifications),
		Queued:  len(toQueue),
	}, nil
}

func (s *notificationService) GetByID(ctx context.Context, id string) (*domain.Notification, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *notificationService) Cancel(ctx context.Context, id string) error {
	return s.repo.Cancel(ctx, id)
}

func (s *notificationService) List(ctx context.Context, filter domain.ListFilter) ([]*domain.Notification, int, error) {
	return s.repo.List(ctx, filter)
}
