package ws_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/suite"
	"go.uber.org/fx"

	"github.com/kenanabbak/notification-management-api/internal/domain"
	"github.com/kenanabbak/notification-management-api/internal/ws"
)

type HubSuite struct {
	suite.Suite
	hub    *ws.Hub
	ctx    context.Context
	cancel context.CancelFunc
}

func TestHubSuite(t *testing.T) {
	suite.Run(t, new(HubSuite))
}

func (s *HubSuite) SetupTest() {
	s.hub = ws.NewHub()
	s.ctx, s.cancel = context.WithCancel(context.Background())
	go s.hub.Run(s.ctx)
}

func (s *HubSuite) TearDownTest() {
	s.cancel()
}

func (s *HubSuite) makeServer() (*httptest.Server, *websocket.Conn) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.hub.ServeWS(w, r)
	}))
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	s.Require().NoError(err)
	time.Sleep(20 * time.Millisecond)
	return srv, conn
}

func (s *HubSuite) TestClientReceivesDeliveredEvent() {
	srv, conn := s.makeServer()
	defer srv.Close()
	defer conn.Close()

	s.hub.Notify("notif-123", domain.StatusDelivered)

	conn.SetReadDeadline(time.Now().Add(time.Second))
	_, msg, err := conn.ReadMessage()
	s.Require().NoError(err)

	var event ws.StatusEvent
	s.Require().NoError(json.Unmarshal(msg, &event))
	s.Equal("notif-123", event.NotificationID)
	s.Equal(string(domain.StatusDelivered), event.Status)
	s.False(event.Timestamp.IsZero())
}

func (s *HubSuite) TestClientReceivesFailedEvent() {
	srv, conn := s.makeServer()
	defer srv.Close()
	defer conn.Close()

	s.hub.Notify("notif-456", domain.StatusFailed)

	conn.SetReadDeadline(time.Now().Add(time.Second))
	_, msg, err := conn.ReadMessage()
	s.Require().NoError(err)

	var event ws.StatusEvent
	s.Require().NoError(json.Unmarshal(msg, &event))
	s.Equal("notif-456", event.NotificationID)
	s.Equal(string(domain.StatusFailed), event.Status)
}

func (s *HubSuite) TestMultipleClients_BothReceive() {
	srv1, conn1 := s.makeServer()
	defer srv1.Close()
	defer conn1.Close()

	srv2, conn2 := s.makeServer()
	defer srv2.Close()
	defer conn2.Close()

	s.hub.Notify("notif-789", domain.StatusDelivered)

	for _, conn := range []*websocket.Conn{conn1, conn2} {
		conn.SetReadDeadline(time.Now().Add(time.Second))
		_, msg, err := conn.ReadMessage()
		s.Require().NoError(err)

		var event ws.StatusEvent
		s.Require().NoError(json.Unmarshal(msg, &event))
		s.Equal("notif-789", event.NotificationID)
	}
}

func (s *HubSuite) TestNotify_NoClients_DoesNotPanic() {
	s.NotPanics(func() {
		s.hub.Notify("notif-000", domain.StatusDelivered)
	})
}

func (s *HubSuite) TestClientDisconnect_HubContinues() {
	srv, conn := s.makeServer()
	defer srv.Close()

	conn.Close()
	time.Sleep(20 * time.Millisecond)

	s.NotPanics(func() {
		s.hub.Notify("notif-111", domain.StatusDelivered)
	})
}

func (s *HubSuite) TestRegisterHubLifecycle_StartsAndStops() {
	app := fx.New(
		fx.Provide(ws.NewHub),
		fx.Provide(func(h *ws.Hub) domain.StatusNotifier { return h }),
		fx.Invoke(ws.RegisterHubLifecycle),
		fx.NopLogger,
	)
	s.Require().NoError(app.Start(context.Background()))
	s.Require().NoError(app.Stop(context.Background()))
}

func (s *HubSuite) TestSlowClient_DroppedFromHub() {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.hub.ServeWS(w, r)
	}))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	s.Require().NoError(err)
	defer conn.Close()
	time.Sleep(20 * time.Millisecond)

	// flood more than send buffer (256) to trigger the default drop path
	for i := 0; i < 300; i++ {
		s.hub.Notify("flood-id", domain.StatusDelivered)
	}
	time.Sleep(50 * time.Millisecond)

	// hub should still work after dropping the slow client
	s.NotPanics(func() {
		s.hub.Notify("after-flood", domain.StatusDelivered)
	})
}
