package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"rutadeorigen/lib/requestcontext"
)

type authenticatorFake struct {
	userID string
	err    error
	calls  int
}

func (a *authenticatorFake) Authenticate(w http.ResponseWriter, r *http.Request) (string, error) {
	a.calls++
	return a.userID, a.err
}

type authenticatorWriterFake struct{}

func (authenticatorWriterFake) Authenticate(w http.ResponseWriter, r *http.Request) (string, error) {
	http.SetCookie(w, &http.Cookie{Name: "session", Value: "refreshed"})
	return "user_01H", nil
}

func TestAuthenticateSetsUserIDAndCallsNext(t *testing.T) {
	fake := &authenticatorFake{userID: "user_01H"}
	var gotUserID string
	nextCalled := false
	rejectCalled := false

	handler := Authenticate(fake, func(w http.ResponseWriter, r *http.Request, err error) {
		rejectCalled = true
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		gotUserID = requestcontext.UserID(r.Context())
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

			handler := Authenticate(fake, func(w http.ResponseWriter, r *http.Request, err error) {
				gotErr = err
				w.WriteHeader(http.StatusUnauthorized)
			})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

func TestAuthenticateDoesNotSetUserIDOnError(t *testing.T) {
	fake := &authenticatorFake{userID: "user_01H", err: errors.New("boom")}
	var gotUserID string

	handler := Authenticate(fake, func(w http.ResponseWriter, r *http.Request, err error) {
		gotUserID = requestcontext.UserID(r.Context())
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

	if gotUserID != "" {
		t.Fatalf("UserID = %q, want empty", gotUserID)
	}
}

func TestAuthenticatePassesResponseWriterToAuthenticator(t *testing.T) {
	handler := Authenticate(authenticatorWriterFake{}, func(w http.ResponseWriter, r *http.Request, err error) {})(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	fake := &authenticatorFake{userID: "user_01H"}
	var gotCorrelation string

	handler := Authenticate(fake, func(w http.ResponseWriter, r *http.Request, err error) {})(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
