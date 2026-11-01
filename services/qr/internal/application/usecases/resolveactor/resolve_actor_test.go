package resolveactor_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"rutadeorigen/qr/internal/application/usecases/resolveactor"
	"rutadeorigen/qr/internal/domain/actorview"
	"rutadeorigen/qr/internal/platform/exception"
)

var errBackend = errors.New("backend is down")

type fakeActors struct {
	view   *actorview.ActorView
	err    error
	byUser []string
	byID   []uuid.UUID
}

func (a *fakeActors) FindByID(_ context.Context, id uuid.UUID) (*actorview.ActorView, error) {
	a.byID = append(a.byID, id)
	return a.result()
}

func (a *fakeActors) FindByUserID(_ context.Context, userID string) (*actorview.ActorView, error) {
	a.byUser = append(a.byUser, userID)
	return a.result()
}

func (a *fakeActors) Save(context.Context, *actorview.ActorView) error { return nil }

func (a *fakeActors) result() (*actorview.ActorView, error) {
	if a.err != nil {
		return actorview.Default(), a.err
	}
	if a.view == nil {
		return actorview.Default(), nil
	}
	return a.view, nil
}

func brandView(t *testing.T) *actorview.ActorView {
	t.Helper()
	v, err := actorview.New(actorview.NewParams{ActorID: uuid.New(), UserID: "user_01", Role: actorview.RoleBrand})
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func TestResolveActorByUser(t *testing.T) {
	view := brandView(t)
	actors := &fakeActors{view: view}

	got, err := resolveactor.New(actors).Execute(context.Background(), resolveactor.Command{UserID: " user_01 ", ActorID: uuid.New()})

	if err != nil || got != view {
		t.Fatalf("expected the view, got %+v and %v", got, err)
	}
	if len(actors.byUser) != 1 || actors.byUser[0] != "user_01" || len(actors.byID) != 0 {
		t.Errorf("the user must win over the actor id, got %v %v", actors.byUser, actors.byID)
	}
}

func TestResolveActorByID(t *testing.T) {
	view := brandView(t)
	actors := &fakeActors{view: view}

	got, err := resolveactor.New(actors).Execute(context.Background(), resolveactor.Command{ActorID: view.ActorID()})

	if err != nil || got != view {
		t.Fatalf("expected the view, got %+v and %v", got, err)
	}
	if len(actors.byID) != 1 || actors.byID[0] != view.ActorID() || len(actors.byUser) != 0 {
		t.Errorf("unexpected lookups %v %v", actors.byUser, actors.byID)
	}
}

func TestResolveActorWithoutIdentityIsUnauthorized(t *testing.T) {
	actors := &fakeActors{}

	got, err := resolveactor.New(actors).Execute(context.Background(), resolveactor.Command{UserID: "  "})

	if !exception.HasCode(err, exception.CodeUnauthorized) || !got.IsDefault() {
		t.Errorf("got %+v and %v, want unauthorized", got, err)
	}
	if len(actors.byUser)+len(actors.byID) != 0 {
		t.Error("no lookup expected without identity")
	}
}

func TestResolveActorUnknownIsNotFound(t *testing.T) {
	got, err := resolveactor.New(&fakeActors{}).Execute(context.Background(), resolveactor.Command{UserID: "user_99"})

	if !exception.HasCode(err, exception.CodeActorViewNotFound) || !got.IsDefault() {
		t.Errorf("got %+v and %v, want actor view not found", got, err)
	}
}

func TestResolveActorWrapsRepositoryFailures(t *testing.T) {
	_, err := resolveactor.New(&fakeActors{err: errBackend}).Execute(context.Background(), resolveactor.Command{UserID: "user_01"})

	if !errors.Is(err, errBackend) {
		t.Errorf("expected the backend error, got %v", err)
	}
}
