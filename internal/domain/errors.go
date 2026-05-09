package domain

import "errors"

var (
	ErrNotFound         = errors.New("notification not found")
	ErrCannotCancel     = errors.New("notification cannot be cancelled in current state")
	ErrDuplicate        = errors.New("duplicate notification: idempotency key already exists")
	ErrContentTooLong   = errors.New("content exceeds channel character limit")
	ErrInvalidRecipient = errors.New("invalid recipient format")
	ErrContentRequired  = errors.New("content is required when no template_id is provided")
	ErrTemplateNotFound = errors.New("template not found")
)

const (
	MaxContentSMS   = 160
	MaxContentPush  = 256
	MaxContentEmail = 10_000
)
