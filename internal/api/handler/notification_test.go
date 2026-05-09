package handler_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"github.com/kenanabbak/notification-management-api/internal/api/handler"
	"github.com/kenanabbak/notification-management-api/internal/domain"
	"github.com/kenanabbak/notification-management-api/internal/domain/mocks"
)

type NotificationHandlerSuite struct {
	suite.Suite
	svc    *mocks.NotificationService
	router *gin.Engine
}

func TestNotificationHandlerSuite(t *testing.T) {
	suite.Run(t, new(NotificationHandlerSuite))
}

func (s *NotificationHandlerSuite) SetupTest() {
	gin.SetMode(gin.TestMode)
	s.svc = &mocks.NotificationService{}
	h := handler.NewNotificationHandler(s.svc)
	r := gin.New()
	r.POST("/notifications", h.Create)
	r.POST("/notifications/batch", h.CreateBatch)
	r.GET("/notifications/:id", h.GetByID)
	r.DELETE("/notifications/:id", h.Cancel)
	r.GET("/notifications", h.List)
	s.router = r
}

func (s *NotificationHandlerSuite) TearDownTest() {
	s.svc.AssertExpectations(s.T())
}

func (s *NotificationHandlerSuite) do(method, path string, body []byte) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(method, path, bytes.NewReader(body))
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	s.router.ServeHTTP(w, req)
	return w
}

func (s *NotificationHandlerSuite) validCreateBody() []byte {
	b, _ := json.Marshal(domain.CreateNotificationInput{
		Recipient: "+905551234567",
		Channel:   domain.ChannelSMS,
		Content:   "Hello",
	})
	return b
}

// --- Create ---

func (s *NotificationHandlerSuite) TestCreate_InvalidBody() {
	w := s.do(http.MethodPost, "/notifications", []byte(`{}`))
	s.Equal(http.StatusBadRequest, w.Code)
}

func (s *NotificationHandlerSuite) TestCreate_Success() {
	n := &domain.Notification{ID: "new-id", Channel: domain.ChannelSMS, Status: domain.StatusQueued}
	s.svc.On("Create", mock.Anything, mock.AnythingOfType("domain.CreateNotificationInput")).Return(n, nil)

	w := s.do(http.MethodPost, "/notifications", s.validCreateBody())
	s.Equal(http.StatusCreated, w.Code)

	var resp domain.Notification
	json.Unmarshal(w.Body.Bytes(), &resp)
	s.Equal("new-id", resp.ID)
}

func (s *NotificationHandlerSuite) TestCreate_ContentTooLong() {
	s.svc.On("Create", mock.Anything, mock.AnythingOfType("domain.CreateNotificationInput")).
		Return(nil, domain.ErrContentTooLong)

	w := s.do(http.MethodPost, "/notifications", s.validCreateBody())
	s.Equal(http.StatusUnprocessableEntity, w.Code)
}

func (s *NotificationHandlerSuite) TestCreate_InvalidRecipient() {
	s.svc.On("Create", mock.Anything, mock.AnythingOfType("domain.CreateNotificationInput")).
		Return(nil, domain.ErrInvalidRecipient)

	w := s.do(http.MethodPost, "/notifications", s.validCreateBody())
	s.Equal(http.StatusUnprocessableEntity, w.Code)
}

func (s *NotificationHandlerSuite) TestCreate_Duplicate() {
	s.svc.On("Create", mock.Anything, mock.AnythingOfType("domain.CreateNotificationInput")).
		Return(nil, domain.ErrDuplicate)

	w := s.do(http.MethodPost, "/notifications", s.validCreateBody())
	s.Equal(http.StatusConflict, w.Code)
}

func (s *NotificationHandlerSuite) TestCreate_TemplateNotFound() {
	s.svc.On("Create", mock.Anything, mock.AnythingOfType("domain.CreateNotificationInput")).
		Return(nil, domain.ErrTemplateNotFound)

	w := s.do(http.MethodPost, "/notifications", s.validCreateBody())
	s.Equal(http.StatusNotFound, w.Code)
}

func (s *NotificationHandlerSuite) TestCreate_ContentRequired() {
	s.svc.On("Create", mock.Anything, mock.AnythingOfType("domain.CreateNotificationInput")).
		Return(nil, domain.ErrContentRequired)

	w := s.do(http.MethodPost, "/notifications", s.validCreateBody())
	s.Equal(http.StatusBadRequest, w.Code)
}

func (s *NotificationHandlerSuite) TestCreate_InternalError() {
	s.svc.On("Create", mock.Anything, mock.AnythingOfType("domain.CreateNotificationInput")).
		Return(nil, errors.New("unexpected db error"))

	w := s.do(http.MethodPost, "/notifications", s.validCreateBody())
	s.Equal(http.StatusInternalServerError, w.Code)
}

// --- CreateBatch ---

func (s *NotificationHandlerSuite) TestCreateBatch_TooMany() {
	items := make([]domain.CreateNotificationInput, 1001)
	for i := range items {
		items[i] = domain.CreateNotificationInput{Recipient: "+905551234567", Channel: domain.ChannelSMS, Content: "x"}
	}
	body, _ := json.Marshal(domain.CreateBatchInput{Notifications: items})

	w := s.do(http.MethodPost, "/notifications/batch", body)
	s.Equal(http.StatusBadRequest, w.Code)
}

func (s *NotificationHandlerSuite) TestCreateBatch_Success() {
	result := &domain.BatchResult{BatchID: "batch-1", Total: 2, Queued: 2}
	s.svc.On("CreateBatch", mock.Anything, mock.AnythingOfType("domain.CreateBatchInput")).
		Return(result, nil)

	body, _ := json.Marshal(domain.CreateBatchInput{
		Notifications: []domain.CreateNotificationInput{
			{Recipient: "+905551234567", Channel: domain.ChannelSMS, Content: "x"},
		},
	})

	w := s.do(http.MethodPost, "/notifications/batch", body)
	s.Equal(http.StatusCreated, w.Code)

	var resp domain.BatchResult
	json.Unmarshal(w.Body.Bytes(), &resp)
	s.Equal("batch-1", resp.BatchID)
}

func (s *NotificationHandlerSuite) TestCreateBatch_ValidationError() {
	s.svc.On("CreateBatch", mock.Anything, mock.AnythingOfType("domain.CreateBatchInput")).
		Return(nil, domain.ErrContentTooLong)

	body, _ := json.Marshal(domain.CreateBatchInput{
		Notifications: []domain.CreateNotificationInput{
			{Recipient: "+905551234567", Channel: domain.ChannelSMS, Content: "x"},
		},
	})

	w := s.do(http.MethodPost, "/notifications/batch", body)
	s.Equal(http.StatusUnprocessableEntity, w.Code)
}

func (s *NotificationHandlerSuite) TestCreateBatch_InternalError() {
	s.svc.On("CreateBatch", mock.Anything, mock.AnythingOfType("domain.CreateBatchInput")).
		Return(nil, errors.New("db error"))

	body, _ := json.Marshal(domain.CreateBatchInput{
		Notifications: []domain.CreateNotificationInput{
			{Recipient: "+905551234567", Channel: domain.ChannelSMS, Content: "x"},
		},
	})

	w := s.do(http.MethodPost, "/notifications/batch", body)
	s.Equal(http.StatusInternalServerError, w.Code)
}

// --- GetByID ---

func (s *NotificationHandlerSuite) TestGetByID_Success() {
	n := &domain.Notification{ID: "abc-123", Channel: domain.ChannelSMS, Status: domain.StatusDelivered}
	s.svc.On("GetByID", mock.Anything, "abc-123").Return(n, nil)

	w := s.do(http.MethodGet, "/notifications/abc-123", nil)
	s.Equal(http.StatusOK, w.Code)

	var resp domain.Notification
	json.Unmarshal(w.Body.Bytes(), &resp)
	s.Equal("abc-123", resp.ID)
}

func (s *NotificationHandlerSuite) TestGetByID_NotFound() {
	s.svc.On("GetByID", mock.Anything, "unknown").Return(nil, domain.ErrNotFound)

	w := s.do(http.MethodGet, "/notifications/unknown", nil)
	s.Equal(http.StatusNotFound, w.Code)
}

func (s *NotificationHandlerSuite) TestGetByID_InternalError() {
	s.svc.On("GetByID", mock.Anything, "err-id").Return(nil, errors.New("db error"))

	w := s.do(http.MethodGet, "/notifications/err-id", nil)
	s.Equal(http.StatusInternalServerError, w.Code)
}

// --- Cancel ---

func (s *NotificationHandlerSuite) TestCancel_Success() {
	s.svc.On("Cancel", mock.Anything, "some-id").Return(nil)

	w := s.do(http.MethodDelete, "/notifications/some-id", nil)
	s.Equal(http.StatusOK, w.Code)
}

func (s *NotificationHandlerSuite) TestCancel_NotFound() {
	s.svc.On("Cancel", mock.Anything, "missing-id").Return(domain.ErrNotFound)

	w := s.do(http.MethodDelete, "/notifications/missing-id", nil)
	s.Equal(http.StatusNotFound, w.Code)
}

func (s *NotificationHandlerSuite) TestCancel_Conflict() {
	s.svc.On("Cancel", mock.Anything, "delivered-id").Return(domain.ErrCannotCancel)

	w := s.do(http.MethodDelete, "/notifications/delivered-id", nil)
	s.Equal(http.StatusConflict, w.Code)
}

func (s *NotificationHandlerSuite) TestCancel_InternalError() {
	s.svc.On("Cancel", mock.Anything, "err-id").Return(errors.New("db error"))

	w := s.do(http.MethodDelete, "/notifications/err-id", nil)
	s.Equal(http.StatusInternalServerError, w.Code)
}

// --- List ---

func (s *NotificationHandlerSuite) TestList_Success() {
	notifications := []*domain.Notification{
		{ID: "1", Channel: domain.ChannelSMS},
		{ID: "2", Channel: domain.ChannelEmail},
	}
	s.svc.On("List", mock.Anything, domain.ListFilter{Page: 1, PageSize: 20}).
		Return(notifications, 2, nil)

	w := s.do(http.MethodGet, "/notifications", nil)
	s.Equal(http.StatusOK, w.Code)

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	s.Equal(float64(2), resp["total"])
}

func (s *NotificationHandlerSuite) TestList_WithFilters() {
	status := domain.StatusQueued
	s.svc.On("List", mock.Anything, domain.ListFilter{
		Status: &status, Page: 2, PageSize: 5,
	}).Return([]*domain.Notification{}, 0, nil)

	w := s.do(http.MethodGet, "/notifications?status=queued&page=2&page_size=5", nil)
	s.Equal(http.StatusOK, w.Code)
}

func (s *NotificationHandlerSuite) TestList_PageBoundaries() {
	s.svc.On("List", mock.Anything, domain.ListFilter{Page: 1, PageSize: 20}).
		Return([]*domain.Notification{}, 0, nil)

	w := s.do(http.MethodGet, "/notifications?page=0&page_size=200", nil)
	s.Equal(http.StatusOK, w.Code)
}

func (s *NotificationHandlerSuite) TestList_InvalidQueryParam() {
	w := s.do(http.MethodGet, "/notifications?page=not-a-number", nil)
	s.Equal(http.StatusBadRequest, w.Code)
}

func (s *NotificationHandlerSuite) TestList_InternalError() {
	s.svc.On("List", mock.Anything, domain.ListFilter{Page: 1, PageSize: 20}).
		Return(nil, 0, errors.New("db error"))

	w := s.do(http.MethodGet, "/notifications", nil)
	s.Equal(http.StatusInternalServerError, w.Code)
}
