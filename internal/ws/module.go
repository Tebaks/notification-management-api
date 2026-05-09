package ws

import (
	"context"

	"github.com/kenanabbak/notification-management-api/internal/domain"
	"go.uber.org/fx"
)

var Module = fx.Module("ws",
	fx.Provide(NewHub),
	fx.Provide(func(h *Hub) domain.StatusNotifier { return h }),
	fx.Invoke(RegisterHubLifecycle),
)

func RegisterHubLifecycle(lc fx.Lifecycle, h *Hub) {
	var cancel context.CancelFunc
	lc.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			ctx, c := context.WithCancel(context.Background())
			cancel = c
			go h.Run(ctx)
			return nil
		},
		OnStop: func(_ context.Context) error {
			cancel()
			return nil
		},
	})
}
