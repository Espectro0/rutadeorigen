package exception_test

import (
	"errors"
	"testing"

	"rutadeorigen/traceability/internal/platform/exception"
)

func TestNewMessages(t *testing.T) {
	tests := []struct {
		name          string
		userMessage   string
		techMessage   string
		wantUser      string
		wantTechnical string
	}{
		{"both messages given", "Invalid lot", "lot name is empty", "Invalid lot", "lot name is empty"},
		{"messages are trimmed", "  Invalid lot  ", "  lot name is empty  ", "Invalid lot", "lot name is empty"},
		{"empty technical message uses user message", "Invalid lot", "", "Invalid lot", "Invalid lot"},
		{"blank technical message uses user message", "Invalid lot", "   ", "Invalid lot", "Invalid lot"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := exception.New(exception.KindValidation, exception.LayerDomain, tt.userMessage, tt.techMessage, nil)

			if err.UserMessage() != tt.wantUser {
				t.Errorf("UserMessage: got %q, want %q", err.UserMessage(), tt.wantUser)
			}
			if err.TechnicalMessage() != tt.wantTechnical {
				t.Errorf("TechnicalMessage: got %q, want %q", err.TechnicalMessage(), tt.wantTechnical)
			}
		})
	}
}

func TestNewWithEmptyMessagesUsesDefault(t *testing.T) {
	err := exception.New(exception.KindInternal, exception.LayerApplication, "   ", "", nil)

	if err.UserMessage() == "" {
		t.Error("expected a default user message, got empty")
	}
	if err.TechnicalMessage() != err.UserMessage() {
		t.Errorf("expected technical message to fall back to the user message, got %q", err.TechnicalMessage())
	}
}

func TestConstructorsSetKind(t *testing.T) {
	cause := errors.New("cause")

	tests := []struct {
		name string
		err  *exception.Exception
		want exception.Kind
	}{
		{"NewValidation", exception.NewValidation(exception.LayerDomain, "msg", ""), exception.KindValidation},
		{"NewNotFound", exception.NewNotFound(exception.LayerDomain, "msg", ""), exception.KindNotFound},
		{"NewConflict", exception.NewConflict(exception.LayerDomain, "msg", ""), exception.KindConflict},
		{"NewInternal", exception.NewInternal(exception.LayerDomain, "msg", "", cause), exception.KindInternal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Kind() != tt.want {
				t.Errorf("Kind: got %v, want %v", tt.err.Kind(), tt.want)
			}
			if tt.err.Layer() != exception.LayerDomain {
				t.Errorf("Layer: got %v, want %v", tt.err.Layer(), exception.LayerDomain)
			}
		})
	}
}

func TestNewInternalKeepsCause(t *testing.T) {
	cause := errors.New("connection refused")
	err := exception.NewInternal(exception.LayerInfrastructure, "", "save lot", cause)

	if !errors.Is(err, cause) {
		t.Error("expected the cause to be kept")
	}
}
