package notification

import (
	"net/mail"
	"strings"

	"rutadeorigen/notifications/internal/platform/exception"
	"rutadeorigen/notifications/internal/platform/helper"
)

const (
	maxEmailLength = 254
	maxNameLength  = 120
)

type Recipient struct {
	name  string
	email string
}

var defaultRecipient = Recipient{name: helper.EmptyString, email: helper.EmptyString}

func DefaultRecipient() Recipient {
	return defaultRecipient
}

func NewRecipient(name, email string) (Recipient, error) {
	normalized := helper.ApplyLower(helper.ApplyTrim(email))
	if !isDeliverableAddress(normalized) {
		return DefaultRecipient(), exception.New(exception.CodeInvalidRecipient)
	}

	return Recipient{name: limitName(helper.ApplyTrim(name)), email: normalized}, nil
}

func isDeliverableAddress(address string) bool {
	if helper.IsEmpty(address) || len(address) > maxEmailLength {
		return false
	}

	parsed, err := mail.ParseAddress(address)
	if err != nil || parsed.Address != address {
		return false
	}

	at := strings.LastIndex(address, "@")
	return strings.Contains(address[at+1:], ".")
}

func limitName(name string) string {
	runes := []rune(name)
	if len(runes) <= maxNameLength {
		return name
	}
	return string(runes[:maxNameLength])
}

func (r Recipient) Name() string  { return r.name }
func (r Recipient) Email() string { return r.email }

func (r Recipient) IsDefault() bool {
	return helper.IsEmpty(r.email)
}
