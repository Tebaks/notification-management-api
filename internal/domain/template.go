package domain

import (
	"context"
	"time"
)

type Template struct {
	ID        string    `db:"id" json:"id"`
	Name      string    `db:"name" json:"name"`
	Channel   Channel   `db:"channel" json:"channel"`
	Body      string    `db:"body" json:"body"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

type CreateTemplateInput struct {
	Name    string  `json:"name" binding:"required"`
	Channel Channel `json:"channel" binding:"required,oneof=sms email push"`
	Body    string  `json:"body" binding:"required"`
}

type TemplateService interface {
	Create(ctx context.Context, input CreateTemplateInput) (*Template, error)
	GetByID(ctx context.Context, id string) (*Template, error)
	List(ctx context.Context) ([]*Template, error)
	Delete(ctx context.Context, id string) error
}

type TemplateRepository interface {
	Create(ctx context.Context, t *Template) error
	GetByID(ctx context.Context, id string) (*Template, error)
	List(ctx context.Context) ([]*Template, error)
	Delete(ctx context.Context, id string) error
}
