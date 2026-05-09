//go:build integration

package postgres_test

import (
	"context"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/kenanabbak/notification-management-api/internal/domain"
	"github.com/kenanabbak/notification-management-api/internal/repository/postgres"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"go.uber.org/zap"

	"github.com/google/uuid"
	tc "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func setupDB(t *testing.T) *sqlx.DB {
	t.Helper()
	ctx := context.Background()

	ctr, err := tcpostgres.Run(ctx, "postgres:16-alpine",
		tcpostgres.WithDatabase("testdb"),
		tcpostgres.WithUsername("test"),
		tcpostgres.WithPassword("test"),
		tc.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2),
		),
	)
	require.NoError(t, err)
	t.Cleanup(func() { ctr.Terminate(ctx) })

	dsn, err := ctr.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	db, err := sqlx.Open("postgres", dsn)
	require.NoError(t, err)
	require.NoError(t, db.Ping())

	require.NoError(t, postgres.RunMigrations(db, zap.NewNop()))
	return db
}

func newNotification(channel domain.Channel, status domain.Status) *domain.Notification {
	now := time.Now().UTC().Truncate(time.Millisecond)
	return &domain.Notification{
		ID:        uuid.NewString(),
		Recipient: "+905551234567",
		Channel:   channel,
		Content:   "hello",
		Priority:  domain.PriorityNormal,
		Status:    status,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func TestIntegration_CreateAndGetByID(t *testing.T) {
	db := setupDB(t)
	repo := postgres.NewNotificationRepository(db)
	ctx := context.Background()

	n := newNotification(domain.ChannelSMS, domain.StatusQueued)
	require.NoError(t, repo.Create(ctx, n))

	got, err := repo.GetByID(ctx, n.ID)
	require.NoError(t, err)
	assert.Equal(t, n.ID, got.ID)
	assert.Equal(t, domain.StatusQueued, got.Status)
}

func TestIntegration_GetByID_NotFound(t *testing.T) {
	db := setupDB(t)
	repo := postgres.NewNotificationRepository(db)

	_, err := repo.GetByID(context.Background(), uuid.NewString())
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestIntegration_GetByIdempotencyKey(t *testing.T) {
	db := setupDB(t)
	repo := postgres.NewNotificationRepository(db)
	ctx := context.Background()

	key := "idem-key-" + uuid.NewString()
	n := newNotification(domain.ChannelEmail, domain.StatusQueued)
	n.Recipient = "user@example.com"
	n.IdempotencyKey = &key
	require.NoError(t, repo.Create(ctx, n))

	got, err := repo.GetByIdempotencyKey(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, n.ID, got.ID)
}

func TestIntegration_Cancel_Success(t *testing.T) {
	db := setupDB(t)
	repo := postgres.NewNotificationRepository(db)
	ctx := context.Background()

	n := newNotification(domain.ChannelSMS, domain.StatusQueued)
	require.NoError(t, repo.Create(ctx, n))
	require.NoError(t, repo.Cancel(ctx, n.ID))
}

func TestIntegration_Cancel_NotFound(t *testing.T) {
	db := setupDB(t)
	repo := postgres.NewNotificationRepository(db)

	err := repo.Cancel(context.Background(), uuid.NewString())
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestIntegration_Cancel_CannotCancel(t *testing.T) {
	db := setupDB(t)
	repo := postgres.NewNotificationRepository(db)
	ctx := context.Background()

	n := newNotification(domain.ChannelSMS, domain.StatusDelivered)
	require.NoError(t, repo.Create(ctx, n))

	err := repo.Cancel(ctx, n.ID)
	assert.ErrorIs(t, err, domain.ErrCannotCancel)
}

func TestIntegration_List_WindowFunction(t *testing.T) {
	db := setupDB(t)
	repo := postgres.NewNotificationRepository(db)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		require.NoError(t, repo.Create(ctx, newNotification(domain.ChannelSMS, domain.StatusDelivered)))
	}

	ch := domain.ChannelSMS
	results, total, err := repo.List(ctx, domain.ListFilter{
		Channel:  &ch,
		Page:     1,
		PageSize: 3,
	})
	require.NoError(t, err)
	assert.Equal(t, 5, total)
	assert.Len(t, results, 3)
}

func TestIntegration_GetDueScheduled_PriorityOrder(t *testing.T) {
	db := setupDB(t)
	repo := postgres.NewNotificationRepository(db)
	ctx := context.Background()

	past := time.Now().Add(-time.Minute)

	high := newNotification(domain.ChannelSMS, domain.StatusPending)
	high.Priority = domain.PriorityHigh
	high.ScheduledAt = &past

	low := newNotification(domain.ChannelSMS, domain.StatusPending)
	low.Priority = domain.PriorityLow
	low.ScheduledAt = &past

	require.NoError(t, repo.Create(ctx, low))
	require.NoError(t, repo.Create(ctx, high))

	results, err := repo.GetDueScheduled(ctx, 10)
	require.NoError(t, err)
	require.Len(t, results, 2)
	assert.Equal(t, domain.PriorityHigh, results[0].Priority)
	assert.Equal(t, domain.PriorityLow, results[1].Priority)
}

func TestIntegration_Archive(t *testing.T) {
	db := setupDB(t)
	repo := postgres.NewNotificationRepository(db)
	ctx := context.Background()

	old := newNotification(domain.ChannelSMS, domain.StatusDelivered)
	old.CreatedAt = time.Now().Add(-31 * 24 * time.Hour)
	old.UpdatedAt = old.CreatedAt
	require.NoError(t, repo.Create(ctx, old))

	recent := newNotification(domain.ChannelSMS, domain.StatusDelivered)
	require.NoError(t, repo.Create(ctx, recent))

	cutoff := time.Now().Add(-30 * 24 * time.Hour)
	archived, err := repo.Archive(ctx, cutoff, 100)
	require.NoError(t, err)
	assert.Equal(t, int64(1), archived)

	_, err = repo.GetByID(ctx, old.ID)
	assert.ErrorIs(t, err, domain.ErrNotFound)

	_, err = repo.GetByID(ctx, recent.ID)
	assert.NoError(t, err)
}
