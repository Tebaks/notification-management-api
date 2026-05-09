package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"

	"github.com/kenanabbak/notification-management-api/internal/domain"
)

type templateRepository struct {
	db *sqlx.DB
}

func NewTemplateRepository(db *sqlx.DB) domain.TemplateRepository {
	return &templateRepository{db: db}
}

func (r *templateRepository) Create(ctx context.Context, t *domain.Template) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO templates (id, name, channel, body, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		t.ID, t.Name, t.Channel, t.Body, t.CreatedAt, t.UpdatedAt,
	)
	return err
}

func (r *templateRepository) GetByID(ctx context.Context, id string) (*domain.Template, error) {
	var t domain.Template
	err := r.db.GetContext(ctx, &t,
		`SELECT id, name, channel, body, created_at, updated_at FROM templates WHERE id = $1`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrTemplateNotFound
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *templateRepository) List(ctx context.Context) ([]*domain.Template, error) {
	var templates []*domain.Template
	err := r.db.SelectContext(ctx, &templates,
		`SELECT id, name, channel, body, created_at, updated_at FROM templates ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	return templates, nil
}

func (r *templateRepository) Delete(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM templates WHERE id = $1`, id)
	if err != nil {
		return err
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return domain.ErrTemplateNotFound
	}
	return nil
}
