package access_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"rutadeorigen/lib/requestcontext"
	"rutadeorigen/traceability/internal/application/output/ports"
	"rutadeorigen/traceability/internal/application/usecases/access"
	"rutadeorigen/traceability/internal/domain/actor"
	"rutadeorigen/traceability/internal/platform/exception"
)

var errInfra = errors.New("infrastructure failure")

type fakeAuthorizer struct {
	allowed  bool
	err      error
	calls    int
	subject  ports.Subject
	action   ports.Action
	resource ports.Resource
}

func (a *fakeAuthorizer) IsAllowed(_ context.Context, s ports.Subject, act ports.Action, r ports.Resource) (bool, error) {
	a.calls++
	a.subject, a.action, a.resource = s, act, r
	return a.allowed, a.err
}

type fakePublisher struct {
	err    error
	events []ports.Event
}

func (p *fakePublisher) Publish(_ context.Context, e ports.Event) error {
	if p.err != nil {
		return p.err
	}
	p.events = append(p.events, e)
	return nil
}

func newActor(t *testing.T) *actor.Actor {
	t.Helper()
	a, err := actor.New("user_01", actor.TypeProducer, "Finca La Esperanza SAS", "user_01@example.com")
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func newResource() ports.Resource {
	return ports.Resource{
		Type:        ports.ResourceLot,
		ID:          uuid.New(),
		OwnerID:     uuid.New(),
		CustodianID: uuid.New(),
	}
}

func TestRequireAllowed(t *testing.T) {
	a := newActor(t)
	resource := newResource()
	auth := &fakeAuthorizer{allowed: true}
	publisher := &fakePublisher{}

	err := access.Require(context.Background(), auth, publisher, a, ports.ActionRegisterStage, resource)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if auth.calls != 1 {
		t.Errorf("got %d policy evaluations, want 1", auth.calls)
	}
	if auth.subject.ActorID != a.ID() || auth.subject.Role != actor.TypeProducer {
		t.Errorf("unexpected subject %+v", auth.subject)
	}
	if auth.action != ports.ActionRegisterStage {
		t.Errorf("got action %v, want %v", auth.action, ports.ActionRegisterStage)
	}
	if auth.resource != resource {
		t.Errorf("got resource %+v, want %+v", auth.resource, resource)
	}
	if len(publisher.events) != 0 {
		t.Error("nothing must be published when access is allowed")
	}
}

func TestRequireDenied(t *testing.T) {
	a := newActor(t)
	resource := newResource()
	auth := &fakeAuthorizer{allowed: false}
	publisher := &fakePublisher{}
	ctx := requestcontext.WithCorrelationID(context.Background(), "req-123")

	err := access.Require(ctx, auth, publisher, a, ports.ActionCreateLot, resource)

	e, ok := exception.From(err)
	if !ok || e.Code() != exception.CodeForbidden || e.Kind() != exception.KindForbidden {
		t.Fatalf("got %v, want a forbidden exception", err)
	}
	if len(publisher.events) != 1 {
		t.Fatalf("got %d events, want 1", len(publisher.events))
	}

	event := publisher.events[0]
	if event.Type != ports.EventAccessDenied || event.AggregateType != ports.AggregateLot {
		t.Errorf("unexpected event type %v / %v", event.Type, event.AggregateType)
	}
	if event.AggregateID != uuid.Nil || event.Version != 0 {
		t.Errorf("unexpected aggregate %s / version %d", event.AggregateID, event.Version)
	}
	if event.ActorID != a.ID() || event.ActorRole != actor.TypeProducer.String() {
		t.Error("expected the requester as the event actor")
	}
	if event.CorrelationID != "req-123" {
		t.Errorf("got correlation %q, want %q", event.CorrelationID, "req-123")
	}

	payload, ok := event.Payload.(access.DeniedPayload)
	if !ok {
		t.Fatalf("unexpected payload type %T", event.Payload)
	}
	want := access.DeniedPayload{
		Action:       string(ports.ActionCreateLot),
		ResourceType: string(ports.ResourceLot),
		ResourceID:   resource.ID.String(),
	}
	if payload != want {
		t.Errorf("got %+v, want %+v", payload, want)
	}
}

func TestRequireFailures(t *testing.T) {
	assertInternal := func(t *testing.T, err error) {
		t.Helper()
		e, ok := exception.From(err)
		if !ok || e.Kind() != exception.KindInternal {
			t.Errorf("expected an internal error, got %v", err)
		}
		if !errors.Is(err, errInfra) {
			t.Error("expected the original cause to be kept")
		}
	}

	t.Run("the policy evaluation fails", func(t *testing.T) {
		publisher := &fakePublisher{}

		err := access.Require(context.Background(), &fakeAuthorizer{err: errInfra}, publisher, newActor(t),
			ports.ActionReadLot, newResource())

		assertInternal(t, err)
		if len(publisher.events) != 0 {
			t.Error("nothing must be published when the policy cannot be evaluated")
		}
	})

	t.Run("the policy fails even if it reports allowed", func(t *testing.T) {
		err := access.Require(context.Background(), &fakeAuthorizer{allowed: true, err: errInfra}, &fakePublisher{},
			newActor(t), ports.ActionReadLot, newResource())

		assertInternal(t, err)
	})

	t.Run("publishing AccessDenied fails", func(t *testing.T) {
		err := access.Require(context.Background(), &fakeAuthorizer{allowed: false}, &fakePublisher{err: errInfra},
			newActor(t), ports.ActionReadLot, newResource())

		assertInternal(t, err)
	})
}
