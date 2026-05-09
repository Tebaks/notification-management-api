package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/kenanabbak/notification-management-api/internal/ws"
)

type WSHandler struct {
	hub *ws.Hub
}

func NewWSHandler(hub *ws.Hub) *WSHandler {
	return &WSHandler{hub: hub}
}

func (h *WSHandler) ServeWS(c *gin.Context) {
	h.hub.ServeWS(c.Writer, c.Request)
}
