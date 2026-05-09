package domain

import (
	"fmt"

	"github.com/go-playground/validator/v10"
)

var validate = validator.New()

func ValidateContent(channel Channel, content string) error {
	limit := contentLimit(channel)
	if len([]rune(content)) > limit {
		return fmt.Errorf("%w: %s limit is %d characters, got %d",
			ErrContentTooLong, channel, limit, len([]rune(content)))
	}
	return nil
}

func ValidateRecipient(channel Channel, recipient string) error {
	switch channel {
	case ChannelSMS, ChannelPush:
		if err := validate.Var(recipient, "e164"); err != nil {
			return fmt.Errorf("%w: sms/push requires E.164 format (e.g. +905551234567)", ErrInvalidRecipient)
		}
	case ChannelEmail:
		if err := validate.Var(recipient, "email"); err != nil {
			return fmt.Errorf("%w: invalid email address", ErrInvalidRecipient)
		}
	}
	return nil
}

func contentLimit(channel Channel) int {
	switch channel {
	case ChannelSMS:
		return MaxContentSMS
	case ChannelPush:
		return MaxContentPush
	default:
		return MaxContentEmail
	}
}
