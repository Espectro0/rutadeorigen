package authorizer_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"rutadeorigen/qr/internal/application/output/ports"
	"rutadeorigen/qr/internal/domain/actorview"
	"rutadeorigen/qr/internal/domain/lotview"
	"rutadeorigen/qr/internal/infrastructure/output/external/authorizer"
)

var roles = []actorview.Role{
	actorview.RoleOrganization, actorview.RoleProducer, actorview.RoleRoaster, actorview.RoleBrand, actorview.RoleUnknown, "",
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

func TestCasbinReadQRImageIsOnlyForTheCustodianBrand(t *testing.T) {
	c := newCasbin(t)

	for _, role := range roles {
		for _, stage := range []lotview.Stage{lotview.StageFarm, lotview.StageRoasting, lotview.StagePackaging} {
			actorID := uuid.New()
			resource := ports.Resource{Type: ports.ResourceLot, ID: uuid.New(), CustodianID: actorID, Stage: stage}

			got := decide(t, c, ports.Subject{ActorID: actorID, Role: role}, ports.ActionReadQRImage, resource)

			if want := role == actorview.RoleBrand; got != want {
				t.Errorf("role %q at %s as custodian: got %v, want %v", role, stage, got, want)
			}
		}
	}
}

func TestCasbinReadQRImageDeniesWhoIsNotTheCustodian(t *testing.T) {
	c := newCasbin(t)
	resource := ports.Resource{Type: ports.ResourceLot, ID: uuid.New(), CustodianID: uuid.New(), Stage: lotview.StagePackaging}

	for _, role := range roles {
		if decide(t, c, ports.Subject{ActorID: uuid.New(), Role: role}, ports.ActionReadQRImage, resource) {
			t.Errorf("role %q must not read the image of a lot it does not hold", role)
		}
	}
}

func TestCasbinReadQRImageDeniesAnUnknownActor(t *testing.T) {
	c := newCasbin(t)
	resource := ports.Resource{Type: ports.ResourceLot, CustodianID: uuid.Nil, Stage: lotview.StagePackaging}

	if decide(t, c, ports.Subject{ActorID: uuid.Nil, Role: actorview.RoleBrand}, ports.ActionReadQRImage, resource) {
		t.Error("the nil actor must never match the custodian")
	}
}

func TestCasbinReadQRStatsIsOnlyForTheOrganization(t *testing.T) {
	c := newCasbin(t)
	resource := ports.Resource{Type: ports.ResourceLot, ID: uuid.New(), CustodianID: uuid.New(), Stage: lotview.StagePackaging}

	for _, role := range roles {
		got := decide(t, c, ports.Subject{ActorID: uuid.New(), Role: role}, ports.ActionReadQRStats, resource)

		if want := role == actorview.RoleOrganization; got != want {
			t.Errorf("role %q: got %v, want %v", role, got, want)
		}
	}
}

func TestCasbinDeniesUnknownActionsAndResources(t *testing.T) {
	c := newCasbin(t)
	subject := ports.Subject{ActorID: uuid.New(), Role: actorview.RoleOrganization}

	if decide(t, c, subject, ports.Action("lot:void"), ports.Resource{Type: ports.ResourceLot}) {
		t.Error("an action outside the policy must be denied")
	}
	if decide(t, c, subject, ports.ActionReadQRStats, ports.Resource{Type: ports.ResourceType("farm")}) {
		t.Error("a resource outside the policy must be denied")
	}
}
