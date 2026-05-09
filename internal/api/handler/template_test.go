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

type TemplateHandlerSuite struct {
	suite.Suite
	svc    *mocks.TemplateService
	router *gin.Engine
}

func TestTemplateHandlerSuite(t *testing.T) {
	suite.Run(t, new(TemplateHandlerSuite))
}

func (s *TemplateHandlerSuite) SetupTest() {
	gin.SetMode(gin.TestMode)
	s.svc = &mocks.TemplateService{}
	h := handler.NewTemplateHandler(s.svc)
	r := gin.New()
	r.POST("/templates", h.Create)
	r.GET("/templates", h.List)
	r.GET("/templates/:id", h.GetByID)
	r.DELETE("/templates/:id", h.Delete)
	s.router = r
}

func (s *TemplateHandlerSuite) TearDownTest() {
	s.svc.AssertExpectations(s.T())
}

func (s *TemplateHandlerSuite) do(method, path string, body []byte) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(method, path, bytes.NewReader(body))
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	s.router.ServeHTTP(w, req)
	return w
}

func (s *TemplateHandlerSuite) validCreateBody() []byte {
	b, _ := json.Marshal(domain.CreateTemplateInput{
		Name:    "welcome_sms",
		Channel: domain.ChannelSMS,
		Body:    "Hello {{name}}!",
	})
	return b
}

// --- Create ---

func (s *TemplateHandlerSuite) TestCreate_InvalidBody() {
	w := s.do(http.MethodPost, "/templates", []byte(`{}`))
	s.Equal(http.StatusBadRequest, w.Code)
}

func (s *TemplateHandlerSuite) TestCreate_Success() {
	t := &domain.Template{ID: "tmpl-1", Name: "welcome_sms", Channel: domain.ChannelSMS}
	s.svc.On("Create", mock.Anything, mock.AnythingOfType("domain.CreateTemplateInput")).Return(t, nil)

	w := s.do(http.MethodPost, "/templates", s.validCreateBody())
	s.Equal(http.StatusCreated, w.Code)

	var resp domain.Template
	json.Unmarshal(w.Body.Bytes(), &resp)
	s.Equal("tmpl-1", resp.ID)
}

func (s *TemplateHandlerSuite) TestCreate_InternalError() {
	s.svc.On("Create", mock.Anything, mock.AnythingOfType("domain.CreateTemplateInput")).
		Return(nil, errors.New("db error"))

	w := s.do(http.MethodPost, "/templates", s.validCreateBody())
	s.Equal(http.StatusInternalServerError, w.Code)
}

// --- GetByID ---

func (s *TemplateHandlerSuite) TestGetByID_Success() {
	t := &domain.Template{ID: "tmpl-1", Name: "welcome_sms"}
	s.svc.On("GetByID", mock.Anything, "tmpl-1").Return(t, nil)

	w := s.do(http.MethodGet, "/templates/tmpl-1", nil)
	s.Equal(http.StatusOK, w.Code)

	var resp domain.Template
	json.Unmarshal(w.Body.Bytes(), &resp)
	s.Equal("tmpl-1", resp.ID)
}

func (s *TemplateHandlerSuite) TestGetByID_NotFound() {
	s.svc.On("GetByID", mock.Anything, "missing").Return(nil, domain.ErrTemplateNotFound)

	w := s.do(http.MethodGet, "/templates/missing", nil)
	s.Equal(http.StatusNotFound, w.Code)
}

func (s *TemplateHandlerSuite) TestGetByID_InternalError() {
	s.svc.On("GetByID", mock.Anything, "err-id").Return(nil, errors.New("db error"))

	w := s.do(http.MethodGet, "/templates/err-id", nil)
	s.Equal(http.StatusInternalServerError, w.Code)
}

// --- List ---

func (s *TemplateHandlerSuite) TestList_Success() {
	templates := []*domain.Template{{ID: "1"}, {ID: "2"}}
	s.svc.On("List", mock.Anything).Return(templates, nil)

	w := s.do(http.MethodGet, "/templates", nil)
	s.Equal(http.StatusOK, w.Code)

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	s.Equal(float64(2), resp["total"])
}

func (s *TemplateHandlerSuite) TestList_InternalError() {
	s.svc.On("List", mock.Anything).Return(nil, errors.New("db error"))

	w := s.do(http.MethodGet, "/templates", nil)
	s.Equal(http.StatusInternalServerError, w.Code)
}

// --- Delete ---

func (s *TemplateHandlerSuite) TestDelete_Success() {
	s.svc.On("Delete", mock.Anything, "tmpl-1").Return(nil)

	w := s.do(http.MethodDelete, "/templates/tmpl-1", nil)
	s.Equal(http.StatusOK, w.Code)
}

func (s *TemplateHandlerSuite) TestDelete_NotFound() {
	s.svc.On("Delete", mock.Anything, "missing").Return(domain.ErrTemplateNotFound)

	w := s.do(http.MethodDelete, "/templates/missing", nil)
	s.Equal(http.StatusNotFound, w.Code)
}

func (s *TemplateHandlerSuite) TestDelete_InternalError() {
	s.svc.On("Delete", mock.Anything, "err-id").Return(errors.New("db error"))

	w := s.do(http.MethodDelete, "/templates/err-id", nil)
	s.Equal(http.StatusInternalServerError, w.Code)
}
