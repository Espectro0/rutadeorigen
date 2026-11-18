package sendevidencenotification

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"rutadeorigen/notifications/internal/application/input/dto/request"
	sendnotificationuc "rutadeorigen/notifications/internal/application/usecases/sendevidencenotification"
	"rutadeorigen/notifications/internal/domain/notification"
	"rutadeorigen/notifications/internal/platform/exception"
)

type recordingUseCase struct {
	received []sendnotificationuc.Command
	err      error
}

func (r *recordingUseCase) Execute(_ context.Context, cmd sendnotificationuc.Command) error {
	r.received = append(r.received, cmd)
	return r.err
}

func validInput(eventType string) request.SendEvidenceNotification {
	return request.SendEvidenceNotification{
		MessageID:     uuid.NewString(),
		EventType:     eventType,
		EvidenceID:    uuid.NewString(),
		LotID:         uuid.NewString(),
		LotCode:       " LT-001 ",
		Stage:         " farm ",
		FileName:      " foto.jpg ",
		Reason:        " La imagen está dañada ",
		UploaderName:  " Maria Lopez ",
		UploaderEmail: " maria@example.com ",
	}
}

func TestExecuteBuildsTheCommandForEachEventType(t *testing.T) {
	tests := map[string]notification.Kind{
		" evidence.processed ": notification.KindEvidenceProcessed,
		"evidence.rejected":    notification.KindEvidenceRejected,
	}

	for eventType, wantKind := range tests {
		t.Run(eventType, func(t *testing.T) {
			useCase := &recordingUseCase{}
			input := validInput(eventType)

			if err := New(useCase).Execute(context.Background(), input); err != nil {
				t.Fatal(err)
			}

			if len(useCase.received) != 1 {
				t.Fatalf("want one call, got %d", len(useCase.received))
			}
			cmd := useCase.received[0]
			if cmd.Kind != wantKind || cmd.EvidenceID.String() != input.EvidenceID || cmd.LotID.String() != input.LotID {
				t.Errorf("unexpected command %+v", cmd)
			}
			if cmd.LotCode != "LT-001" || cmd.Stage != "farm" || cmd.FileName != "foto.jpg" || cmd.Reason != "La imagen está dañada" {
				t.Errorf("fields must be trimmed: %+v", cmd)
			}
			if cmd.RecipientName != "Maria Lopez" || cmd.RecipientEmail != "maria@example.com" {
				t.Errorf("recipient must be trimmed: %+v", cmd)
			}
		})
	}
}

func TestExecuteRejectsInvalidInput(t *testing.T) {
	tests := map[string]func(*request.SendEvidenceNotification){
		"unknown event type": func(in *request.SendEvidenceNotification) { in.EventType = "lot.created" },
		"empty event type":   func(in *request.SendEvidenceNotification) { in.EventType = "" },
		"message id":         func(in *request.SendEvidenceNotification) { in.MessageID = "" },
		"evidence id":        func(in *request.SendEvidenceNotification) { in.EvidenceID = "not-a-uuid" },
		"lot id":             func(in *request.SendEvidenceNotification) { in.LotID = "" },
	}

	for name, change := range tests {
		t.Run(name, func(t *testing.T) {
			useCase := &recordingUseCase{}
			input := validInput("evidence.processed")
			change(&input)

			err := New(useCase).Execute(context.Background(), input)

			if !exception.HasCode(err, exception.CodeEventInvalidPayload) {
				t.Errorf("want an invalid payload error, got %v", err)
			}
			if len(useCase.received) != 0 {
				t.Error("the use case must not be called")
			}
		})
	}
}

func TestExecutePropagatesUseCaseErrors(t *testing.T) {
	boom := errors.New("db down")
	useCase := &recordingUseCase{err: boom}

	err := New(useCase).Execute(context.Background(), validInput("evidence.processed"))

	if !errors.Is(err, boom) {
		t.Errorf("want the use case error, got %v", err)
	}
}
