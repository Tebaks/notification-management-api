package handler_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/suite"

	"github.com/kenanabbak/notification-management-api/internal/api/handler"
)

type HealthHandlerSuite struct {
	suite.Suite
}

func TestHealthHandlerSuite(t *testing.T) {
	suite.Run(t, new(HealthHandlerSuite))
}

func (s *HealthHandlerSuite) serve(pingDB, pingRedis func(context.Context) error) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	h := handler.NewHealthHandlerForTest(pingDB, pingRedis)
	r := gin.New()
	r.GET("/health", h.Health)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/health", nil)
	r.ServeHTTP(w, req)
	return w
}

func (s *HealthHandlerSuite) TestHealth_AllHealthy() {
	w := s.serve(
		func(_ context.Context) error { return nil },
		func(_ context.Context) error { return nil },
	)
	s.Equal(http.StatusOK, w.Code)
}

func (s *HealthHandlerSuite) TestHealth_DBDown() {
	w := s.serve(
		func(_ context.Context) error { return errors.New("connection refused") },
		func(_ context.Context) error { return nil },
	)
	s.Equal(http.StatusServiceUnavailable, w.Code)
}

func (s *HealthHandlerSuite) TestHealth_RedisDown() {
	w := s.serve(
		func(_ context.Context) error { return nil },
		func(_ context.Context) error { return errors.New("connection refused") },
	)
	s.Equal(http.StatusServiceUnavailable, w.Code)
}

func (s *HealthHandlerSuite) TestHealth_BothDown() {
	w := s.serve(
		func(_ context.Context) error { return errors.New("db down") },
		func(_ context.Context) error { return errors.New("redis down") },
	)
	s.Equal(http.StatusServiceUnavailable, w.Code)
}

func (s *HealthHandlerSuite) TestNewHealthHandler_ProductionConstructor() {
	db, err := sqlx.Open("postgres", "postgres://u:p@127.0.0.1:1/db?sslmode=disable&connect_timeout=1")
	s.Require().NoError(err)
	defer db.Close()

	rdb := redis.NewClient(&redis.Options{
		Addr:        "127.0.0.1:1",
		DialTimeout: 50 * time.Millisecond,
	})
	defer rdb.Close()

	gin.SetMode(gin.TestMode)
	h := handler.NewHealthHandler(db, rdb)
	r := gin.New()
	r.GET("/health", h.Health)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/health", nil)
	r.ServeHTTP(w, req)

	s.Equal(http.StatusServiceUnavailable, w.Code)
}
