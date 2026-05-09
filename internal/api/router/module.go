package router

import (
	"go.uber.org/fx"

	"github.com/kenanabbak/notification-management-api/internal/api/handler"
)

var Module = fx.Module("router",
	fx.Provide(handler.NewNotificationHandler),
	fx.Provide(handler.NewMetricsHandler),
	fx.Provide(handler.NewHealthHandler),
	fx.Provide(handler.NewTemplateHandler),
	fx.Provide(handler.NewWSHandler),
	fx.Provide(New),
	fx.Invoke(RegisterLifecycle),
)
