package queue

import (
	"context"
	"encoding/json"
	"strconv"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"
	"go.uber.org/zap"

	"github.com/kenanabbak/notification-management-api/internal/config"
	"github.com/kenanabbak/notification-management-api/internal/domain"
)

const schedulerInterval = 10 * time.Second
const archiveInterval = time.Hour
const schedulerBatchSize = 500

type Scheduler struct {
	rdb    *redis.Client
	repo   domain.NotificationRepository
	cfg    *config.Config
	log    *zap.Logger
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func NewScheduler(rdb *redis.Client, repo domain.NotificationRepository, cfg *config.Config, log *zap.Logger) *Scheduler {
	return &Scheduler{rdb: rdb, repo: repo, cfg: cfg, log: log}
}

func RegisterSchedulerLifecycle(lc fx.Lifecycle, s *Scheduler) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			schedCtx, cancel := context.WithCancel(context.Background())
			s.cancel = cancel
			s.wg.Add(1)
			go s.run(schedCtx)
			return nil
		},
		OnStop: func(ctx context.Context) error {
			s.log.Info("stopping scheduler")
			s.cancel()
			s.wg.Wait()
			s.log.Info("scheduler stopped")
			return nil
		},
	})
}

func (s *Scheduler) run(ctx context.Context) {
	defer s.wg.Done()
	ticker := time.NewTicker(schedulerInterval)
	archiveTicker := time.NewTicker(archiveInterval)
	defer ticker.Stop()
	defer archiveTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.processScheduled(ctx)
			s.processRetries(ctx)
		case <-archiveTicker.C:
			s.processArchive(ctx)
		}
	}
}

func (s *Scheduler) processScheduled(ctx context.Context) {
	notifications, err := s.repo.GetDueScheduled(ctx, schedulerBatchSize)
	if err != nil {
		s.log.Error("scheduler: failed to fetch due notifications", zap.Error(err))
		return
	}
	if len(notifications) == 0 {
		return
	}

	pipe := s.rdb.Pipeline()
	for _, n := range notifications {
		data, err := json.Marshal(queueMessage{Notification: n})
		if err != nil {
			s.log.Error("scheduler: failed to marshal notification", zap.String("id", n.ID), zap.Error(err))
			continue
		}
		pipe.LPush(ctx, queueKey(n.Priority), data)
	}
	if _, err := pipe.Exec(ctx); err != nil {
		s.log.Error("scheduler: failed to enqueue notifications", zap.Error(err))
		return
	}

	ids := make([]string, len(notifications))
	for i, n := range notifications {
		ids[i] = n.ID
	}
	if err := s.repo.BulkUpdateStatus(ctx, ids, domain.StatusQueued); err != nil {
		s.log.Error("scheduler: failed to bulk update status", zap.Error(err))
		return
	}
	s.log.Info("scheduler: enqueued scheduled notifications", zap.Int("count", len(notifications)))
}

func (s *Scheduler) processRetries(ctx context.Context) {
	now := strconv.FormatInt(time.Now().Unix(), 10)

	results, err := s.rdb.ZRangeArgs(ctx, redis.ZRangeArgs{
		Key:     retryQueue,
		Start:   "-inf",
		Stop:    now,
		ByScore: true,
		Count:   schedulerBatchSize,
	}).Result()
	if err != nil || len(results) == 0 {
		return
	}

	pipe := s.rdb.Pipeline()
	for _, data := range results {
		msg, err := unmarshal([]byte(data))
		if err != nil {
			s.log.Error("scheduler: invalid retry message, discarding", zap.Error(err))
			pipe.ZRem(ctx, retryQueue, data)
			continue
		}
		pipe.LPush(ctx, queueKey(msg.Notification.Priority), data)
		pipe.ZRem(ctx, retryQueue, data)
	}

	if _, err := pipe.Exec(ctx); err != nil {
		s.log.Error("scheduler: failed to process retries", zap.Error(err))
		return
	}
	s.log.Info("scheduler: requeued retried notifications", zap.Int("count", len(results)))
}

func (s *Scheduler) processArchive(ctx context.Context) {
	olderThan := time.Now().Add(-s.cfg.Worker.ArchiveAfter)
	archived, err := s.repo.Archive(ctx, olderThan, s.cfg.Worker.ArchiveBatchSize)
	if err != nil {
		s.log.Error("scheduler: archive failed", zap.Error(err))
		return
	}
	if archived > 0 {
		s.log.Info("scheduler: archived old notifications", zap.Int64("count", archived))
	}
}
