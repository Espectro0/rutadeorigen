package notification_test

import (
	"strings"
	"testing"

	"rutadeorigen/notifications/internal/domain/notification"
	"rutadeorigen/notifications/internal/platform/exception"
)

func TestNewRecipientNormalizesTheAddress(t *testing.T) {
	r, err := notification.NewRecipient("  Maria Lopez  ", "  Maria.Lopez@Example.COM ")

	if err != nil || r.IsDefault() {
		t.Fatalf("got %+v %v", r, err)
	}
	if r.Name() != "Maria Lopez" || r.Email() != "maria.lopez@example.com" {
		t.Errorf("got name %q and email %q", r.Name(), r.Email())
	}
}

func TestNewRecipientAcceptsAMissingName(t *testing.T) {
	r, err := notification.NewRecipient("", "maria@example.com")

	if err != nil || r.Name() != "" || r.Email() != "maria@example.com" {
		t.Errorf("got %+v %v", r, err)
	}
}

func TestNewRecipientLimitsTheName(t *testing.T) {
	r, err := notification.NewRecipient(strings.Repeat("ñ", 300), "maria@example.com")

	if err != nil {
		t.Fatal(err)
	}
	if got := len([]rune(r.Name())); got != 120 {
		t.Errorf("name length: got %d, want 120", got)
	}
}

func TestNewRecipientRejectsInvalidAddresses(t *testing.T) {
	tooLong := strings.Repeat("a", 250) + "@example.com"
	addresses := map[string]string{
		"empty":              "",
		"blank":              "   ",
		"no at sign":         "maria.example.com",
		"no domain":          "maria@",
		"no local part":      "@example.com",
		"domain without dot": "maria@localhost",
		"display name form":  "Maria <maria@example.com>",
		"two addresses":      "a@example.com, b@example.com",
		"too long":           tooLong,
	}

	for name, address := range addresses {
		t.Run(name, func(t *testing.T) {
			r, err := notification.NewRecipient("Maria", address)

			if !exception.HasCode(err, exception.CodeInvalidRecipient) {
				t.Errorf("error: got %v, want %s", err, exception.CodeInvalidRecipient)
			}
			if !r.IsDefault() {
				t.Errorf("want the default recipient, got %+v", r)
			}
		})
	}
}

func TestDefaultRecipient(t *testing.T) {
	r := notification.DefaultRecipient()

	if !r.IsDefault() || r.Email() != "" || r.Name() != "" {
		t.Errorf("unexpected default %+v", r)
	}
}
