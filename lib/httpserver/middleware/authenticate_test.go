package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"rutadeorigen/lib/requestcontext"
)

type authenticatorFake struct {
	identity requestcontext.Identity
	err      error
	calls    int
}

func (a *authenticatorFake) Authenticate(_ http.ResponseWriter, _ *http.Request) (requestcontext.Identity, error) {
	a.calls++
	return a.identity, a.err
}

type authenticatorWriterFake struct{}

func (authenticatorWriterFake) Authenticate(w http.ResponseWriter, _ *http.Request) (requestcontext.Identity, error) {
	http.SetCookie(w, &http.Cookie{Name: "session", Value: "refreshed"})
	return requestcontext.Identity{UserID: "user_01H"}, nil
}

func TestAuthenticateSetsUserIDAndEmailAndCallsNext(t *testing.T) {
	fake := &authenticatorFake{identity: requestcontext.Identity{UserID: "user_01H", Email: "producer@example.com"}}
	var gotUserID, gotEmail string
	nextCalled := false
	rejectCalled := false

	handler := Authenticate(fake, func(_ http.ResponseWriter, _ *http.Request, _ error) {
		rejectCalled = true
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		gotUserID = requestcontext.UserID(r.Context())
		gotEmail = requestcontext.Email(r.Context())
		w.WriteHeader(http.StatusNoContent)
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if !nextCalled {
		t.Fatal("next handler was not called")
	}
	if rejectCalled {
		t.Fatal("reject must not be called")
	}
	if gotUserID != "user_01H" {
		t.Fatalf("UserID = %q, want %q", gotUserID, "user_01H")
	}
	if gotEmail != "producer@example.com" {
		t.Fatalf("Email = %q, want %q", gotEmail, "producer@example.com")
	}
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if fake.calls != 1 {
		t.Fatalf("authenticator calls = %d, want 1", fake.calls)
	}
}

func TestAuthenticateRejectsAndSkipsNextOnError(t *testing.T) {
	errUnauthenticated := errors.New("unauthenticated")
	errUnavailable := errors.New("unavailable")
	errUnknown := errors.New("boom")

	tests := []struct {
		name string
		err  error
	}{
		{name: "unauthenticated", err: errUnauthenticated},
		{name: "unavailable", err: errUnavailable},
		{name: "unknown", err: errUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &authenticatorFake{err: tt.err}
			nextCalled := false
			var gotErr error

			handler := Authenticate(fake, func(w http.ResponseWriter, _ *http.Request, err error) {
				gotErr = err
				w.WriteHeader(http.StatusUnauthorized)
			})(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
				nextCalled = true
			}))

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

			if nextCalled {
				t.Fatal("next handler must not be called")
			}
			if !errors.Is(gotErr, tt.err) {
				t.Fatalf("reject error = %v, want %v", gotErr, tt.err)
			}
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
			}
		})
	}
}

func TestAuthenticateDoesNotSetIdentityOnError(t *testing.T) {
	fake := &authenticatorFake{identity: requestcontext.Identity{UserID: "user_01H", Email: "producer@example.com"}, err: errors.New("boom")}
	var gotUserID, gotEmail string

	handler := Authenticate(fake, func(_ http.ResponseWriter, r *http.Request, _ error) {
		gotUserID = requestcontext.UserID(r.Context())
		gotEmail = requestcontext.Email(r.Context())
	})(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {}))

	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

	if gotUserID != "" {
		t.Fatalf("UserID = %q, want empty", gotUserID)
	}
	if gotEmail != "" {
		t.Fatalf("Email = %q, want empty", gotEmail)
	}
}

func TestAuthenticateAcceptsAnIdentityWithoutEmail(t *testing.T) {
	fake := &authenticatorFake{identity: requestcontext.Identity{UserID: "user_01H"}}
	var gotUserID, gotEmail string

	handler := Authenticate(fake, func(_ http.ResponseWriter, _ *http.Request, _ error) {})(
		http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			gotUserID = requestcontext.UserID(r.Context())
			gotEmail = requestcontext.Email(r.Context())
		}),
	)

	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

	if gotUserID != "user_01H" || gotEmail != "" {
		t.Fatalf("got (%q, %q), want (user_01H, empty)", gotUserID, gotEmail)
	}
}

func TestAuthenticatePassesResponseWriterToAuthenticator(t *testing.T) {
	handler := Authenticate(authenticatorWriterFake{}, func(_ http.ResponseWriter, _ *http.Request, _ error) {})(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}),
	)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	cookies := rec.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != "session" || cookies[0].Value != "refreshed" {
		t.Fatalf("cookies = %v, want refreshed session cookie", cookies)
	}
}

func TestAuthenticatePreservesOriginalContextValues(t *testing.T) {
	fake := &authenticatorFake{identity: requestcontext.Identity{UserID: "user_01H", Email: "producer@example.com"}}
	var gotCorrelation string

	handler := Authenticate(fake, func(_ http.ResponseWriter, _ *http.Request, _ error) {})(
		http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			gotCorrelation = requestcontext.CorrelationID(r.Context())
		}),
	)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(requestcontext.WithCorrelationID(req.Context(), "corr-1"))
	handler.ServeHTTP(httptest.NewRecorder(), req)

	if gotCorrelation != "corr-1" {
		t.Fatalf("CorrelationID = %q, want %q", gotCorrelation, "corr-1")
	}
}
