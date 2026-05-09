package service_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	"github.com/kenanabbak/notification-management-api/internal/domain"
	"github.com/kenanabbak/notification-management-api/internal/domain/mocks"
	"github.com/kenanabbak/notification-management-api/internal/service"
)

type NotificationServiceSuite struct {
	suite.Suite
	repo *mocks.NotificationRepository
	pub  *mocks.QueuePublisher
	tmpl *mocks.TemplateService
	svc  domain.NotificationService
}

func TestNotificationServiceSuite(t *testing.T) {
	suite.Run(t, new(NotificationServiceSuite))
}

func (s *NotificationServiceSuite) SetupTest() {
	s.repo = &mocks.NotificationRepository{}
	s.pub = &mocks.QueuePublisher{}
	s.tmpl = &mocks.TemplateService{}
	s.svc = service.NewNotificationService(s.repo, s.pub, s.tmpl, zap.NewNop())
}

func (s *NotificationServiceSuite) TearDownTest() {
	s.repo.AssertExpectations(s.T())
	s.pub.AssertExpectations(s.T())
	s.tmpl.AssertExpectations(s.T())
}

// --- Create ---

func (s *NotificationServiceSuite) TestCreate_Success() {
	input := domain.CreateNotificationInput{
		Recipient: "+905551234567",
		Channel:   domain.ChannelSMS,
		Content:   "Hello",
	}
	s.repo.On("GetByIdempotencyKey", mock.Anything, mock.Anything).Return(nil, domain.ErrNotFound).Maybe()
	s.repo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Notification")).Return(nil)
	s.pub.On("Enqueue", mock.Anything, mock.AnythingOfType("*domain.Notification")).Return(nil)

	n, err := s.svc.Create(context.Background(), input)

	s.NoError(err)
	s.NotEmpty(n.ID)
	s.Equal(domain.ChannelSMS, n.Channel)
	s.Equal(domain.PriorityNormal, n.Priority)
	s.Equal(domain.StatusQueued, n.Status)
}

func (s *NotificationServiceSuite) TestCreate_Idempotency_ReturnsExisting() {
	key := "my-key"
	existing := &domain.Notification{ID: "existing-id", Status: domain.StatusDelivered}
	input := domain.CreateNotificationInput{
		Recipient:      "+905551234567",
		Channel:        domain.ChannelSMS,
		Content:        "Hello",
		IdempotencyKey: &key,
	}
	s.repo.On("GetByIdempotencyKey", mock.Anything, key).Return(existing, nil)

	n, err := s.svc.Create(context.Background(), input)

	s.NoError(err)
	s.Equal("existing-id", n.ID)
	s.pub.AssertNotCalled(s.T(), "Enqueue")
}

func (s *NotificationServiceSuite) TestCreate_IdempotencyKey_LookupError() {
	key := "my-key"
	dbErr := errors.New("db unavailable")
	input := domain.CreateNotificationInput{
		Recipient:      "+905551234567",
		Channel:        domain.ChannelSMS,
		Content:        "Hello",
		IdempotencyKey: &key,
	}
	s.repo.On("GetByIdempotencyKey", mock.Anything, key).Return(nil, dbErr)

	_, err := s.svc.Create(context.Background(), input)

	s.ErrorIs(err, dbErr)
	s.repo.AssertNotCalled(s.T(), "Create")
}

func (s *NotificationServiceSuite) TestCreate_Scheduled_StatusIsPending() {
	future := time.Now().Add(time.Hour)
	input := domain.CreateNotificationInput{
		Recipient:   "user@example.com",
		Channel:     domain.ChannelEmail,
		Content:     "Hello",
		ScheduledAt: &future,
	}
	s.repo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Notification")).Return(nil)

	n, err := s.svc.Create(context.Background(), input)

	s.NoError(err)
	s.Equal(domain.StatusPending, n.Status)
	s.pub.AssertNotCalled(s.T(), "Enqueue")
}

func (s *NotificationServiceSuite) TestCreate_WithTemplate() {
	tmplID := "tmpl-1"
	tmpl := &domain.Template{ID: tmplID, Channel: domain.ChannelSMS, Body: "Hello {{name}}!"}
	input := domain.CreateNotificationInput{
		Recipient:  "+905551234567",
		Channel:    domain.ChannelSMS,
		TemplateID: &tmplID,
		Variables:  map[string]string{"name": "Kenan"},
	}
	s.tmpl.On("GetByID", mock.Anything, tmplID).Return(tmpl, nil)
	s.repo.On("GetByIdempotencyKey", mock.Anything, mock.Anything).Return(nil, domain.ErrNotFound).Maybe()
	s.repo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Notification")).Return(nil)
	s.pub.On("Enqueue", mock.Anything, mock.AnythingOfType("*domain.Notification")).Return(nil)

	n, err := s.svc.Create(context.Background(), input)

	s.NoError(err)
	s.Equal("Hello Kenan!", n.Content)
}

func (s *NotificationServiceSuite) TestCreate_TemplateNotFound() {
	tmplID := "missing"
	input := domain.CreateNotificationInput{
		Recipient:  "+905551234567",
		Channel:    domain.ChannelSMS,
		TemplateID: &tmplID,
	}
	s.tmpl.On("GetByID", mock.Anything, tmplID).Return(nil, domain.ErrTemplateNotFound)

	_, err := s.svc.Create(context.Background(), input)

	s.ErrorIs(err, domain.ErrTemplateNotFound)
	s.repo.AssertNotCalled(s.T(), "Create")
}

func (s *NotificationServiceSuite) TestCreate_ContentRequired() {
	input := domain.CreateNotificationInput{
		Recipient: "+905551234567",
		Channel:   domain.ChannelSMS,
	}

	_, err := s.svc.Create(context.Background(), input)

	s.ErrorIs(err, domain.ErrContentRequired)
	s.repo.AssertNotCalled(s.T(), "Create")
}

func (s *NotificationServiceSuite) TestCreate_ContentTooLong() {
	input := domain.CreateNotificationInput{
		Recipient: "+905551234567",
		Channel:   domain.ChannelSMS,
		Content:   strings.Repeat("x", 161),
	}

	_, err := s.svc.Create(context.Background(), input)

	s.ErrorIs(err, domain.ErrContentTooLong)
	s.repo.AssertNotCalled(s.T(), "Create")
}

func (s *NotificationServiceSuite) TestCreate_InvalidRecipient() {
	input := domain.CreateNotificationInput{
		Recipient: "not-a-phone",
		Channel:   domain.ChannelSMS,
		Content:   "Hello",
	}

	_, err := s.svc.Create(context.Background(), input)

	s.ErrorIs(err, domain.ErrInvalidRecipient)
	s.repo.AssertNotCalled(s.T(), "Create")
}

func (s *NotificationServiceSuite) TestCreate_RepoError() {
	dbErr := errors.New("db unavailable")
	input := domain.CreateNotificationInput{
		Recipient: "+905551234567",
		Channel:   domain.ChannelSMS,
		Content:   "Hello",
	}
	s.repo.On("GetByIdempotencyKey", mock.Anything, mock.Anything).Return(nil, domain.ErrNotFound).Maybe()
	s.repo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Notification")).Return(dbErr)

	_, err := s.svc.Create(context.Background(), input)

	s.ErrorIs(err, dbErr)
}

func (s *NotificationServiceSuite) TestCreate_EnqueueError() {
	input := domain.CreateNotificationInput{
		Recipient: "+905551234567",
		Channel:   domain.ChannelSMS,
		Content:   "Hello",
	}
	s.repo.On("GetByIdempotencyKey", mock.Anything, mock.Anything).Return(nil, domain.ErrNotFound).Maybe()
	s.repo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Notification")).Return(nil)
	s.pub.On("Enqueue", mock.Anything, mock.AnythingOfType("*domain.Notification")).
		Return(errors.New("queue unavailable"))

	n, err := s.svc.Create(context.Background(), input)

	s.NoError(err)
	s.NotNil(n)
}

// --- CreateBatch ---

func (s *NotificationServiceSuite) TestCreateBatch_Success() {
	input := domain.CreateBatchInput{
		Notifications: []domain.CreateNotificationInput{
			{Recipient: "+905551234567", Channel: domain.ChannelSMS, Content: "msg1"},
			{Recipient: "user@example.com", Channel: domain.ChannelEmail, Content: "msg2"},
		},
	}
	s.repo.On("CreateBatch", mock.Anything, mock.AnythingOfType("[]*domain.Notification")).Return(nil)
	s.pub.On("EnqueueBatch", mock.Anything, mock.AnythingOfType("[]*domain.Notification")).Return(nil)

	result, err := s.svc.CreateBatch(context.Background(), input)

	s.NoError(err)
	s.NotEmpty(result.BatchID)
	s.Equal(2, result.Total)
	s.Equal(2, result.Queued)
}

func (s *NotificationServiceSuite) TestCreateBatch_WithTemplate() {
	tmplID := "tmpl-1"
	tmpl := &domain.Template{ID: tmplID, Channel: domain.ChannelSMS, Body: "Hello {{name}}!"}
	input := domain.CreateBatchInput{
		Notifications: []domain.CreateNotificationInput{
			{Recipient: "+905551234567", Channel: domain.ChannelSMS, TemplateID: &tmplID, Variables: map[string]string{"name": "Kenan"}},
		},
	}
	s.tmpl.On("GetByID", mock.Anything, tmplID).Return(tmpl, nil)
	s.repo.On("CreateBatch", mock.Anything, mock.AnythingOfType("[]*domain.Notification")).Return(nil)
	s.pub.On("EnqueueBatch", mock.Anything, mock.AnythingOfType("[]*domain.Notification")).Return(nil)

	result, err := s.svc.CreateBatch(context.Background(), input)

	s.NoError(err)
	s.Equal(1, result.Total)
}

func (s *NotificationServiceSuite) TestCreateBatch_TemplateNotFound() {
	tmplID := "missing"
	input := domain.CreateBatchInput{
		Notifications: []domain.CreateNotificationInput{
			{Recipient: "+905551234567", Channel: domain.ChannelSMS, TemplateID: &tmplID},
		},
	}
	s.tmpl.On("GetByID", mock.Anything, tmplID).Return(nil, domain.ErrTemplateNotFound)

	_, err := s.svc.CreateBatch(context.Background(), input)

	s.ErrorIs(err, domain.ErrTemplateNotFound)
	s.repo.AssertNotCalled(s.T(), "CreateBatch")
}

func (s *NotificationServiceSuite) TestCreateBatch_ContentRequired() {
	input := domain.CreateBatchInput{
		Notifications: []domain.CreateNotificationInput{
			{Recipient: "+905551234567", Channel: domain.ChannelSMS},
		},
	}

	_, err := s.svc.CreateBatch(context.Background(), input)

	s.ErrorIs(err, domain.ErrContentRequired)
	s.repo.AssertNotCalled(s.T(), "CreateBatch")
}

func (s *NotificationServiceSuite) TestCreateBatch_InvalidRecipient() {
	input := domain.CreateBatchInput{
		Notifications: []domain.CreateNotificationInput{
			{Recipient: "bad-phone", Channel: domain.ChannelSMS, Content: "msg"},
		},
	}

	_, err := s.svc.CreateBatch(context.Background(), input)

	s.ErrorIs(err, domain.ErrInvalidRecipient)
	s.repo.AssertNotCalled(s.T(), "CreateBatch")
}

func (s *NotificationServiceSuite) TestCreateBatch_ContentTooLong() {
	input := domain.CreateBatchInput{
		Notifications: []domain.CreateNotificationInput{
			{Recipient: "+905551234567", Channel: domain.ChannelSMS, Content: strings.Repeat("x", 161)},
		},
	}

	_, err := s.svc.CreateBatch(context.Background(), input)

	s.ErrorIs(err, domain.ErrContentTooLong)
	s.repo.AssertNotCalled(s.T(), "CreateBatch")
}

func (s *NotificationServiceSuite) TestCreateBatch_AllScheduled_NothingQueued() {
	future := time.Now().Add(time.Hour)
	input := domain.CreateBatchInput{
		Notifications: []domain.CreateNotificationInput{
			{Recipient: "+905551234567", Channel: domain.ChannelSMS, Content: "msg", ScheduledAt: &future},
		},
	}
	s.repo.On("CreateBatch", mock.Anything, mock.AnythingOfType("[]*domain.Notification")).Return(nil)

	result, err := s.svc.CreateBatch(context.Background(), input)

	s.NoError(err)
	s.Equal(0, result.Queued)
	s.pub.AssertNotCalled(s.T(), "EnqueueBatch")
}

func (s *NotificationServiceSuite) TestCreateBatch_EnqueueBatchError() {
	input := domain.CreateBatchInput{
		Notifications: []domain.CreateNotificationInput{
			{Recipient: "+905551234567", Channel: domain.ChannelSMS, Content: "msg"},
		},
	}
	s.repo.On("CreateBatch", mock.Anything, mock.AnythingOfType("[]*domain.Notification")).Return(nil)
	s.pub.On("EnqueueBatch", mock.Anything, mock.AnythingOfType("[]*domain.Notification")).
		Return(errors.New("queue unavailable"))

	result, err := s.svc.CreateBatch(context.Background(), input)

	s.NoError(err)
	s.Equal(1, result.Total)
}

func (s *NotificationServiceSuite) TestCreateBatch_RepoError() {
	dbErr := errors.New("db unavailable")
	input := domain.CreateBatchInput{
		Notifications: []domain.CreateNotificationInput{
			{Recipient: "+905551234567", Channel: domain.ChannelSMS, Content: "msg"},
		},
	}
	s.repo.On("CreateBatch", mock.Anything, mock.AnythingOfType("[]*domain.Notification")).Return(dbErr)

	_, err := s.svc.CreateBatch(context.Background(), input)

	s.ErrorIs(err, dbErr)
}

// --- GetByID ---

func (s *NotificationServiceSuite) TestGetByID_NotFound() {
	s.repo.On("GetByID", mock.Anything, "missing-id").Return(nil, domain.ErrNotFound)

	_, err := s.svc.GetByID(context.Background(), "missing-id")

	s.ErrorIs(err, domain.ErrNotFound)
}

// --- Cancel ---

func (s *NotificationServiceSuite) TestCancel_Success() {
	s.repo.On("Cancel", mock.Anything, "some-id").Return(nil)

	s.NoError(s.svc.Cancel(context.Background(), "some-id"))
}

func (s *NotificationServiceSuite) TestCancel_CannotCancel() {
	s.repo.On("Cancel", mock.Anything, "some-id").Return(domain.ErrCannotCancel)

	s.ErrorIs(s.svc.Cancel(context.Background(), "some-id"), domain.ErrCannotCancel)
}

// --- List ---

func (s *NotificationServiceSuite) TestList_Success() {
	expected := []*domain.Notification{{ID: "1"}, {ID: "2"}}
	s.repo.On("List", mock.Anything, domain.ListFilter{Page: 1, PageSize: 20}).
		Return(expected, 2, nil)

	results, total, err := s.svc.List(context.Background(), domain.ListFilter{Page: 1, PageSize: 20})

	s.NoError(err)
	s.Equal(2, total)
	s.Len(results, 2)
}
