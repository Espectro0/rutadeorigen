package listactors_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"rutadeorigen/lib/requestcontext"
	"rutadeorigen/traceability/internal/application/output/ports"
	"rutadeorigen/traceability/internal/application/usecases/listactors"
	"rutadeorigen/traceability/internal/domain/actor"
	"rutadeorigen/traceability/internal/platform/exception"
)

var errInfra = errors.New("infrastructure failure")

type fakeActors struct {
	byID      map[uuid.UUID]*actor.Actor
	listed    []*actor.Actor
	findErr   error
	listErr   error
	listCalls []actor.Type
}

func (r *fakeActors) FindByID(_ context.Context, id uuid.UUID) (*actor.Actor, error) {
	if r.findErr != nil {
		return actor.Default(), r.findErr
	}
	if a, ok := r.byID[id]; ok {
		return a, nil
	}
	return actor.Default(), nil
}

func (r *fakeActors) FindByUserID(_ context.Context, _ string) (*actor.Actor, error) {
	return actor.Default(), nil
}

func (r *fakeActors) ListByType(_ context.Context, actorType actor.Type) ([]*actor.Actor, error) {
	r.listCalls = append(r.listCalls, actorType)
	if r.listErr != nil {
		return nil, r.listErr
	}
	return r.listed, nil
}

func (r *fakeActors) Save(_ context.Context, _ *actor.Actor) error {
	return nil
}

type fakeAuthorizer struct {
	allowed  bool
	err      error
	subject  ports.Subject
	action   ports.Action
	resource ports.Resource
}

func (a *fakeAuthorizer) IsAllowed(_ context.Context, s ports.Subject, act ports.Action, r ports.Resource) (bool, error) {
	a.subject, a.action, a.resource = s, act, r
	return a.allowed, a.err
}

type scenario struct {
	requester  *actor.Actor
	actors     *fakeActors
	authorizer *fakeAuthorizer
}

func newActor(t *testing.T, userID string, actorType actor.Type) *actor.Actor {
	t.Helper()

	a, err := actor.New(userID, actorType, "Actor "+userID, userID+"@example.com")
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func newScenario(t *testing.T) *scenario {
	t.Helper()

	requester := newActor(t, "user_01", actor.TypeProducer)
	return &scenario{
		requester:  requester,
		actors:     &fakeActors{byID: map[uuid.UUID]*actor.Actor{requester.ID(): requester}},
		authorizer: &fakeAuthorizer{allowed: true},
	}
}

func (s *scenario) execute(q listactors.Query) ([]*actor.Actor, error) {
	ctx := requestcontext.WithActorID(context.Background(), s.requester.ID())
	return listactors.New(s.actors, s.authorizer).Execute(ctx, q)
}

func assertEmpty(t *testing.T, got []*actor.Actor) {
	t.Helper()
	if got == nil || len(got) != 0 {
		t.Errorf("got %v, want an empty non nil list", got)
	}
}

func TestListActorsReturnsTheActiveActorsOfTheType(t *testing.T) {
	s := newScenario(t)
	first := newActor(t, "user_02", actor.TypeRoaster)
	second := newActor(t, "user_03", actor.TypeRoaster)
	inactive := actor.Restore(uuid.New(), "user_04", actor.TypeRoaster, "Inactive roaster", "user_04@example.com",
		actor.StatusInactive, time.Now())
	s.actors.listed = []*actor.Actor{first, inactive, second, actor.Default()}

	got, err := s.execute(listactors.Query{Type: actor.TypeRoaster})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 || got[0] != first || got[1] != second {
		t.Errorf("got %v, want the two active roasters in order", got)
	}
	if len(s.actors.listCalls) != 1 || s.actors.listCalls[0] != actor.TypeRoaster {
		t.Errorf("unexpected repository calls %v", s.actors.listCalls)
	}
}

func TestListActorsReturnsAnEmptyListWhenNothingMatches(t *testing.T) {
	s := newScenario(t)

	got, err := s.execute(listactors.Query{Type: actor.TypeBrand})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertEmpty(t, got)
}

func TestListActorsAsksABACForTheActorDirectory(t *testing.T) {
	s := newScenario(t)

	_, err := s.execute(listactors.Query{Type: actor.TypeRoaster})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	a := s.authorizer
	if a.action != ports.ActionReadActor || a.resource.Type != ports.ResourceActor {
		t.Errorf("unexpected action %v or resource %+v", a.action, a.resource)
	}
	if a.subject.ActorID != s.requester.ID() || a.subject.Role != actor.TypeProducer {
		t.Error("unexpected subject")
	}
}

func TestListActorsErrors(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*scenario)
		query listactors.Query
		check func(*testing.T, error)
	}{
		{
			name:  "the requester does not exist",
			setup: func(s *scenario) { s.actors.byID = map[uuid.UUID]*actor.Actor{} },
			query: listactors.Query{Type: actor.TypeRoaster},
			check: func(t *testing.T, err error) {
				if !exception.HasCode(err, exception.CodeActorNotFound) {
					t.Errorf("got %v, want code %q", err, exception.CodeActorNotFound)
				}
			},
		},
		{
			name:  "ABAC denies",
			setup: func(s *scenario) { s.authorizer.allowed = false },
			query: listactors.Query{Type: actor.TypeRoaster},
			check: func(t *testing.T, err error) {
				if !exception.HasCode(err, exception.CodeForbidden) {
					t.Errorf("got %v, want code %q", err, exception.CodeForbidden)
				}
			},
		},
		{
			name:  "the type is unknown",
			setup: func(*scenario) {},
			query: listactors.Query{Type: actor.TypeUnknown},
			check: func(t *testing.T, err error) {
				if !exception.HasCode(err, exception.CodeActorInvalidType) {
					t.Errorf("got %v, want code %q", err, exception.CodeActorInvalidType)
				}
			},
		},
		{
			name:  "the requester lookup fails",
			setup: func(s *scenario) { s.actors.findErr = errInfra },
			query: listactors.Query{Type: actor.TypeRoaster},
			check: assertInternal,
		},
		{
			name:  "ABAC fails",
			setup: func(s *scenario) { s.authorizer.err = errInfra },
			query: listactors.Query{Type: actor.TypeRoaster},
			check: assertInternal,
		},
		{
			name:  "the listing fails",
			setup: func(s *scenario) { s.actors.listErr = errInfra },
			query: listactors.Query{Type: actor.TypeRoaster},
			check: assertInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newScenario(t)
			tt.setup(s)

			got, err := s.execute(tt.query)

			tt.check(t, err)
			assertEmpty(t, got)
		})
	}
}

func assertInternal(t *testing.T, err error) {
	t.Helper()

	e, ok := exception.From(err)
	if !ok || e.Kind() != exception.KindInternal {
		t.Errorf("expected an internal error, got %v", err)
	}
	if !errors.Is(err, errInfra) {
		t.Error("expected the original cause to be kept")
	}
}

func TestListActorsDoesNotQueryTheRepositoryOnEarlyFailures(t *testing.T) {
	s := newScenario(t)
	s.authorizer.allowed = false

	_, _ = s.execute(listactors.Query{Type: actor.TypeRoaster})

	if len(s.actors.listCalls) != 0 {
		t.Errorf("got %d repository calls, want 0", len(s.actors.listCalls))
	}
}
