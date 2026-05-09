package domain_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/kenanabbak/notification-management-api/internal/domain"
)

type ValidationSuite struct {
	suite.Suite
}

func TestValidationSuite(t *testing.T) {
	suite.Run(t, new(ValidationSuite))
}

func (s *ValidationSuite) TestValidateContent() {
	cases := []struct {
		name    string
		channel domain.Channel
		content string
		wantErr error
	}{
		{"sms_at_limit", domain.ChannelSMS, strings.Repeat("a", 160), nil},
		{"sms_over_limit", domain.ChannelSMS, strings.Repeat("a", 161), domain.ErrContentTooLong},
		{"push_at_limit", domain.ChannelPush, strings.Repeat("a", 256), nil},
		{"push_over_limit", domain.ChannelPush, strings.Repeat("a", 257), domain.ErrContentTooLong},
		{"email_at_limit", domain.ChannelEmail, strings.Repeat("a", 10_000), nil},
		{"email_over_limit", domain.ChannelEmail, strings.Repeat("a", 10_001), domain.ErrContentTooLong},
		{"sms_unicode_at_limit", domain.ChannelSMS, strings.Repeat("😀", 160), nil},
		{"sms_unicode_over_limit", domain.ChannelSMS, strings.Repeat("😀", 161), domain.ErrContentTooLong},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			err := domain.ValidateContent(tc.channel, tc.content)
			if tc.wantErr != nil {
				s.ErrorIs(err, tc.wantErr)
			} else {
				s.NoError(err)
			}
		})
	}
}

func (s *ValidationSuite) TestValidateRecipient() {
	cases := []struct {
		name      string
		channel   domain.Channel
		recipient string
		wantErr   error
	}{
		{"sms_e164_turkey", domain.ChannelSMS, "+905551234567", nil},
		{"sms_e164_us", domain.ChannelSMS, "+12025551234", nil},
		{"sms_no_plus", domain.ChannelSMS, "05551234567", domain.ErrInvalidRecipient},
		{"sms_letters", domain.ChannelSMS, "abc", domain.ErrInvalidRecipient},
		{"sms_empty", domain.ChannelSMS, "", domain.ErrInvalidRecipient},
		{"email_valid", domain.ChannelEmail, "user@example.com", nil},
		{"email_plus_tag", domain.ChannelEmail, "user+tag@domain.co.uk", nil},
		{"email_no_at", domain.ChannelEmail, "not-an-email", domain.ErrInvalidRecipient},
		{"email_no_local", domain.ChannelEmail, "@nodomain", domain.ErrInvalidRecipient},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			err := domain.ValidateRecipient(tc.channel, tc.recipient)
			if tc.wantErr != nil {
				s.ErrorIs(err, tc.wantErr)
			} else {
				s.NoError(err)
			}
		})
	}
}
