package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/kenanabbak/notification-management-api/internal/domain"
)

const columns = `id, batch_id, recipient, channel, content, priority, status, idempotency_key, provider_msg_id, retry_count, scheduled_at, created_at, updated_at`

type notificationRepository struct {
	db *sqlx.DB
}

func NewNotificationRepository(db *sqlx.DB) domain.NotificationRepository {
	return &notificationRepository{db: db}
}

func (r *notificationRepository) Create(ctx context.Context, n *domain.Notification) error {
	query := `
		INSERT INTO notifications (id, batch_id, recipient, channel, content, priority, status, idempotency_key, scheduled_at, created_at, updated_at)
		VALUES (:id, :batch_id, :recipient, :channel, :content, :priority, :status, :idempotency_key, :scheduled_at, :created_at, :updated_at)`
	_, err := r.db.NamedExecContext(ctx, query, n)
	return err
}

func (r *notificationRepository) CreateBatch(ctx context.Context, notifications []*domain.Notification) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.PrepareNamedContext(ctx, `
		INSERT INTO notifications (id, batch_id, recipient, channel, content, priority, status, idempotency_key, scheduled_at, created_at, updated_at)
		VALUES (:id, :batch_id, :recipient, :channel, :content, :priority, :status, :idempotency_key, :scheduled_at, :created_at, :updated_at)`)
	if err != nil {
		return err
	}
	defer func() { _ = stmt.Close() }()

	for _, n := range notifications {
		if _, err = stmt.ExecContext(ctx, n); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *notificationRepository) GetByID(ctx context.Context, id string) (*domain.Notification, error) {
	var n domain.Notification
	err := r.db.GetContext(ctx, &n, `SELECT `+columns+` FROM notifications WHERE id = $1`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &n, nil
}

func (r *notificationRepository) GetByIdempotencyKey(ctx context.Context, key string) (*domain.Notification, error) {
	var n domain.Notification
	err := r.db.GetContext(ctx, &n, `SELECT `+columns+` FROM notifications WHERE idempotency_key = $1`, key)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &n, nil
}

func (r *notificationRepository) UpdateStatus(ctx context.Context, id string, status domain.Status, providerMsgID *string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE notifications SET status = $1, provider_msg_id = $2, updated_at = $3 WHERE id = $4`,
		status, providerMsgID, time.Now(), id)
	return err
}

func (r *notificationRepository) IncrementRetry(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE notifications SET retry_count = retry_count + 1, updated_at = $1 WHERE id = $2`,
		time.Now(), id)
	return err
}

func (r *notificationRepository) Cancel(ctx context.Context, id string) error {
	var exists bool
	if err := r.db.GetContext(ctx, &exists, `SELECT EXISTS(SELECT 1 FROM notifications WHERE id = $1)`, id); err != nil {
		return err
	}
	if !exists {
		return domain.ErrNotFound
	}

	res, err := r.db.ExecContext(ctx,
		`UPDATE notifications SET status = $1, updated_at = $2 WHERE id = $3 AND status IN ('pending', 'queued')`,
		domain.StatusCancelled, time.Now(), id)
	if err != nil {
		return err
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return domain.ErrCannotCancel
	}
	return nil
}

type notificationWithCount struct {
	domain.Notification
	TotalCount int `db:"total_count"`
}

func (r *notificationRepository) List(ctx context.Context, filter domain.ListFilter) ([]*domain.Notification, int, error) {
	where := []string{"1=1"}
	args := []any{}
	idx := 1

	if filter.Status != nil {
		where = append(where, fmt.Sprintf("status = $%d", idx))
		args = append(args, *filter.Status)
		idx++
	}
	if filter.Channel != nil {
		where = append(where, fmt.Sprintf("channel = $%d", idx))
		args = append(args, *filter.Channel)
		idx++
	}
	if filter.BatchID != nil {
		where = append(where, fmt.Sprintf("batch_id = $%d", idx))
		args = append(args, *filter.BatchID)
		idx++
	}
	if filter.DateFrom != nil {
		where = append(where, fmt.Sprintf("created_at >= $%d", idx))
		args = append(args, *filter.DateFrom)
		idx++
	}
	if filter.DateTo != nil {
		where = append(where, fmt.Sprintf("created_at <= $%d", idx))
		args = append(args, *filter.DateTo)
		idx++
	}

	whereClause := strings.Join(where, " AND ")
	offset := (filter.Page - 1) * filter.PageSize
	args = append(args, filter.PageSize, offset)

	query := fmt.Sprintf(
		`SELECT `+columns+`, COUNT(*) OVER() AS total_count FROM notifications WHERE %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`,
		whereClause, idx, idx+1)

	var rows []notificationWithCount
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, 0, err
	}
	if len(rows) == 0 {
		return []*domain.Notification{}, 0, nil
	}

	total := rows[0].TotalCount
	notifications := make([]*domain.Notification, len(rows))
	for i := range rows {
		n := rows[i].Notification
		notifications[i] = &n
	}
	return notifications, total, nil
}

func (r *notificationRepository) GetDueScheduled(ctx context.Context, limit int) ([]*domain.Notification, error) {
	var notifications []*domain.Notification
	err := r.db.SelectContext(ctx, &notifications,
		`SELECT `+columns+` FROM notifications WHERE status = 'pending' AND scheduled_at IS NOT NULL AND scheduled_at <= $1 ORDER BY priority ASC, scheduled_at ASC LIMIT $2`,
		time.Now(), limit)
	return notifications, err
}

func (r *notificationRepository) BulkUpdateStatus(ctx context.Context, ids []string, status domain.Status) error {
	if len(ids) == 0 {
		return nil
	}
	query, args, err := sqlx.In(
		`UPDATE notifications SET status = ?, updated_at = ? WHERE id IN (?)`,
		status, time.Now(), ids,
	)
	if err != nil {
		return err
	}
	query = r.db.Rebind(query)
	_, err = r.db.ExecContext(ctx, query, args...)
	return err
}

func (r *notificationRepository) Archive(ctx context.Context, olderThan time.Time, batchSize int) (int64, error) {
	res, err := r.db.ExecContext(ctx, `
		WITH to_archive AS (
			SELECT id FROM notifications
			WHERE status IN ('delivered', 'failed', 'cancelled')
			AND created_at < $1
			LIMIT $2
		), moved AS (
			DELETE FROM notifications
			WHERE id IN (SELECT id FROM to_archive)
			RETURNING `+columns+`
		)
		INSERT INTO notifications_archive (`+columns+`)
		SELECT `+columns+` FROM moved`,
		olderThan, batchSize)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
