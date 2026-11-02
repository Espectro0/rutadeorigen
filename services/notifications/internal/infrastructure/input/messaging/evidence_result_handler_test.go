package messaging_test

import (
	"context"
	"errors"
	"testing"

	libmessaging "rutadeorigen/lib/messaging"
	"rutadeorigen/lib/requestcontext"
	"rutadeorigen/notifications/internal/application/input/dto/request"
	"rutadeorigen/notifications/internal/infrastructure/input/messaging"
	"rutadeorigen/notifications/internal/platform/exception"
)

type sendFake struct {
	calls         int
	got           request.SendEvidenceNotification
	correlationID string
	err           error
}

func (f *sendFake) Execute(ctx context.Context, in request.SendEvidenceNotification) error {
	f.calls++
	f.got = in
	f.correlationID = requestcontext.CorrelationID(ctx)
	return f.err
}

const processedBody = `{"id":"env-id","type":"evidence.processed","version":1,"correlation_id":"env-corr","data":{"evidence_id":"ev-1","lot_id":"lot-1","lot_code":"L1","stage":"farm","file_name":"foto.jpg","sha256":"abc","final_key":"lot-1/ev-1/imagen.jpg","thumbnail_key":"lot-1/ev-1/miniatura.webp","content_type":"image/jpeg","uploader":{"actor_id":"actor-1","role":"producer","name":"Maria","email":"maria@example.com"}}}`

const rejectedBody = `{"id":"env-id","type":"evidence.rejected","version":1,"data":{"evidence_id":"ev-2","lot_id":"lot-1","lot_code":"L1","stage":"roasting","file_name":"raro.pdf","reason":"El archivo no es una imagen","uploader":{"actor_id":"actor-1","role":"roaster","name":"Pedro","email":"pedro@example.com"}}}`

func delivery(eventType, body string) libmessaging.Delivery {
	return libmessaging.Delivery{ID: "msg-1", Type: eventType, CorrelationID: "corr-1", Body: []byte(body)}
}

func TestHandleMapsProcessedEventToRequest(t *testing.T) {
	fake := &sendFake{}

	if err := messaging.NewEvidenceResultHandler(fake).Handle(context.Background(), delivery("evidence.processed", processedBody)); err != nil {
		t.Fatal(err)
	}

	want := request.SendEvidenceNotification{
		MessageID:     "msg-1",
		EventType:     "evidence.processed",
		EvidenceID:    "ev-1",
		LotID:         "lot-1",
		LotCode:       "L1",
		Stage:         "farm",
		FileName:      "foto.jpg",
		UploaderName:  "Maria",
		UploaderEmail: "maria@example.com",
	}
	if fake.got != want {
		t.Errorf("got %+v, want %+v", fake.got, want)
	}
	if fake.correlationID != "corr-1" {
		t.Errorf("correlation id = %q", fake.correlationID)
	}
}

func TestHandleMapsRejectedEventToRequest(t *testing.T) {
	fake := &sendFake{}

	if err := messaging.NewEvidenceResultHandler(fake).Handle(context.Background(), delivery("evidence.rejected", rejectedBody)); err != nil {
		t.Fatal(err)
	}

	want := request.SendEvidenceNotification{
		MessageID:     "msg-1",
		EventType:     "evidence.rejected",
		EvidenceID:    "ev-2",
		LotID:         "lot-1",
		LotCode:       "L1",
		Stage:         "roasting",
		FileName:      "raro.pdf",
		Reason:        "El archivo no es una imagen",
		UploaderName:  "Pedro",
		UploaderEmail: "pedro@example.com",
	}
	if fake.got != want {
		t.Errorf("got %+v, want %+v", fake.got, want)
	}
}

func TestHandleFallsBackToEnvelopeFields(t *testing.T) {
	fake := &sendFake{}
	d := delivery("evidence.processed", processedBody)
	d.ID, d.CorrelationID = "", ""

	if err := messaging.NewEvidenceResultHandler(fake).Handle(context.Background(), d); err != nil {
		t.Fatal(err)
	}

	if fake.got.MessageID != "env-id" || fake.correlationID != "env-corr" {
		t.Errorf("id=%q correlation=%q", fake.got.MessageID, fake.correlationID)
	}
}

func TestHandleIgnoresOtherEventTypes(t *testing.T) {
	fake := &sendFake{}

	if err := messaging.NewEvidenceResultHandler(fake).Handle(context.Background(), delivery("lot.created", processedBody)); err != nil {
		t.Fatal(err)
	}
	if fake.calls != 0 {
		t.Error("the use case must not be called")
	}
}

func TestHandleMarksBrokenJSONPermanent(t *testing.T) {
	fake := &sendFake{}

	err := messaging.NewEvidenceResultHandler(fake).Handle(context.Background(), delivery("evidence.processed", "{not json"))

	if !libmessaging.IsPermanent(err) || !exception.HasCode(err, exception.CodeEventInvalidPayload) {
		t.Errorf("want a permanent invalid payload error, got %v", err)
	}
	if fake.calls != 0 {
		t.Error("the use case must not be called")
	}
}

func TestHandleClassifiesUseCaseErrors(t *testing.T) {
	cases := map[string]struct {
		err       error
		permanent bool
	}{
		"invalid payload":   {exception.New(exception.CodeEventInvalidPayload), true},
		"invalid recipient": {exception.New(exception.CodeInvalidRecipient), true},
		"email rejected":    {exception.New(exception.CodeEmailRejected), true},
		"email unavailable": {exception.New(exception.CodeEmailUnavailable), false},
		"render failed":     {exception.New(exception.CodeEmailRenderFailed), false},
		"unknown":           {errors.New("db down"), false},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			fake := &sendFake{err: tc.err}

			err := messaging.NewEvidenceResultHandler(fake).Handle(context.Background(), delivery("evidence.processed", processedBody))

			if err == nil {
				t.Fatal("want an error")
			}
			if libmessaging.IsPermanent(err) != tc.permanent {
				t.Errorf("permanent = %v, want %v", libmessaging.IsPermanent(err), tc.permanent)
			}
		})
	}
}

func TestHandleSucceedsWhenTheUseCaseSucceeds(t *testing.T) {
	fake := &sendFake{}

	err := messaging.NewEvidenceResultHandler(fake).Handle(context.Background(), delivery("evidence.processed", processedBody))

	if err != nil || fake.calls != 1 {
		t.Errorf("got error %v after %d calls", err, fake.calls)
	}
}
