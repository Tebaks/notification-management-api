package telemetry

import "go.uber.org/fx"

var Module = fx.Module("telemetry",
	fx.Provide(NewTracerConfig),
	fx.Provide(NewTracerProvider),
	fx.Invoke(RegisterTracerLifecycle),
)
