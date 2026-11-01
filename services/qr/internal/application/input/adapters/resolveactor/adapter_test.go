package resolveactor_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"rutadeorigen/qr/internal/application/input/adapters/resolveactor"
	"rutadeorigen/qr/internal/application/input/dto/request"
	resolveactoruc "rutadeorigen/qr/internal/application/usecases/resolveactor"
	"rutadeorigen/qr/internal/domain/actorview"
)

type ucFake struct {
	got  resolveactoruc.Command
	view *actorview.ActorView
	err  error
}

func (f *ucFake) Execute(_ context.Context, cmd resolveactoruc.Command) (*actorview.ActorView, error) {
	f.got = cmd
	if f.err != nil {
		return actorview.Default(), f.err
	}
	return f.view, nil
}

func TestAdapterMapsRequestAndResult(t *testing.T) {
	view, _ := actorview.New(actorview.NewParams{ActorID: uuid.New(), UserID: "user_01", Role: actorview.RoleOrganization})
	fake := &ucFake{view: view}
	actorID := uuid.New()

	got, err := resolveactor.New(fake).Execute(context.Background(), request.ResolveActor{UserID: "user_01", ActorID: actorID.String()})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fake.got.UserID != "user_01" || fake.got.ActorID != actorID {
		t.Errorf("unexpected command %+v", fake.got)
	}
	if got.ActorID != view.ActorID() || got.Role != "organization" {
		t.Errorf("unexpected result %+v", got)
	}
}

func TestAdapterIgnoresAnInvalidActorID(t *testing.T) {
	fake := &ucFake{view: actorview.Default()}

	_, _ = resolveactor.New(fake).Execute(context.Background(), request.ResolveActor{ActorID: "not-a-uuid"})

	if fake.got.ActorID != uuid.Nil {
		t.Errorf("an invalid id must become the nil uuid, got %s", fake.got.ActorID)
	}
}

func TestAdapterPropagatesErrors(t *testing.T) {
	boom := errors.New("boom")

	_, err := resolveactor.New(&ucFake{err: boom}).Execute(context.Background(), request.ResolveActor{UserID: "u"})

	if !errors.Is(err, boom) {
		t.Errorf("expected the use case error, got %v", err)
	}
}
