package domain

import (
	"time"
)

type Channel string
type Status string
type Priority string

const (
	ChannelSMS   Channel = "sms"
	ChannelEmail Channel = "email"
	ChannelPush  Channel = "push"
)

const (
	StatusPending    Status = "pending"
	StatusQueued     Status = "queued"
	StatusProcessing Status = "processing"
	StatusDelivered  Status = "delivered"
	StatusFailed     Status = "failed"
	StatusCancelled  Status = "cancelled"
)

const (
	PriorityHigh   Priority = "high"
	PriorityNormal Priority = "normal"
	PriorityLow    Priority = "low"
)

type Notification struct {
	ID             string     `db:"id" json:"id"`
	BatchID        *string    `db:"batch_id" json:"batch_id,omitempty"`
	Recipient      string     `db:"recipient" json:"recipient"`
	Channel        Channel    `db:"channel" json:"channel"`
	Content        string     `db:"content" json:"content"`
	Priority       Priority   `db:"priority" json:"priority"`
	Status         Status     `db:"status" json:"status"`
	IdempotencyKey *string    `db:"idempotency_key" json:"idempotency_key,omitempty"`
	ProviderMsgID  *string    `db:"provider_msg_id" json:"provider_msg_id,omitempty"`
	RetryCount     int        `db:"retry_count" json:"retry_count"`
	ScheduledAt    *time.Time `db:"scheduled_at" json:"scheduled_at,omitempty"`
	CreatedAt      time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt      time.Time  `db:"updated_at" json:"updated_at"`
}

type CreateNotificationInput struct {
	Recipient      string            `json:"recipient" binding:"required"`
	Channel        Channel           `json:"channel" binding:"required,oneof=sms email push"`
	Content        string            `json:"content"`
	Priority       Priority          `json:"priority" binding:"omitempty,oneof=high normal low"`
	IdempotencyKey *string           `json:"idempotency_key"`
	ScheduledAt    *time.Time        `json:"scheduled_at"`
	TemplateID     *string           `json:"template_id"`
	Variables      map[string]string `json:"variables"`
}

type CreateBatchInput struct {
	Notifications []CreateNotificationInput `json:"notifications" binding:"required,min=1,max=1000"`
}

type ListFilter struct {
	Status    *Status   `form:"status"`
	Channel   *Channel  `form:"channel"`
	BatchID   *string   `form:"batch_id"`
	DateFrom  *time.Time `form:"date_from" time_format:"2006-01-02T15:04:05Z07:00"`
	DateTo    *time.Time `form:"date_to" time_format:"2006-01-02T15:04:05Z07:00"`
	Page      int       `form:"page,default=1"`
	PageSize  int       `form:"page_size,default=20"`
}

type BatchResult struct {
	BatchID string `json:"batch_id"`
	Total   int    `json:"total"`
	Queued  int    `json:"queued"`
}
