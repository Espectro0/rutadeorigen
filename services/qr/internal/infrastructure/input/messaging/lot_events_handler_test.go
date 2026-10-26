package messaging_test

import (
	"context"
	"errors"
	"testing"
	"time"

	libmessaging "rutadeorigen/lib/messaging"
	"rutadeorigen/lib/requestcontext"
	"rutadeorigen/qr/internal/application/input/dto/request"
	"rutadeorigen/qr/internal/application/input/dto/response"
	"rutadeorigen/qr/internal/infrastructure/input/messaging"
	"rutadeorigen/qr/internal/platform/exception"
)

type createdFake struct {
	calls         int
	got           request.ProjectLotCreated
	correlationID string
	err           error
}

func (f *createdFake) Execute(ctx context.Context, in request.ProjectLotCreated) (response.Empty, error) {
	f.calls++
	f.got = in
	f.correlationID = requestcontext.CorrelationID(ctx)
	return response.Empty{}, f.err
}

type stageFake struct {
	calls         int
	got           request.ProjectLotStage
	correlationID string
	err           error
}

func (f *stageFake) Execute(ctx context.Context, in request.ProjectLotStage) (response.Empty, error) {
	f.calls++
	f.got = in
	f.correlationID = requestcontext.CorrelationID(ctx)
	return response.Empty{}, f.err
}

const createdBody = `{"id":"env-id","type":"lot.created","version":1,"occurred_at":"2026-09-20T14:30:00Z","actor_id":"a","actor_role":"producer","correlation_id":"env-corr","data":{"lot_id":"lot-1","farm":{"id":"f1","name":"La Esperanza","altitude":1800},"producer":{"id":"p1","name":"Maria"},"custodian_id":"c1","code":"LT-001","variety":"Castillo","process":"washed","harvest_date":"2026-01-10","stage":"none","status":"active"}}`

const stageBody = `{"id":"env-id","type":"lot.stage_registered","version":4,"occurred_at":"2026-09-21T10:00:00Z","actor_id":"a","actor_role":"producer","correlation_id":"env-corr","data":{"lot_id":"lot-1","stage":"roasting","status":"active","event_id":"ev-1","sequence":2,"event_date":"2026-09-20","extra":"ignored"}}`

func createdDelivery() libmessaging.Delivery {
	return libmessaging.Delivery{
		ID:            "msg-1",
		Type:          "lot.created",
		CorrelationID: "corr-1",
		Version:       2,
		OccurredAt:    time.Date(2026, 9, 20, 14, 30, 0, 0, time.UTC),
		Body:          []byte(createdBody),
	}
}

func stageDelivery() libmessaging.Delivery {
	return libmessaging.Delivery{
		ID:            "msg-2",
		Type:          "lot.stage_registered",
		CorrelationID: "corr-2",
		Version:       5,
		OccurredAt:    time.Date(2026, 9, 21, 10, 0, 0, 0, time.UTC),
		Body:          []byte(stageBody),
	}
}

func newHandler(created *createdFake, staged *stageFake) *messaging.LotEventsHandler {
	return messaging.NewLotEventsHandler(created, staged)
}

func TestHandleLotCreatedMapsDeliveryToRequest(t *testing.T) {
	created, staged := &createdFake{}, &stageFake{}

	if err := newHandler(created, staged).Handle(context.Background(), createdDelivery()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if created.got.MessageID != "msg-1" || created.got.EventType != "lot.created" || created.got.Version != 2 {
		t.Errorf("unexpected metadata %+v", created.got)
	}
	if created.got.OccurredAt != "2026-09-20T14:30:00Z" {
		t.Errorf("unexpected occurred at %q", created.got.OccurredAt)
	}
	if created.got.Data.LotID != "lot-1" || created.got.Data.Farm.Altitude != 1800 || created.got.Data.Producer.Name != "Maria" {
		t.Errorf("unexpected data %+v", created.got.Data)
	}
	if created.correlationID != "corr-1" {
		t.Errorf("unexpected correlation id %q", created.correlationID)
	}
	if staged.calls != 0 {
		t.Errorf("the stage use case must not run, got %d calls", staged.calls)
	}
}

func TestHandleLotStageRegisteredMapsDeliveryToRequest(t *testing.T) {
	created, staged := &createdFake{}, &stageFake{}

	if err := newHandler(created, staged).Handle(context.Background(), stageDelivery()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if staged.calls != 1 {
		t.Fatalf("stage use case calls: got %d, want 1", staged.calls)
	}
	if staged.got.MessageID != "msg-2" || staged.got.EventType != "lot.stage_registered" || staged.got.Version != 5 {
		t.Errorf("unexpected metadata %+v", staged.got)
	}
	if staged.got.OccurredAt != "2026-09-21T10:00:00Z" {
		t.Errorf("unexpected occurred at %q", staged.got.OccurredAt)
	}
	want := request.LotStageData{
		LotID:     "lot-1",
		Stage:     "roasting",
		Status:    "active",
		EventID:   "ev-1",
		Sequence:  2,
		EventDate: "2026-09-20",
	}
	if staged.got.Data != want {
		t.Errorf("unexpected data %+v, want %+v", staged.got.Data, want)
	}
	if staged.correlationID != "corr-2" {
		t.Errorf("unexpected correlation id %q", staged.correlationID)
	}
	if created.calls != 0 {
		t.Errorf("the created use case must not run, got %d calls", created.calls)
	}
}

func TestHandleFallsBackToEnvelopeFields(t *testing.T) {
	tests := []struct {
		name     string
		delivery libmessaging.Delivery
		check    func(t *testing.T, created *createdFake, staged *stageFake)
	}{
		{
			name: "lot created",
			delivery: libmessaging.Delivery{
				Body: []byte(createdBody),
			},
			check: func(t *testing.T, created *createdFake, _ *stageFake) {
				if created.got.MessageID != "env-id" || created.got.EventType != "lot.created" || created.got.Version != 1 ||
					created.got.OccurredAt != "2026-09-20T14:30:00Z" || created.correlationID != "env-corr" {
					t.Errorf("unexpected fallback %+v corr %q", created.got, created.correlationID)
				}
			},
		},
		{
			name: "lot stage registered",
			delivery: libmessaging.Delivery{
				Body: []byte(stageBody),
			},
			check: func(t *testing.T, _ *createdFake, staged *stageFake) {
				if staged.got.MessageID != "env-id" || staged.got.EventType != "lot.stage_registered" || staged.got.Version != 4 ||
					staged.got.OccurredAt != "2026-09-21T10:00:00Z" || staged.correlationID != "env-corr" {
					t.Errorf("unexpected fallback %+v corr %q", staged.got, staged.correlationID)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			created, staged := &createdFake{}, &stageFake{}

			if err := newHandler(created, staged).Handle(context.Background(), tt.delivery); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			tt.check(t, created, staged)
		})
	}
}

func TestHandleUsesTheOccurredAtOfTheDeliveryWhenTheEnvelopeHasNone(t *testing.T) {
	staged := &stageFake{}
	d := stageDelivery()
	d.Body = []byte(`{"id":"env-id","type":"lot.stage_registered","data":{"lot_id":"lot-1","stage":"farm"}}`)
	d.OccurredAt = time.Date(2026, 9, 21, 5, 0, 0, 0, time.FixedZone("UTC-5", -5*60*60))

	if err := newHandler(&createdFake{}, staged).Handle(context.Background(), d); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if staged.got.OccurredAt != "2026-09-21T10:00:00Z" {
		t.Errorf("unexpected occurred at %q", staged.got.OccurredAt)
	}
}

func TestHandleLeavesOccurredAtEmptyWhenNoSourceHasIt(t *testing.T) {
	staged := &stageFake{}
	d := stageDelivery()
	d.Body = []byte(`{"data":{"lot_id":"lot-1","stage":"farm"}}`)
	d.OccurredAt = time.Time{}

	if err := newHandler(&createdFake{}, staged).Handle(context.Background(), d); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if staged.got.OccurredAt != "" {
		t.Errorf("unexpected occurred at %q", staged.got.OccurredAt)
	}
}

func TestHandleIgnoresOtherEventTypes(t *testing.T) {
	created, staged := &createdFake{}, &stageFake{}
	d := createdDelivery()
	d.Type = "lot.voided"

	if err := newHandler(created, staged).Handle(context.Background(), d); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if created.calls != 0 || staged.calls != 0 {
		t.Errorf("expected no use case call, got %d and %d", created.calls, staged.calls)
	}
}

func TestHandleMalformedBodyIsPermanent(t *testing.T) {
	for _, d := range []libmessaging.Delivery{createdDelivery(), stageDelivery()} {
		t.Run(d.Type, func(t *testing.T) {
			created, staged := &createdFake{}, &stageFake{}
			d.Body = []byte("{not json")

			err := newHandler(created, staged).Handle(context.Background(), d)

			if !libmessaging.IsPermanent(err) || !exception.HasCode(err, exception.CodeEventInvalidPayload) {
				t.Fatalf("expected permanent invalid payload, got %v", err)
			}
			if created.calls != 0 || staged.calls != 0 {
				t.Error("use cases must not run")
			}
		})
	}
}

func TestHandleMalformedBodyOfAnIgnoredTypeIsStillPermanent(t *testing.T) {
	d := createdDelivery()
	d.Type = "lot.voided"
	d.Body = []byte("{not json")

	err := newHandler(&createdFake{}, &stageFake{}).Handle(context.Background(), d)

	if !libmessaging.IsPermanent(err) {
		t.Fatalf("expected permanent error, got %v", err)
	}
}

func TestHandleMalformedDataIsPermanent(t *testing.T) {
	tests := []struct {
		name string
		d    libmessaging.Delivery
		body string
	}{
		{"created with a wrong field type", createdDelivery(), `{"type":"lot.created","data":{"farm":"not-an-object"}}`},
		{"created without data", createdDelivery(), `{"type":"lot.created"}`},
		{"stage with a wrong field type", stageDelivery(), `{"type":"lot.stage_registered","data":{"sequence":"two"}}`},
		{"stage without data", stageDelivery(), `{"type":"lot.stage_registered"}`},
		{"stage with data that is not an object", stageDelivery(), `{"type":"lot.stage_registered","data":"lot-1"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			created, staged := &createdFake{}, &stageFake{}
			tt.d.Body = []byte(tt.body)

			err := newHandler(created, staged).Handle(context.Background(), tt.d)

			if !libmessaging.IsPermanent(err) || !exception.HasCode(err, exception.CodeEventInvalidPayload) {
				t.Fatalf("expected permanent invalid payload, got %v", err)
			}
			if created.calls != 0 || staged.calls != 0 {
				t.Error("use cases must not run")
			}
		})
	}
}

func TestHandleValidationErrorIsPermanent(t *testing.T) {
	tests := []struct {
		name     string
		delivery libmessaging.Delivery
		created  *createdFake
		staged   *stageFake
	}{
		{"lot created", createdDelivery(), &createdFake{err: exception.New(exception.CodeLotViewInvalidID)}, &stageFake{}},
		{"lot stage registered", stageDelivery(), &createdFake{}, &stageFake{err: exception.New(exception.CodeLotViewInvalidStage)}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := newHandler(tt.created, tt.staged).Handle(context.Background(), tt.delivery)

			if !libmessaging.IsPermanent(err) {
				t.Fatalf("expected permanent error, got %v", err)
			}
		})
	}
}

func TestHandleOtherErrorsAreRetried(t *testing.T) {
	boom := errors.New("connection refused")
	internal := exception.New(exception.CodeInternal).WithCause(boom)

	tests := []struct {
		name     string
		delivery libmessaging.Delivery
		created  *createdFake
		staged   *stageFake
	}{
		{"lot created", createdDelivery(), &createdFake{err: internal}, &stageFake{}},
		{"lot stage registered", stageDelivery(), &createdFake{}, &stageFake{err: internal}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := newHandler(tt.created, tt.staged).Handle(context.Background(), tt.delivery)

			if err == nil || libmessaging.IsPermanent(err) {
				t.Fatalf("expected retryable error, got %v", err)
			}
		})
	}
}

func TestHandleStageForALotThatIsNotProjectedYetIsRetried(t *testing.T) {
	staged := &stageFake{err: exception.New(exception.CodeLotViewNotFound)}

	err := newHandler(&createdFake{}, staged).Handle(context.Background(), stageDelivery())

	if err == nil || libmessaging.IsPermanent(err) {
		t.Fatalf("expected retryable error, got %v", err)
	}
	if !exception.HasCode(err, exception.CodeLotViewNotFound) {
		t.Errorf("the original code must be kept, got %v", err)
	}
}
