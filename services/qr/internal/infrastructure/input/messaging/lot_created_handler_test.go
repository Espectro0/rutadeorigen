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

type projectFake struct {
	calls         int
	got           request.ProjectLotCreated
	correlationID string
	err           error
}

func (f *projectFake) Execute(ctx context.Context, in request.ProjectLotCreated) (response.Empty, error) {
	f.calls++
	f.got = in
	f.correlationID = requestcontext.CorrelationID(ctx)
	return response.Empty{}, f.err
}

const body = `{"id":"env-id","type":"lot.created","version":1,"occurred_at":"2026-09-20T14:30:00Z","actor_id":"a","actor_role":"producer","correlation_id":"env-corr","data":{"lot_id":"lot-1","farm":{"id":"f1","name":"La Esperanza","altitude":1800},"producer":{"id":"p1","name":"Maria"},"custodian_id":"c1","code":"LT-001","variety":"Castillo","process":"washed","harvest_date":"2026-01-10","stage":"none","status":"active"}}`

func delivery() libmessaging.Delivery {
	return libmessaging.Delivery{
		ID:            "msg-1",
		Type:          "lot.created",
		CorrelationID: "corr-1",
		Version:       2,
		OccurredAt:    time.Date(2026, 9, 20, 14, 30, 0, 0, time.UTC),
		Body:          []byte(body),
	}
}

func TestHandleMapsDeliveryToRequest(t *testing.T) {
	fake := &projectFake{}

	if err := messaging.NewLotCreatedHandler(fake).Handle(context.Background(), delivery()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if fake.got.MessageID != "msg-1" || fake.got.EventType != "lot.created" || fake.got.Version != 2 {
		t.Errorf("unexpected metadata %+v", fake.got)
	}
	if fake.got.OccurredAt != "2026-09-20T14:30:00Z" {
		t.Errorf("unexpected occurred at %q", fake.got.OccurredAt)
	}
	if fake.got.Data.LotID != "lot-1" || fake.got.Data.Farm.Altitude != 1800 || fake.got.Data.Producer.Name != "Maria" {
		t.Errorf("unexpected data %+v", fake.got.Data)
	}
	if fake.correlationID != "corr-1" {
		t.Errorf("unexpected correlation id %q", fake.correlationID)
	}
}

func TestHandleFallsBackToEnvelopeFields(t *testing.T) {
	fake := &projectFake{}
	d := delivery()
	d.ID, d.Type, d.CorrelationID, d.Version = "", "", "", 0

	if err := messaging.NewLotCreatedHandler(fake).Handle(context.Background(), d); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if fake.got.MessageID != "env-id" || fake.got.Version != 1 || fake.correlationID != "env-corr" {
		t.Errorf("unexpected fallback %+v corr %q", fake.got, fake.correlationID)
	}
}

func TestHandleIgnoresOtherEventTypes(t *testing.T) {
	fake := &projectFake{}
	d := delivery()
	d.Type = "lot.voided"

	if err := messaging.NewLotCreatedHandler(fake).Handle(context.Background(), d); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fake.calls != 0 {
		t.Errorf("expected no use case call, got %d", fake.calls)
	}
}

func TestHandleMalformedBodyIsPermanent(t *testing.T) {
	fake := &projectFake{}
	d := delivery()
	d.Body = []byte("{not json")

	err := messaging.NewLotCreatedHandler(fake).Handle(context.Background(), d)

	if !libmessaging.IsPermanent(err) || !exception.HasCode(err, exception.CodeEventInvalidPayload) {
		t.Fatalf("expected permanent invalid payload, got %v", err)
	}
	if fake.calls != 0 {
		t.Error("use case must not run")
	}
}

func TestHandleValidationErrorIsPermanent(t *testing.T) {
	fake := &projectFake{err: exception.New(exception.CodeLotViewInvalidID)}

	err := messaging.NewLotCreatedHandler(fake).Handle(context.Background(), delivery())

	if !libmessaging.IsPermanent(err) {
		t.Fatalf("expected permanent error, got %v", err)
	}
}

func TestHandleOtherErrorsAreRetried(t *testing.T) {
	boom := errors.New("connection refused")
	fake := &projectFake{err: exception.New(exception.CodeInternal).WithCause(boom)}

	err := messaging.NewLotCreatedHandler(fake).Handle(context.Background(), delivery())

	if err == nil || libmessaging.IsPermanent(err) {
		t.Fatalf("expected retryable error, got %v", err)
	}
}
