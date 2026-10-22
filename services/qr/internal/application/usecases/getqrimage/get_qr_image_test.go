package getqrimage_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"rutadeorigen/lib/requestcontext"
	"rutadeorigen/qr/internal/application/output/ports"
	"rutadeorigen/qr/internal/application/usecases/getqrimage"
	"rutadeorigen/qr/internal/domain/lotview"
	"rutadeorigen/qr/internal/domain/qrcode"
	"rutadeorigen/qr/internal/platform/exception"
)

const (
	baseURL        = "https://rutadeorigen.co"
	libraryVersion = "skip2/go-qrcode@test"
)

var errBackend = errors.New("backend is down")

type fakeViews struct {
	view *lotview.LotView
	err  error
}

func (v fakeViews) FindByID(_ context.Context, _ uuid.UUID) (*lotview.LotView, error) {
	if v.err != nil {
		return lotview.Default(), v.err
	}
	if v.view == nil {
		return lotview.Default(), nil
	}
	return v.view, nil
}

func (v fakeViews) Save(_ context.Context, _ *lotview.LotView) error { return nil }

type fakeCache struct{}

func (fakeCache) Get(_ context.Context, _ uuid.UUID) (*lotview.LotView, error) {
	return lotview.Default(), nil
}

func (fakeCache) Set(_ context.Context, _ *lotview.LotView) error { return nil }

func (fakeCache) Delete(_ context.Context, _ uuid.UUID) error { return nil }

type fakeCodes struct {
	existing *qrcode.QRCode
	findErr  error
	saveErr  error
	saved    []*qrcode.QRCode
}

func (c *fakeCodes) FindByLotID(_ context.Context, _ uuid.UUID) (*qrcode.QRCode, error) {
	if c.findErr != nil {
		return qrcode.Default(), c.findErr
	}
	if c.existing == nil {
		return qrcode.Default(), nil
	}
	return c.existing, nil
}

func (c *fakeCodes) Save(_ context.Context, code *qrcode.QRCode) error {
	if c.saveErr != nil {
		return c.saveErr
	}
	c.saved = append(c.saved, code)
	return nil
}

type fakeRenderer struct {
	data    []byte
	err     error
	calls   int
	request ports.RenderRequest
}

func (r *fakeRenderer) Render(_ context.Context, request ports.RenderRequest) ([]byte, error) {
	r.calls++
	r.request = request
	return r.data, r.err
}

func (r *fakeRenderer) LibraryVersion() string { return libraryVersion }

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

type fixture struct {
	codes    *fakeCodes
	renderer *fakeRenderer
	auth     *fakeAuthorizer
	view     *lotview.LotView
	actorID  uuid.UUID
	ctx      context.Context
	uc       *getqrimage.GetQRImage
}

func newView(t *testing.T, stage lotview.Stage, status lotview.Status) *lotview.LotView {
	t.Helper()

	v, err := lotview.New(lotview.NewParams{
		LotID:        uuid.New(),
		Code:         "LT-001",
		Variety:      "Castillo",
		Process:      lotview.ProcessWashed,
		HarvestDate:  time.Now().AddDate(0, -1, 0),
		FarmID:       uuid.New(),
		FarmName:     "La Esperanza",
		FarmAltitude: 1800,
		ProducerID:   uuid.New(),
		ProducerName: "Maria Lopez",
		CustodianID:  uuid.New(),
		CurrentStage: stage,
		Status:       status,
		Version:      1,
	})
	if err != nil {
		t.Fatalf("cannot build lot view: %v", err)
	}
	return v
}

func newFixture(t *testing.T, view *lotview.LotView) *fixture {
	t.Helper()

	f := &fixture{
		codes:    &fakeCodes{},
		renderer: &fakeRenderer{data: []byte{0x89, 0x50, 0x4e, 0x47}},
		auth:     &fakeAuthorizer{allowed: true},
		view:     view,
		actorID:  uuid.New(),
	}
	f.ctx = requestcontext.WithActorID(context.Background(), f.actorID)
	f.uc = getqrimage.New(fakeViews{view: view}, fakeCache{}, f.codes, f.renderer, f.auth, baseURL)
	return f
}

func (f *fixture) query() getqrimage.Query {
	return getqrimage.Query{LotID: f.view.LotID(), Size: qrcode.SizeMedium, Format: qrcode.FormatPNG}
}

func assertCode(t *testing.T, err error, want exception.Code) {
	t.Helper()

	ex, ok := exception.From(err)
	if !ok {
		t.Fatalf("expected an *Exception, got %v", err)
	}
	if ex.Code() != want {
		t.Errorf("Code: got %v, want %v", ex.Code(), want)
	}
}

func TestGetQRImageExecute(t *testing.T) {
	t.Run("creates the QR code the first time and renders the image", func(t *testing.T) {
		f := newFixture(t, newView(t, lotview.StagePackaging, lotview.StatusActive))

		got, err := f.uc.Execute(f.ctx, f.query())

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.IsDefault() || got.ContentType() != "image/png" || got.Filename() != "qr-LT-001.png" {
			t.Errorf("unexpected image: %q %q", got.ContentType(), got.Filename())
		}
		if string(got.Data()) != string(f.renderer.data) {
			t.Error("the image must carry the rendered bytes")
		}
		if len(f.codes.saved) != 1 {
			t.Fatalf("expected the QR code to be saved once, got %d", len(f.codes.saved))
		}
		saved := f.codes.saved[0]
		wantContent := baseURL + "/l/" + f.view.LotID().String()
		if saved.Content() != wantContent || saved.ErrorCorrection() != qrcode.DefaultErrorCorrection || saved.LibraryVersion() != libraryVersion {
			t.Errorf("unexpected saved QR code: %q %q %q", saved.Content(), saved.ErrorCorrection(), saved.LibraryVersion())
		}
		wantRequest := ports.RenderRequest{
			Content:         wantContent,
			ErrorCorrection: qrcode.DefaultErrorCorrection,
			Size:            qrcode.SizeMedium,
			Format:          qrcode.FormatPNG,
		}
		if f.renderer.request != wantRequest {
			t.Errorf("render request: got %+v, want %+v", f.renderer.request, wantRequest)
		}
	})

	t.Run("reuses the stored QR code and its parameters", func(t *testing.T) {
		f := newFixture(t, newView(t, lotview.StagePackaging, lotview.StatusActive))
		f.codes.existing = qrcode.Restore(qrcode.RestoreParams{
			LotID:           f.view.LotID(),
			Content:         "https://old.example.com/l/" + f.view.LotID().String(),
			ErrorCorrection: qrcode.ErrorCorrectionHigh,
			LibraryVersion:  "old-lib",
			CreatedAt:       time.Now().Add(-24 * time.Hour),
		})
		query := f.query()
		query.Format = qrcode.FormatSVG
		query.Size = qrcode.SizeLarge

		got, err := f.uc.Execute(f.ctx, query)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(f.codes.saved) != 0 {
			t.Errorf("a stored QR code must not be saved again, got %d saves", len(f.codes.saved))
		}
		wantRequest := ports.RenderRequest{
			Content:         f.codes.existing.Content(),
			ErrorCorrection: qrcode.ErrorCorrectionHigh,
			Size:            qrcode.SizeLarge,
			Format:          qrcode.FormatSVG,
		}
		if f.renderer.request != wantRequest {
			t.Errorf("render request: got %+v, want %+v", f.renderer.request, wantRequest)
		}
		if got.ContentType() != "image/svg+xml" || got.Filename() != "qr-LT-001.svg" {
			t.Errorf("unexpected image: %q %q", got.ContentType(), got.Filename())
		}
	})

	t.Run("authorizes the actor against the lot custodian", func(t *testing.T) {
		f := newFixture(t, newView(t, lotview.StagePackaging, lotview.StatusActive))

		if _, err := f.uc.Execute(f.ctx, f.query()); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if f.auth.action != ports.ActionReadQRImage || f.auth.subject.ActorID != f.actorID {
			t.Errorf("unexpected action %q or subject %+v", f.auth.action, f.auth.subject)
		}
		wantResource := ports.Resource{Type: ports.ResourceLot, ID: f.view.LotID(), CustodianID: f.view.CustodianID()}
		if f.auth.resource != wantResource {
			t.Errorf("resource: got %+v, want %+v", f.auth.resource, wantResource)
		}
	})

	t.Run("rejects a request without an authenticated actor", func(t *testing.T) {
		f := newFixture(t, newView(t, lotview.StagePackaging, lotview.StatusActive))

		got, err := f.uc.Execute(context.Background(), f.query())

		assertCode(t, err, exception.CodeUnauthorized)
		if !got.IsDefault() || f.renderer.calls != 0 {
			t.Errorf("got %+v and %d renders, want default and 0", got, f.renderer.calls)
		}
	})

	t.Run("rejects an unsupported size or format", func(t *testing.T) {
		tests := []struct {
			name     string
			mutate   func(q *getqrimage.Query)
			wantCode exception.Code
		}{
			{"unknown size", func(q *getqrimage.Query) { q.Size = qrcode.SizeUnknown }, exception.CodeQRCodeInvalidSize},
			{"unsupported size", func(q *getqrimage.Query) { q.Size = qrcode.Size(300) }, exception.CodeQRCodeInvalidSize},
			{"unknown format", func(q *getqrimage.Query) { q.Format = qrcode.FormatUnknown }, exception.CodeQRCodeInvalidFormat},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				f := newFixture(t, newView(t, lotview.StagePackaging, lotview.StatusActive))
				query := f.query()
				tt.mutate(&query)

				got, err := f.uc.Execute(f.ctx, query)

				assertCode(t, err, tt.wantCode)
				if !got.IsDefault() || f.renderer.calls != 0 {
					t.Errorf("got %+v and %d renders, want default and 0", got, f.renderer.calls)
				}
			})
		}
	})

	t.Run("reports an unknown lot as not found", func(t *testing.T) {
		f := newFixture(t, nil)
		f.view = newView(t, lotview.StagePackaging, lotview.StatusActive)

		got, err := f.uc.Execute(f.ctx, f.query())

		assertCode(t, err, exception.CodeLotViewNotFound)
		if !got.IsDefault() || f.auth.calls != 0 {
			t.Errorf("got %+v and %d authorizer calls, want default and 0", got, f.auth.calls)
		}
	})

	t.Run("forbids a denied actor before rendering", func(t *testing.T) {
		f := newFixture(t, newView(t, lotview.StagePackaging, lotview.StatusActive))
		f.auth.allowed = false

		got, err := f.uc.Execute(f.ctx, f.query())

		assertCode(t, err, exception.CodeForbidden)
		if !got.IsDefault() || f.renderer.calls != 0 || len(f.codes.saved) != 0 {
			t.Errorf("got %+v, %d renders and %d saves, want default, 0 and 0", got, f.renderer.calls, len(f.codes.saved))
		}
	})

	t.Run("wraps an authorizer failure", func(t *testing.T) {
		f := newFixture(t, newView(t, lotview.StagePackaging, lotview.StatusActive))
		f.auth.err = errBackend

		_, err := f.uc.Execute(f.ctx, f.query())

		assertCode(t, err, exception.CodeInternal)
		if !errors.Is(err, errBackend) {
			t.Errorf("the cause must be preserved, got %v", err)
		}
	})

	t.Run("refuses the image until the lot is packaged", func(t *testing.T) {
		for _, stage := range []lotview.Stage{lotview.StageNone, lotview.StageFarm, lotview.StageRoasting} {
			t.Run(stage.String(), func(t *testing.T) {
				f := newFixture(t, newView(t, stage, lotview.StatusActive))

				got, err := f.uc.Execute(f.ctx, f.query())

				assertCode(t, err, exception.CodeLotViewImageNotAvailable)
				if ex, _ := exception.From(err); ex.Kind() != exception.KindConflict {
					t.Errorf("Kind: got %v, want %v", ex.Kind(), exception.KindConflict)
				}
				if !got.IsDefault() || f.renderer.calls != 0 || len(f.codes.saved) != 0 {
					t.Errorf("got %+v, %d renders and %d saves, want default, 0 and 0", got, f.renderer.calls, len(f.codes.saved))
				}
			})
		}
	})

	t.Run("refuses the image of a voided lot", func(t *testing.T) {
		f := newFixture(t, newView(t, lotview.StagePackaging, lotview.StatusVoided))

		_, err := f.uc.Execute(f.ctx, f.query())

		assertCode(t, err, exception.CodeLotViewImageNotAvailable)
	})

	t.Run("wraps QR code storage failures", func(t *testing.T) {
		tests := []struct {
			name  string
			setup func(f *fixture)
		}{
			{"find failure", func(f *fixture) { f.codes.findErr = errBackend }},
			{"save failure", func(f *fixture) { f.codes.saveErr = errBackend }},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				f := newFixture(t, newView(t, lotview.StagePackaging, lotview.StatusActive))
				tt.setup(f)

				got, err := f.uc.Execute(f.ctx, f.query())

				assertCode(t, err, exception.CodeInternal)
				if !errors.Is(err, errBackend) {
					t.Errorf("the cause must be preserved, got %v", err)
				}
				if !got.IsDefault() || f.renderer.calls != 0 {
					t.Errorf("got %+v and %d renders, want default and 0", got, f.renderer.calls)
				}
			})
		}
	})

	t.Run("wraps a render failure", func(t *testing.T) {
		f := newFixture(t, newView(t, lotview.StagePackaging, lotview.StatusActive))
		f.renderer.err = errBackend

		got, err := f.uc.Execute(f.ctx, f.query())

		assertCode(t, err, exception.CodeInternal)
		if !errors.Is(err, errBackend) {
			t.Errorf("the cause must be preserved, got %v", err)
		}
		if !got.IsDefault() {
			t.Errorf("got %+v, want default", got)
		}
	})

	t.Run("reports an empty render as an internal error", func(t *testing.T) {
		f := newFixture(t, newView(t, lotview.StagePackaging, lotview.StatusActive))
		f.renderer.data = nil

		got, err := f.uc.Execute(f.ctx, f.query())

		assertCode(t, err, exception.CodeInternal)
		if !got.IsDefault() {
			t.Errorf("got %+v, want default", got)
		}
	})

	t.Run("reports a misconfigured public URL as an internal error", func(t *testing.T) {
		f := newFixture(t, newView(t, lotview.StagePackaging, lotview.StatusActive))
		f.uc = getqrimage.New(fakeViews{view: f.view}, fakeCache{}, f.codes, f.renderer, f.auth, "not-a-url")

		_, err := f.uc.Execute(f.ctx, f.query())

		assertCode(t, err, exception.CodeInternal)
		if len(f.codes.saved) != 0 || f.renderer.calls != 0 {
			t.Errorf("expected no saves or renders, got %d and %d", len(f.codes.saved), f.renderer.calls)
		}
	})
}
