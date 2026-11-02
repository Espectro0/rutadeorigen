package messaging_test

import (
	"context"
	"errors"
	"testing"

	libmessaging "rutadeorigen/lib/messaging"
	"rutadeorigen/lib/requestcontext"
	"rutadeorigen/processor/internal/application/input/dto/request"
	"rutadeorigen/processor/internal/infrastructure/input/messaging"
	"rutadeorigen/processor/internal/platform/exception"
)

type processFake struct {
	calls         int
	got           request.ProcessEvidence
	correlationID string
	err           error
}

func (f *processFake) Execute(ctx context.Context, in request.ProcessEvidence) error {
	f.calls++
	f.got = in
	f.correlationID = requestcontext.CorrelationID(ctx)
	return f.err
}

const uploadedBody = `{"id":"env-id","type":"evidence.uploaded","version":1,"occurred_at":"2026-11-02T06:00:00Z","actor_id":"a","actor_role":"producer","correlation_id":"env-corr","data":{"evidence_id":"ev-1","lot_id":"lot-1","lot_code":"L1","stage":"farm","file_name":"foto.jpg","content_type":"image/jpeg","size_bytes":2048,"quarantine_key":"lot-1/ev-1","uploader":{"actor_id":"actor-1","role":"producer","name":"x","email":"x@y.z"}}}`

func uploadedDelivery() libmessaging.Delivery {
	return libmessaging.Delivery{
		ID:            "msg-1",
		Type:          "evidence.uploaded",
		CorrelationID: "corr-1",
		Body:          []byte(uploadedBody),
	}
}

func TestHandleMapsDeliveryToRequest(t *testing.T) {
	fake := &processFake{}

	if err := messaging.NewEvidenceUploadedHandler(fake).Handle(context.Background(), uploadedDelivery()); err != nil {
		t.Fatal(err)
	}

	want := request.ProcessEvidence{
		MessageID:       "msg-1",
		EventType:       "evidence.uploaded",
		EvidenceID:      "ev-1",
		LotID:           "lot-1",
		Stage:           "farm",
		QuarantineKey:   "lot-1/ev-1",
		SizeBytes:       2048,
		UploaderActorID: "actor-1",
		UploaderRole:    "producer",
	}
	if fake.got != want {
		t.Errorf("got %+v, want %+v", fake.got, want)
	}
	if fake.correlationID != "corr-1" {
		t.Errorf("correlation id = %q", fake.correlationID)
	}
}

func TestHandleFallsBackToEnvelopeFields(t *testing.T) {
	fake := &processFake{}
	delivery := uploadedDelivery()
	delivery.ID, delivery.CorrelationID = "", ""

	if err := messaging.NewEvidenceUploadedHandler(fake).Handle(context.Background(), delivery); err != nil {
		t.Fatal(err)
	}

	if fake.got.MessageID != "env-id" || fake.correlationID != "env-corr" {
		t.Errorf("id=%q correlation=%q", fake.got.MessageID, fake.correlationID)
	}
}

func TestHandleIgnoresOtherEventTypes(t *testing.T) {
	fake := &processFake{}
	delivery := uploadedDelivery()
	delivery.Type = "lot.created"

	if err := messaging.NewEvidenceUploadedHandler(fake).Handle(context.Background(), delivery); err != nil {
		t.Fatal(err)
	}
	if fake.calls != 0 {
		t.Error("use case must not be called")
	}
}

func TestHandleMarksBrokenJSONPermanent(t *testing.T) {
	fake := &processFake{}
	delivery := uploadedDelivery()
	delivery.Body = []byte("{not json")

	err := messaging.NewEvidenceUploadedHandler(fake).Handle(context.Background(), delivery)

	if !libmessaging.IsPermanent(err) || !exception.HasCode(err, exception.CodeEventInvalidPayload) {
		t.Errorf("want permanent invalid payload, got %v", err)
	}
	if fake.calls != 0 {
		t.Error("use case must not be called")
	}
}

func TestHandleClassifiesUseCaseErrors(t *testing.T) {
	cases := map[string]struct {
		err       error
		permanent bool
	}{
		"invalid payload": {exception.New(exception.CodeEventInvalidPayload), true},
		"key mismatch":    {exception.New(exception.CodeEvidenceKeyMismatch), true},
		"storage down":    {exception.New(exception.CodeStorageUnavailable), false},
		"image failure":   {exception.New(exception.CodeImageProcessingFailed), false},
		"unknown":         {errors.New("db down"), false},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			fake := &processFake{err: tc.err}

			err := messaging.NewEvidenceUploadedHandler(fake).Handle(context.Background(), uploadedDelivery())

			if err == nil {
				t.Fatal("want error")
			}
			if libmessaging.IsPermanent(err) != tc.permanent {
				t.Errorf("permanent = %v, want %v", libmessaging.IsPermanent(err), tc.permanent)
			}
		})
	}
}
