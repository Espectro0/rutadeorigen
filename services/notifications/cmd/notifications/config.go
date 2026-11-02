package main

import (
	"errors"
	"strings"

	"rutadeorigen/lib/config"
	"rutadeorigen/notifications/internal/infrastructure/output/external/resend"
)

const (
	notificationsQueue = "notifications.events"
	emailsPerConsumer  = 5
)

var consumedEvents = []string{"evidence.processed", "evidence.rejected"}

func loadResend() (resend.Config, error) {
	var problems []error
	required := func(key string) string {
		value, err := config.Required(key)
		if err != nil {
			problems = append(problems, err)
		}
		return strings.TrimSpace(value)
	}

	cfg := resend.Config{
		APIKey:     required("RESEND_API_KEY"),
		From:       required("RESEND_FROM"),
		SenderName: config.String("RESEND_FROM_NAME", resend.DefaultSenderName),
		BaseURL:    config.String("RESEND_BASE_URL", resend.DefaultBaseURL),
	}

	if len(problems) > 0 {
		return resend.Config{}, errors.Join(problems...)
	}
	return cfg, nil
}

func loadPublicBaseURL() (string, error) {
	return config.Required("PUBLIC_BASE_URL")
}
