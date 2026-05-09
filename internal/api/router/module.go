package router

import (
	"github.com/kenanabbak/notification-management-api/internal/api/handler"
	"go.uber.org/fx"
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
