package main

import (
	"bytes"
	"image/png"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"rutadeorigen/lib/requestcontext"
	"rutadeorigen/lib/session"
	"rutadeorigen/qr/internal/infrastructure/input/controllers"
)

const (
	sessionCookie     = "ro_session"
	sealedValid       = "sealed-valid"
	sealedExpired     = "sealed-expired"
	sealedUnavailable = "sealed-unavailable"
)

type cookieAuthenticator struct{}

func (cookieAuthenticator) Authenticate(_ http.ResponseWriter, r *http.Request) (requestcontext.Identity, error) {
	cookie, err := r.Cookie(sessionCookie)
	if err != nil {
		return requestcontext.Identity{}, session.ErrUnauthenticated
	}
	switch cookie.Value {
	case sealedValid:
		return requestcontext.Identity{UserID: "user_01", Email: "producer@example.com"}, nil
	case sealedUnavailable:
		return requestcontext.Identity{}, session.ErrAuthUnavailable
	default:
		return requestcontext.Identity{}, session.ErrUnauthenticated
	}
}

func newOIDCHarness(t *testing.T) harness {
	t.Helper()

	return newHarnessWith(t, authn{mode: authModeOIDC, authenticator: cookieAuthenticator{}})
}

func (h harness) getWithSession(path, sealed string, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	if sealed != "" {
		req.AddCookie(&http.Cookie{Name: sessionCookie, Value: sealed})
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.mux.ServeHTTP(rec, req)
	return rec
}

func TestOIDCQRImageRequiresASession(t *testing.T) {
	h := newOIDCHarness(t)
	lotID := uuid.New()
	if err := h.publish(t, uuid.New(), lotID, "packaging"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	path := "/public/lots/" + lotID.String() + "/qr?size=256&format=png"

	tests := []struct {
		name   string
		sealed string
		want   int
	}{
		{"without cookie", "", http.StatusUnauthorized},
		{"expired session", sealedExpired, http.StatusUnauthorized},
		{"provider unavailable", sealedUnavailable, http.StatusServiceUnavailable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if rec := h.getWithSession(path, tt.sealed, nil); rec.Code != tt.want {
				t.Errorf("status: got %d, want %d", rec.Code, tt.want)
			}
		})
	}
}

func TestOIDCQRImageIsServedToAnAuthenticatedUser(t *testing.T) {
	h := newOIDCHarness(t)
	brandID := uuid.New()
	h.publishActor(t, "user_01", brandID, "brand")
	lotID := uuid.New()
	if err := h.publishWithCustodian(t, uuid.New(), lotID, "packaging", brandID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	rec := h.getWithSession("/public/lots/"+lotID.String()+"/qr?size=256&format=png", sealedValid, nil)

	if rec.Code != http.StatusOK || rec.Header().Get("Content-Type") != "image/png" {
		t.Fatalf("expected png, got %d %q", rec.Code, rec.Header().Get("Content-Type"))
	}
	if _, err := png.Decode(bytes.NewReader(rec.Body.Bytes())); err != nil {
		t.Errorf("invalid png: %v", err)
	}
}

func TestOIDCQRImageStillAppliesTheLotRules(t *testing.T) {
	h := newOIDCHarness(t)
	brandID := uuid.New()
	h.publishActor(t, "user_01", brandID, "brand")
	early := uuid.New()
	if err := h.publishWithCustodian(t, uuid.New(), early, "roasting", brandID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rec := h.getWithSession("/public/lots/"+early.String()+"/qr", sealedValid, nil); rec.Code != http.StatusConflict {
		t.Errorf("early stage: expected 409, got %d", rec.Code)
	}
	if rec := h.getWithSession("/public/lots/"+uuid.NewString()+"/qr", sealedValid, nil); rec.Code != http.StatusNotFound {
		t.Errorf("unknown lot: expected 404, got %d", rec.Code)
	}
}

func TestOIDCIgnoresTheActorHeader(t *testing.T) {
	h := newOIDCHarness(t)
	lotID := uuid.New()
	if err := h.publish(t, uuid.New(), lotID, "packaging"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	headers := map[string]string{controllers.ActorIDHeader: uuid.NewString()}

	rec := h.getWithSession("/public/lots/"+lotID.String()+"/qr", "", headers)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("the actor header alone must not authenticate, got %d", rec.Code)
	}
}

func TestOIDCPublicRoutesDoNotNeedASession(t *testing.T) {
	h := newOIDCHarness(t)
	lotID := uuid.New()
	if err := h.publish(t, uuid.New(), lotID, "packaging"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rec := h.getWithSession("/public/lots/"+lotID.String(), "", nil); rec.Code != http.StatusOK {
		t.Errorf("public lot: expected 200, got %d", rec.Code)
	}
	if rec := h.getWithSession("/public/lots/"+lotID.String(), sealedUnavailable, nil); rec.Code != http.StatusOK {
		t.Errorf("public lot with the provider down: expected 200, got %d", rec.Code)
	}
	if rec := h.getWithSession("/l/"+lotID.String(), "", nil); rec.Code != http.StatusMovedPermanently {
		t.Errorf("scan: expected 301, got %d", rec.Code)
	}
}

func TestOIDCQRImageRejectsAUserWhoseActorIsNotProjectedYet(t *testing.T) {
	h := newOIDCHarness(t)
	brandID := uuid.New()
	lotID := uuid.New()
	if err := h.publishWithCustodian(t, uuid.New(), lotID, "packaging", brandID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	rec := h.getWithSession("/public/lots/"+lotID.String()+"/qr", sealedValid, nil)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 until actor.registered arrives, got %d", rec.Code)
	}

	h.publishActor(t, "user_01", brandID, "brand")

	if rec := h.getWithSession("/public/lots/"+lotID.String()+"/qr?format=png", sealedValid, nil); rec.Code != http.StatusOK {
		t.Errorf("expected 200 once the actor is projected, got %d", rec.Code)
	}
}

func TestOIDCQRImageIsDeniedToAUserWithAnotherRole(t *testing.T) {
	h := newOIDCHarness(t)
	producerID := uuid.New()
	h.publishActor(t, "user_01", producerID, "producer")
	lotID := uuid.New()
	if err := h.publishWithCustodian(t, uuid.New(), lotID, "packaging", producerID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	rec := h.getWithSession("/public/lots/"+lotID.String()+"/qr?format=png", sealedValid, nil)

	if rec.Code != http.StatusForbidden {
		t.Errorf("only the brand may read the image, got %d", rec.Code)
	}
}
