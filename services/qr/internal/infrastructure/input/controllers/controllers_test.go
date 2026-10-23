package controllers_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"rutadeorigen/lib/requestcontext"
	"rutadeorigen/qr/internal/application/input/dto/request"
	"rutadeorigen/qr/internal/application/input/dto/response"
	"rutadeorigen/qr/internal/infrastructure/input/controllers"
	"rutadeorigen/qr/internal/platform/exception"
)

type resolver struct{}

func (resolver) Resolve(code string, _ map[string]any) (string, error) {
	return "message for " + code, nil
}

func newErrs() *controllers.ErrorResponder {
	return controllers.NewErrorResponder(slog.New(slog.NewTextHandler(io.Discard, nil)), resolver{})
}

func discardLog() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

type getLotFake struct {
	got request.GetPublicLot
	res response.PublicLot
	err error
}

func (f *getLotFake) Execute(_ context.Context, in request.GetPublicLot) (response.PublicLot, error) {
	f.got = in
	return f.res, f.err
}

type scanFake struct {
	got request.ResolveScan
	res response.ScanRedirect
	err error
}

func (f *scanFake) Execute(_ context.Context, in request.ResolveScan) (response.ScanRedirect, error) {
	f.got = in
	return f.res, f.err
}

type imageFake struct {
	got   request.GetQRImage
	res   response.QRImage
	err   error
	actor uuid.UUID
}

func (f *imageFake) Execute(ctx context.Context, in request.GetQRImage) (response.QRImage, error) {
	f.got = in
	f.actor = requestcontext.ActorID(ctx)
	return f.res, f.err
}

func serve(mux *http.ServeMux, req *http.Request) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func TestLotControllerReturnsPublicLot(t *testing.T) {
	fake := &getLotFake{res: response.PublicLot{ID: "abc", Code: "LT-001", QRAvailable: true}}
	mux := http.NewServeMux()
	controllers.NewLotController(fake, newErrs(), discardLog()).Register(mux)

	rec := serve(mux, httptest.NewRequest(http.MethodGet, "/public/lots/abc", nil))

	if rec.Code != http.StatusOK || fake.got.ID != "abc" {
		t.Fatalf("unexpected response: %d %q", rec.Code, fake.got.ID)
	}
	if rec.Header().Get("Cache-Control") != "public, max-age=60" {
		t.Errorf("unexpected cache control %q", rec.Header().Get("Cache-Control"))
	}
	var body response.PublicLot
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil || body.Code != "LT-001" {
		t.Errorf("unexpected body %+v %v", body, err)
	}
}

func TestLotControllerMapsNotFound(t *testing.T) {
	fake := &getLotFake{err: exception.New(exception.CodeLotViewNotFound)}
	mux := http.NewServeMux()
	controllers.NewLotController(fake, newErrs(), discardLog()).Register(mux)

	rec := serve(mux, httptest.NewRequest(http.MethodGet, "/public/lots/abc", nil))

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestScanControllerRedirects(t *testing.T) {
	fake := &scanFake{res: response.ScanRedirect{Location: "/lotes/abc", StatusCode: http.StatusMovedPermanently}}
	mux := http.NewServeMux()
	controllers.NewScanController(fake, newErrs()).Register(mux)

	rec := serve(mux, httptest.NewRequest(http.MethodGet, "/l/abc", nil))

	if rec.Code != http.StatusMovedPermanently || rec.Header().Get("Location") != "/lotes/abc" {
		t.Fatalf("unexpected redirect: %d %q", rec.Code, rec.Header().Get("Location"))
	}
	if fake.got.Path != "/l/abc" {
		t.Errorf("unexpected path %q", fake.got.Path)
	}
	if rec.Header().Get("Cache-Control") != "no-store" {
		t.Errorf("expected no-store")
	}
}

func TestScanControllerMapsErrors(t *testing.T) {
	fake := &scanFake{err: exception.New(exception.CodeLotViewNotFound)}
	mux := http.NewServeMux()
	controllers.NewScanController(fake, newErrs()).Register(mux)

	rec := serve(mux, httptest.NewRequest(http.MethodGet, "/l/abc", nil))

	if rec.Code != http.StatusNotFound || rec.Header().Get("Location") != "" {
		t.Fatalf("unexpected response: %d", rec.Code)
	}
}

func imageMux(fake *imageFake) *http.ServeMux {
	errs := newErrs()
	mux := http.NewServeMux()
	controllers.NewQRImageController(fake, errs).Register(mux, func(next http.Handler) http.Handler {
		return controllers.RequireActor(errs, next)
	})
	return mux
}

func TestQRImageControllerRequiresActor(t *testing.T) {
	mux := imageMux(&imageFake{})

	rec := serve(mux, httptest.NewRequest(http.MethodGet, "/public/lots/abc/qr", nil))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestQRImageControllerServesAttachment(t *testing.T) {
	fake := &imageFake{res: response.QRImage{Data: []byte{1, 2, 3}, ContentType: "image/png", Filename: "qr-LT-001.png"}}
	mux := imageMux(fake)
	actor := uuid.New()

	req := httptest.NewRequest(http.MethodGet, "/public/lots/abc/qr?size=256&format=png", nil)
	req.Header.Set(controllers.ActorIDHeader, actor.String())
	rec := serve(mux, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if fake.got != (request.GetQRImage{LotID: "abc", Size: "256", Format: "png"}) || fake.actor != actor {
		t.Errorf("unexpected input %+v actor %v", fake.got, fake.actor)
	}
	if rec.Header().Get("Content-Type") != "image/png" {
		t.Errorf("unexpected content type %q", rec.Header().Get("Content-Type"))
	}
	if rec.Header().Get("Content-Disposition") != `attachment; filename=qr-LT-001.png` {
		t.Errorf("unexpected disposition %q", rec.Header().Get("Content-Disposition"))
	}
	if rec.Header().Get("Cache-Control") != "private, max-age=300" || rec.Header().Get("Etag") == "" {
		t.Errorf("missing cache headers")
	}
	if rec.Body.Len() != 3 {
		t.Errorf("unexpected body length %d", rec.Body.Len())
	}
}

func TestQRImageControllerHonorsIfNoneMatch(t *testing.T) {
	fake := &imageFake{res: response.QRImage{Data: []byte{1, 2, 3}, ContentType: "image/png", Filename: "qr.png"}}
	mux := imageMux(fake)

	first := httptest.NewRequest(http.MethodGet, "/public/lots/abc/qr", nil)
	first.Header.Set(controllers.ActorIDHeader, uuid.NewString())
	etag := serve(mux, first).Header().Get("Etag")

	second := httptest.NewRequest(http.MethodGet, "/public/lots/abc/qr", nil)
	second.Header.Set(controllers.ActorIDHeader, uuid.NewString())
	second.Header.Set("If-None-Match", etag)
	rec := serve(mux, second)

	if rec.Code != http.StatusNotModified || rec.Body.Len() != 0 {
		t.Fatalf("expected empty 304, got %d with %d bytes", rec.Code, rec.Body.Len())
	}
}

func TestQRImageControllerMapsConflict(t *testing.T) {
	fake := &imageFake{err: exception.New(exception.CodeLotViewImageNotAvailable)}
	mux := imageMux(fake)

	req := httptest.NewRequest(http.MethodGet, "/public/lots/abc/qr", nil)
	req.Header.Set(controllers.ActorIDHeader, uuid.NewString())
	rec := serve(mux, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", rec.Code)
	}
}

func TestErrorResponderHidesInternalDetails(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/x", nil)

	newErrs().Write(rec, req, errors.New("database exploded"))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if body["code"] != string(exception.CodeInternal) || body["kind"] != "internal" {
		t.Errorf("unexpected body %v", body)
	}
}

func TestErrorResponderMapsValidation(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/x", nil)

	newErrs().Write(rec, req, exception.New(exception.CodeQRCodeInvalidSize))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}
