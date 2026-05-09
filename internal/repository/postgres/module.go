package postgres

import "go.uber.org/fx"

var Module = fx.Module("postgres",
	fx.Provide(NewDB),
	fx.Provide(NewNotificationRepository),
	fx.Provide(NewTemplateRepository),
	fx.Invoke(InvokeMigrations),
)
