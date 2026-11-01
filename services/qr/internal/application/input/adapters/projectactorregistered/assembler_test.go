package projectactorregistered_test

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"rutadeorigen/qr/internal/application/input/adapters/projectactorregistered"
	"rutadeorigen/qr/internal/application/input/dto/request"
	"rutadeorigen/qr/internal/domain/actorview"
	"rutadeorigen/qr/internal/platform/exception"
)

func validRequest() request.ProjectActorRegistered {
	return request.ProjectActorRegistered{
		MessageID:  uuid.NewString(),
		EventType:  " actor.registered ",
		Version:    1,
		OccurredAt: "2026-10-01T09:00:00Z",
		Data: request.ActorRegisteredData{
			ActorID: uuid.NewString(),
			UserID:  " user_01 ",
			Type:    "Roaster",
		},
	}
}

func TestToDomainMapsTheRequest(t *testing.T) {
	req := validRequest()

	cmd, err := projectactorregistered.Assembler{}.ToDomain(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cmd.MessageID.String() != req.MessageID || cmd.ActorID.String() != req.Data.ActorID {
		t.Errorf("unexpected ids %+v", cmd)
	}
	if cmd.EventType != "actor.registered" || cmd.UserID != "user_01" || cmd.Role != actorview.RoleRoaster {
		t.Errorf("unexpected values %+v", cmd)
	}
	if !cmd.OccurredAt.Equal(time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC)) {
		t.Errorf("unexpected occurred at %v", cmd.OccurredAt)
	}
}

func TestToDomainKeepsUnknownRolesForTheUseCaseToReject(t *testing.T) {
	req := validRequest()
	req.Data.Type = "admin"

	cmd, err := projectactorregistered.Assembler{}.ToDomain(req)

	if err != nil || cmd.Role != actorview.RoleUnknown {
		t.Errorf("got %+v and %v", cmd, err)
	}
}

func TestToDomainAcceptsAMissingOccurredAt(t *testing.T) {
	req := validRequest()
	req.OccurredAt = ""

	if _, err := (projectactorregistered.Assembler{}).ToDomain(req); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestToDomainRejectsInvalidPayloads(t *testing.T) {
	for name, mutate := range map[string]func(*request.ProjectActorRegistered){
		"message id":  func(r *request.ProjectActorRegistered) { r.MessageID = "x" },
		"actor id":    func(r *request.ProjectActorRegistered) { r.Data.ActorID = "x" },
		"occurred at": func(r *request.ProjectActorRegistered) { r.OccurredAt = "yesterday" },
		"empty ids":   func(r *request.ProjectActorRegistered) { r.MessageID, r.Data.ActorID = "", "" },
	} {
		t.Run(name, func(t *testing.T) {
			req := validRequest()
			mutate(&req)

			_, err := projectactorregistered.Assembler{}.ToDomain(req)

			if !exception.HasCode(err, exception.CodeEventInvalidPayload) {
				t.Errorf("got error %v, want %s", err, exception.CodeEventInvalidPayload)
			}
		})
	}
}
