package queue

import "go.uber.org/fx"

var Module = fx.Module("queue",
	fx.Provide(NewRedisClient),
	fx.Provide(NewPublisher),
	fx.Provide(NewWorker),
	fx.Provide(NewScheduler),
	fx.Invoke(RegisterWorkerLifecycle),
	fx.Invoke(RegisterSchedulerLifecycle),
)
