package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	"github.com/kenanabbak/notification-management-api/internal/domain"
	"github.com/kenanabbak/notification-management-api/internal/domain/mocks"
	"github.com/kenanabbak/notification-management-api/internal/service"
)

type TemplateServiceSuite struct {
	suite.Suite
	repo *mocks.TemplateRepository
	svc  domain.TemplateService
}

func TestTemplateServiceSuite(t *testing.T) {
	suite.Run(t, new(TemplateServiceSuite))
}

func (s *TemplateServiceSuite) SetupTest() {
	s.repo = &mocks.TemplateRepository{}
	s.svc = service.NewTemplateService(s.repo, zap.NewNop())
}

func (s *TemplateServiceSuite) TearDownTest() {
	s.repo.AssertExpectations(s.T())
}

func (s *TemplateServiceSuite) TestCreate_Success() {
	s.repo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Template")).Return(nil)

	t, err := s.svc.Create(context.Background(), domain.CreateTemplateInput{
		Name:    "welcome_sms",
		Channel: domain.ChannelSMS,
		Body:    "Hello {{name}}!",
	})

	s.NoError(err)
	s.NotEmpty(t.ID)
	s.Equal("welcome_sms", t.Name)
	s.Equal(domain.ChannelSMS, t.Channel)
}

func (s *TemplateServiceSuite) TestCreate_RepoError() {
	dbErr := errors.New("db unavailable")
	s.repo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Template")).Return(dbErr)

	_, err := s.svc.Create(context.Background(), domain.CreateTemplateInput{
		Name:    "welcome_sms",
		Channel: domain.ChannelSMS,
		Body:    "Hello",
	})

	s.ErrorIs(err, dbErr)
}

func (s *TemplateServiceSuite) TestGetByID_Success() {
	expected := &domain.Template{ID: "tmpl-1", Name: "welcome_sms"}
	s.repo.On("GetByID", mock.Anything, "tmpl-1").Return(expected, nil)

	t, err := s.svc.GetByID(context.Background(), "tmpl-1")

	s.NoError(err)
	s.Equal("tmpl-1", t.ID)
}

func (s *TemplateServiceSuite) TestGetByID_NotFound() {
	s.repo.On("GetByID", mock.Anything, "missing").Return(nil, domain.ErrTemplateNotFound)

	_, err := s.svc.GetByID(context.Background(), "missing")

	s.ErrorIs(err, domain.ErrTemplateNotFound)
}

func (s *TemplateServiceSuite) TestList_Success() {
	expected := []*domain.Template{{ID: "1"}, {ID: "2"}}
	s.repo.On("List", mock.Anything).Return(expected, nil)

	templates, err := s.svc.List(context.Background())

	s.NoError(err)
	s.Len(templates, 2)
}

func (s *TemplateServiceSuite) TestList_RepoError() {
	dbErr := errors.New("db unavailable")
	s.repo.On("List", mock.Anything).Return(nil, dbErr)

	_, err := s.svc.List(context.Background())

	s.ErrorIs(err, dbErr)
}

func (s *TemplateServiceSuite) TestDelete_Success() {
	s.repo.On("Delete", mock.Anything, "tmpl-1").Return(nil)

	s.NoError(s.svc.Delete(context.Background(), "tmpl-1"))
}

func (s *TemplateServiceSuite) TestDelete_NotFound() {
	s.repo.On("Delete", mock.Anything, "missing").Return(domain.ErrTemplateNotFound)

	s.ErrorIs(s.svc.Delete(context.Background(), "missing"), domain.ErrTemplateNotFound)
}

// --- RenderTemplate ---

type RenderSuite struct{ suite.Suite }

func TestRenderSuite(t *testing.T) { suite.Run(t, new(RenderSuite)) }

func (s *RenderSuite) TestRender_WithVariables() {
	result := service.RenderTemplate("Hello {{name}}, code: {{code}}", map[string]string{
		"name": "Kenan",
		"code": "1234",
	})
	s.Equal("Hello Kenan, code: 1234", result)
}

func (s *RenderSuite) TestRender_NoVariables() {
	result := service.RenderTemplate("Hello world", nil)
	s.Equal("Hello world", result)
}

func (s *RenderSuite) TestRender_UnknownPlaceholder() {
	result := service.RenderTemplate("Hello {{name}} {{unknown}}", map[string]string{"name": "Kenan"})
	s.Equal("Hello Kenan {{unknown}}", result)
}

func (s *RenderSuite) TestRender_EmptyVariables() {
	result := service.RenderTemplate("Hello {{name}}", map[string]string{})
	s.Equal("Hello {{name}}", result)
}
