package actorview_test

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"rutadeorigen/qr/internal/domain/actorview"
	"rutadeorigen/qr/internal/platform/exception"
)

func TestNewBuildsAnActorView(t *testing.T) {
	id := uuid.New()
	at := time.Date(2026, 10, 1, 8, 0, 0, 0, time.FixedZone("x", -5*3600))

	got, err := actorview.New(actorview.NewParams{ActorID: id, UserID: "  user_01 ", Role: actorview.RoleBrand, CreatedAt: at})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.ActorID() != id || got.UserID() != "user_01" || got.Role() != actorview.RoleBrand {
		t.Errorf("unexpected view %+v", got)
	}
	if !got.CreatedAt().Equal(at) || got.CreatedAt().Location() != time.UTC {
		t.Errorf("created at must be kept in UTC, got %v", got.CreatedAt())
	}
	if got.IsDefault() || got.IsOrganization() {
		t.Error("a brand view is not default nor organization")
	}
}

func TestNewDefaultsCreatedAtToNow(t *testing.T) {
	got, err := actorview.New(actorview.NewParams{ActorID: uuid.New(), UserID: "u", Role: actorview.RoleOrganization})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if time.Since(got.CreatedAt()) > time.Minute || !got.IsOrganization() {
		t.Errorf("unexpected view %+v", got)
	}
}

func TestNewRejectsInvalidData(t *testing.T) {
	tests := []struct {
		name   string
		params actorview.NewParams
		code   exception.Code
	}{
		{"default id", actorview.NewParams{UserID: "u", Role: actorview.RoleBrand}, exception.CodeActorViewInvalidID},
		{"empty user", actorview.NewParams{ActorID: uuid.New(), UserID: "  ", Role: actorview.RoleBrand}, exception.CodeActorViewInvalidUser},
		{"unknown role", actorview.NewParams{ActorID: uuid.New(), UserID: "u", Role: actorview.RoleUnknown}, exception.CodeActorViewInvalidRole},
		{"role outside the catalog", actorview.NewParams{ActorID: uuid.New(), UserID: "u", Role: actorview.Role("admin")}, exception.CodeActorViewInvalidRole},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := actorview.New(tt.params)

			if !exception.HasCode(err, tt.code) {
				t.Errorf("got error %v, want code %s", err, tt.code)
			}
			if !got.IsDefault() {
				t.Error("expected the default view")
			}
		})
	}
}

func TestParseRole(t *testing.T) {
	for in, want := range map[string]actorview.Role{
		"producer": actorview.RoleProducer, " Roaster ": actorview.RoleRoaster, "BRAND": actorview.RoleBrand,
		"organization": actorview.RoleOrganization, "": actorview.RoleUnknown, "admin": actorview.RoleUnknown,
	} {
		if got := actorview.ParseRole(in); got != want {
			t.Errorf("ParseRole(%q) = %q, want %q", in, got, want)
		}
	}
	if actorview.RoleUnknown.IsValid() || !actorview.RoleBrand.IsValid() || actorview.RoleBrand.String() != "brand" {
		t.Error("unexpected role validity or string")
	}
}
