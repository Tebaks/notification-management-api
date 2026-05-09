package handler_test

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/suite"

	"github.com/kenanabbak/notification-management-api/internal/api/handler"
	"github.com/kenanabbak/notification-management-api/internal/domain"
	"github.com/kenanabbak/notification-management-api/internal/ws"
)

type WSHandlerSuite struct {
	suite.Suite
	hub    *ws.Hub
	ctx    context.Context
	cancel context.CancelFunc
}

func TestWSHandlerSuite(t *testing.T) {
	suite.Run(t, new(WSHandlerSuite))
}

func (s *WSHandlerSuite) SetupTest() {
	s.hub = ws.NewHub()
	s.ctx, s.cancel = context.WithCancel(context.Background())
	go s.hub.Run(s.ctx)
}

func (s *WSHandlerSuite) TearDownTest() {
	s.cancel()
}

func (s *WSHandlerSuite) TestServeWS_ClientReceivesEvent() {
	gin.SetMode(gin.TestMode)
	h := handler.NewWSHandler(s.hub)
	r := gin.New()
	r.GET("/ws", h.ServeWS)

	srv := httptest.NewServer(r)
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	s.Require().NoError(err)
	defer conn.Close()

	time.Sleep(20 * time.Millisecond)
	s.hub.Notify("ws-handler-id", domain.StatusDelivered)

	conn.SetReadDeadline(time.Now().Add(time.Second))
	_, msg, err := conn.ReadMessage()
	s.Require().NoError(err)

	var event ws.StatusEvent
	s.Require().NoError(json.Unmarshal(msg, &event))
	s.Equal("ws-handler-id", event.NotificationID)
	s.Equal(string(domain.StatusDelivered), event.Status)
}
