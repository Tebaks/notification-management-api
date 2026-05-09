package main

import (
	"go.uber.org/fx"

	"github.com/kenanabbak/notification-management-api/internal/api/router"
	"github.com/kenanabbak/notification-management-api/internal/config"
	"github.com/kenanabbak/notification-management-api/internal/metrics"
	"github.com/kenanabbak/notification-management-api/internal/queue"
	"github.com/kenanabbak/notification-management-api/internal/repository/postgres"
	"github.com/kenanabbak/notification-management-api/internal/service"
	"github.com/kenanabbak/notification-management-api/internal/telemetry"
	"github.com/kenanabbak/notification-management-api/internal/ws"
)

//go:generate swag init -g ../../cmd/api/main.go -o ../../docs

// @title           Notification Management API
// @version         1.0
// @description     Scalable multi-channel notification system
// @host            localhost:8080
// @BasePath        /
func main() {
	fx.New(
		config.Module,
		telemetry.Module,
		metrics.Module,
		postgres.Module,
		ws.Module,
		queue.Module,
		service.Module,
		router.Module,
	).Run()
}
