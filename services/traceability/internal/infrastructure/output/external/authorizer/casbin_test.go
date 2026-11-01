package authorizer_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"rutadeorigen/traceability/internal/application/output/ports"
	"rutadeorigen/traceability/internal/domain/actor"
	"rutadeorigen/traceability/internal/domain/lot"
	"rutadeorigen/traceability/internal/infrastructure/output/external/authorizer"
)

var roles = []actor.Type{actor.TypeOrganization, actor.TypeProducer, actor.TypeRoaster, actor.TypeBrand}

func allowedRoles(roles ...actor.Type) map[actor.Type]bool {
	set := make(map[actor.Type]bool, len(roles))
	for _, role := range roles {
		set[role] = true
	}
	return set
}

func newCasbin(t *testing.T) *authorizer.Casbin {
	t.Helper()
	c, err := authorizer.NewCasbin()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return c
}

func decide(t *testing.T, c *authorizer.Casbin, subject ports.Subject, action ports.Action, resource ports.Resource) bool {
	t.Helper()
	allowed, err := c.IsAllowed(context.Background(), subject, action, resource)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return allowed
}

func TestMustNewCasbinBuilds(t *testing.T) {
	if authorizer.MustNewCasbin() == nil {
		t.Fatal("expected an authorizer")
	}
}

func TestCasbinCreateFarm(t *testing.T) {
	c := newCasbin(t)
	want := allowedRoles(actor.TypeProducer)

	for _, role := range roles {
		subject := ports.Subject{ActorID: uuid.New(), Role: role}
		got := decide(t, c, subject, ports.ActionCreateFarm, ports.Resource{Type: ports.ResourceFarm, OwnerID: subject.ActorID})
		if got != want[role] {
			t.Errorf("role %s: got %v, want %v", role, got, want[role])
		}
	}
}

func TestCasbinReadEveryRole(t *testing.T) {
	c := newCasbin(t)

	for _, role := range roles {
		subject := ports.Subject{ActorID: uuid.New(), Role: role}
		farm := ports.Resource{Type: ports.ResourceFarm, OwnerID: uuid.New()}
		batch := ports.Resource{Type: ports.ResourceLot, ID: uuid.New(), CustodianID: uuid.New()}
		directory := ports.Resource{Type: ports.ResourceActor}

		if !decide(t, c, subject, ports.ActionReadFarm, farm) {
			t.Errorf("role %s must read farms", role)
		}
		if !decide(t, c, subject, ports.ActionReadLot, batch) {
			t.Errorf("role %s must read lots", role)
		}
		if !decide(t, c, subject, ports.ActionReadActor, directory) {
			t.Errorf("role %s must read actors", role)
		}
	}
}

func TestCasbinFarmOwnership(t *testing.T) {
	c := newCasbin(t)
	owner := uuid.New()
	farm := ports.Resource{Type: ports.ResourceFarm, ID: uuid.New(), OwnerID: owner}

	for _, action := range []ports.Action{ports.ActionDeleteFarm, ports.ActionCreateLot} {
		if !decide(t, c, ports.Subject{ActorID: owner, Role: actor.TypeProducer}, action, farm) {
			t.Errorf("%s: owner producer must be allowed", action)
		}
		if decide(t, c, ports.Subject{ActorID: uuid.New(), Role: actor.TypeProducer}, action, farm) {
			t.Errorf("%s: another producer must be denied", action)
		}
		for _, role := range []actor.Type{actor.TypeOrganization, actor.TypeRoaster, actor.TypeBrand} {
			if decide(t, c, ports.Subject{ActorID: owner, Role: role}, action, farm) {
				t.Errorf("%s: role %s must be denied", action, role)
			}
		}
	}
}

func TestCasbinRegisterStage(t *testing.T) {
	c := newCasbin(t)
	stageRoles := map[lot.Stage]actor.Type{
		lot.StageFarm:      actor.TypeProducer,
		lot.StageRoasting:  actor.TypeRoaster,
		lot.StagePackaging: actor.TypeBrand,
	}

	for stage, owner := range stageRoles {
		for _, role := range roles {
			custodian := uuid.New()
			resource := ports.Resource{Type: ports.ResourceLot, ID: uuid.New(), CustodianID: custodian, Stage: stage}

			got := decide(t, c, ports.Subject{ActorID: custodian, Role: role}, ports.ActionRegisterStage, resource)
			if want := role == owner; got != want {
				t.Errorf("stage %s role %s as custodian: got %v, want %v", stage, role, got, want)
			}

			stranger := decide(t, c, ports.Subject{ActorID: uuid.New(), Role: role}, ports.ActionRegisterStage, resource)
			if stranger {
				t.Errorf("stage %s role %s as non custodian must be denied", stage, role)
			}
		}
	}

	custodian := uuid.New()
	for _, stage := range []lot.Stage{lot.StageNone, lot.StageUnknown, ""} {
		resource := ports.Resource{Type: ports.ResourceLot, ID: uuid.New(), CustodianID: custodian, Stage: stage}
		for _, role := range roles {
			if decide(t, c, ports.Subject{ActorID: custodian, Role: role}, ports.ActionRegisterStage, resource) {
				t.Errorf("stage %q role %s must be denied", stage, role)
			}
		}
	}
}

func TestCasbinTransfer(t *testing.T) {
	c := newCasbin(t)
	want := allowedRoles(actor.TypeProducer, actor.TypeRoaster)

	for _, role := range roles {
		custodian := uuid.New()
		resource := ports.Resource{Type: ports.ResourceLot, ID: uuid.New(), CustodianID: custodian}

		if got := decide(t, c, ports.Subject{ActorID: custodian, Role: role}, ports.ActionTransferLot, resource); got != want[role] {
			t.Errorf("role %s as custodian: got %v, want %v", role, got, want[role])
		}
		if decide(t, c, ports.Subject{ActorID: uuid.New(), Role: role}, ports.ActionTransferLot, resource) {
			t.Errorf("role %s as non custodian must be denied", role)
		}
	}
}

func TestCasbinVoidLot(t *testing.T) {
	c := newCasbin(t)
	want := allowedRoles(actor.TypeOrganization)

	for _, role := range roles {
		custodian := uuid.New()
		resource := ports.Resource{Type: ports.ResourceLot, ID: uuid.New(), CustodianID: custodian}

		if got := decide(t, c, ports.Subject{ActorID: custodian, Role: role}, ports.ActionVoidLot, resource); got != want[role] {
			t.Errorf("role %s: got %v, want %v", role, got, want[role])
		}
	}
}

func TestCasbinDeniesByDefault(t *testing.T) {
	c := newCasbin(t)
	id := uuid.New()
	resource := ports.Resource{Type: ports.ResourceLot, ID: uuid.New(), OwnerID: id, CustodianID: id}

	t.Run("unknown action", func(t *testing.T) {
		for _, role := range roles {
			if decide(t, c, ports.Subject{ActorID: id, Role: role}, ports.Action("lot:explode"), resource) {
				t.Errorf("role %s must be denied", role)
			}
		}
	})

	t.Run("unknown role", func(t *testing.T) {
		if decide(t, c, ports.Subject{ActorID: id, Role: actor.TypeUnknown}, ports.ActionVoidLot, resource) {
			t.Error("unknown role must be denied")
		}
	})

	t.Run("empty subject never matches ownership", func(t *testing.T) {
		empty := ports.Resource{Type: ports.ResourceFarm}
		if decide(t, c, ports.Subject{Role: actor.TypeProducer}, ports.ActionDeleteFarm, empty) {
			t.Error("nil actor must not match a nil owner")
		}
	})
}
