package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/suite"

	"github.com/kenanabbak/notification-management-api/internal/api/handler"
	"github.com/kenanabbak/notification-management-api/internal/domain"
	appmetrics "github.com/kenanabbak/notification-management-api/internal/metrics"
)

type mockQueueReader struct{ high, normal, low int64 }

func (m *mockQueueReader) QueueDepths(_ context.Context) (int64, int64, int64) {
	return m.high, m.normal, m.low
}

type MetricsHandlerSuite struct {
	suite.Suite
	collector *appmetrics.Collector
}

func TestMetricsHandlerSuite(t *testing.T) {
	suite.Run(t, new(MetricsHandlerSuite))
}

func (s *MetricsHandlerSuite) SetupTest() {
	s.collector = appmetrics.New()
}

func (s *MetricsHandlerSuite) makeRouter(reader handler.QueueDepthReader) *gin.Engine {
	gin.SetMode(gin.TestMode)
	h := handler.NewMetricsHandlerForTest(reader, s.collector)
	r := gin.New()
	r.GET("/metrics", h.Metrics)
	return r
}

func (s *MetricsHandlerSuite) TestMetrics_Success() {
	reader := &mockQueueReader{high: 3, normal: 10, low: 1}
	r := s.makeRouter(reader)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/metrics", nil)
	r.ServeHTTP(w, req)

	s.Equal(http.StatusOK, w.Code)

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	depth := resp["queue_depth"].(map[string]any)
	s.Equal(float64(3), depth["high"])
	s.Equal(float64(10), depth["normal"])
	s.Equal(float64(1), depth["low"])
	s.Equal(float64(14), depth["total"])
}

func (s *MetricsHandlerSuite) TestMetrics_WithDeliveryStats() {
	s.collector.RecordDelivered(domain.ChannelSMS, 100*time.Millisecond)
	s.collector.RecordFailed(domain.ChannelEmail)

	r := s.makeRouter(&mockQueueReader{})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/metrics", nil)
	r.ServeHTTP(w, req)

	s.Equal(http.StatusOK, w.Code)

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	delivery := resp["delivery"].(map[string]any)
	s.Equal(float64(1), delivery["delivered_total"])
	s.Equal(float64(1), delivery["failed_total"])
}

func (s *MetricsHandlerSuite) TestNewMetricsHandler_ProductionConstructor() {
	rdb := redis.NewClient(&redis.Options{
		Addr:        "127.0.0.1:1",
		DialTimeout: 50 * time.Millisecond,
	})
	defer rdb.Close()

	gin.SetMode(gin.TestMode)
	h := handler.NewMetricsHandler(rdb, appmetrics.New())
	r := gin.New()
	r.GET("/metrics", h.Metrics)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/metrics", nil)
	r.ServeHTTP(w, req)

	s.Equal(http.StatusOK, w.Code)
}
