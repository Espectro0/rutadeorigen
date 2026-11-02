package main

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log/slog"
	"strings"
	"testing"

	"github.com/google/uuid"

	libmessaging "rutadeorigen/lib/messaging"
	"rutadeorigen/processor/internal/domain/picture"
	inputmessaging "rutadeorigen/processor/internal/infrastructure/input/messaging"
	"rutadeorigen/processor/internal/infrastructure/output/external/rejectionmessages"
	"rutadeorigen/processor/internal/infrastructure/output/external/storage"
	"rutadeorigen/processor/internal/platform/catalog"
)

type processorFixture struct {
	handler    *inputmessaging.EvidenceUploadedHandler
	logs       *bytes.Buffer
	quarantine *storage.MemoryQuarantine
	evidences  *storage.MemoryEvidences
}

func newProcessorFixture(t *testing.T) processorFixture {
	t.Helper()

	processorCatalog, err := catalog.New()
	if err != nil {
		t.Fatal(err)
	}
	limits, err := picture.NewLimits(defaultMaxFileBytes, defaultMaxMegapixels, defaultThumbnailSide)
	if err != nil {
		t.Fatal(err)
	}

	var logs bytes.Buffer
	quarantine, evidences := storage.NewMemoryQuarantine(), storage.NewMemoryEvidences()
	handler := newEvidenceUploadedHandler(slog.New(slog.NewJSONHandler(&logs, nil)), nil,
		buckets{quarantine: quarantine, evidences: evidences},
		rejectionmessages.NewCatalog(processorCatalog), limits)

	return processorFixture{handler: handler, logs: &logs, quarantine: quarantine, evidences: evidences}
}

func pngBytes(t *testing.T) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, 50, 40))
	for y := 0; y < 40; y++ {
		for x := 0; x < 50; x++ {
			img.SetNRGBA(x, y, color.NRGBA{R: uint8(x * 5), G: uint8(y * 6), B: 90, A: 255})
		}
	}
	var buffer bytes.Buffer
	if err := png.Encode(&buffer, img); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

type uploadedEvent struct {
	delivery libmessaging.Delivery
	lotID    uuid.UUID
	key      string
}

func newUploadedEvent(size int, quarantineKey string) uploadedEvent {
	lotID, evidenceID := uuid.New(), uuid.New()
	key := lotID.String() + "/" + evidenceID.String()
	if quarantineKey != "" {
		key = quarantineKey
	}

	body := fmt.Sprintf(`{"id":"env","type":"evidence.uploaded","version":1,"correlation_id":"corr","data":{"evidence_id":%q,"lot_id":%q,"size_bytes":%d,"quarantine_key":%q,"uploader":{"actor_id":%q,"role":"producer"}}}`,
		evidenceID, lotID, size, key, uuid.New())

	return uploadedEvent{
		delivery: libmessaging.Delivery{ID: uuid.NewString(), Type: "evidence.uploaded", Body: []byte(body)},
		lotID:    lotID,
		key:      lotID.String() + "/" + evidenceID.String(),
	}
}

func TestProcessesUploadedEvidenceEndToEnd(t *testing.T) {
	f := newProcessorFixture(t)
	data := pngBytes(t)
	event := newUploadedEvent(len(data), "")
	f.quarantine.Seed(event.key, data)

	if err := f.handler.Handle(context.Background(), event.delivery); err != nil {
		t.Fatal(err)
	}

	if len(f.evidences.Keys()) != 2 {
		t.Errorf("want final image and thumbnail, got %v", f.evidences.Keys())
	}
	if !strings.Contains(f.logs.String(), `"event_type":"evidence.processed"`) {
		t.Errorf("want evidence.processed event, logs: %s", f.logs)
	}
	if object, _ := f.quarantine.Get(context.Background(), event.key, 1<<20); object.Exists {
		t.Error("original must leave quarantine")
	}
}

func TestRejectsFileThatIsNotAnImage(t *testing.T) {
	f := newProcessorFixture(t)
	notAnImage := []byte("this is not an image")
	event := newUploadedEvent(len(notAnImage), "")
	f.quarantine.Seed(event.key, notAnImage)

	if err := f.handler.Handle(context.Background(), event.delivery); err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(f.logs.String(), `"event_type":"evidence.rejected"`) {
		t.Errorf("want evidence.rejected event, logs: %s", f.logs)
	}
	if !strings.Contains(f.logs.String(), "no es una imagen JPEG, PNG o WebP") {
		t.Error("rejection must carry the Spanish reason")
	}
	if len(f.evidences.Keys()) != 0 {
		t.Error("nothing must be stored for a rejected file")
	}
}

func TestRedeliveredMessageIsProcessedOnce(t *testing.T) {
	f := newProcessorFixture(t)
	data := pngBytes(t)
	event := newUploadedEvent(len(data), "")
	f.quarantine.Seed(event.key, data)

	for attempt := 0; attempt < 2; attempt++ {
		if err := f.handler.Handle(context.Background(), event.delivery); err != nil {
			t.Fatal(err)
		}
	}

	if published := strings.Count(f.logs.String(), "event published"); published != 1 {
		t.Errorf("want one published event, got %d", published)
	}
}

func TestBrokenEventGoesToDeadLetter(t *testing.T) {
	f := newProcessorFixture(t)
	event := newUploadedEvent(10, uuid.NewString()+"/"+uuid.NewString())

	err := f.handler.Handle(context.Background(), event.delivery)

	if !libmessaging.IsPermanent(err) {
		t.Errorf("want permanent error, got %v", err)
	}
}

func TestLoadLimitsUsesDefaults(t *testing.T) {
	limits, err := loadLimits()

	if err != nil {
		t.Fatal(err)
	}
	if limits.MaxFileBytes() != defaultMaxFileBytes || limits.MaxMegapixels() != defaultMaxMegapixels ||
		limits.ThumbnailSide() != defaultThumbnailSide {
		t.Errorf("unexpected defaults: %+v", limits)
	}
}

func TestLoadLimitsRejectsInvalidValues(t *testing.T) {
	t.Setenv("IMAGE_MAX_MEGAPIXELS", "-3")
	t.Setenv("THUMBNAIL_SIDE", "abc")

	_, err := loadLimits()

	if err == nil || !strings.Contains(err.Error(), "IMAGE_MAX_MEGAPIXELS") || !strings.Contains(err.Error(), "THUMBNAIL_SIDE") {
		t.Errorf("want both invalid variables reported, got %v", err)
	}
}

func TestLoadGarageRequiresCredentials(t *testing.T) {
	t.Setenv("GARAGE_ENDPOINT", "")
	t.Setenv("GARAGE_ACCESS_KEY", "")
	t.Setenv("GARAGE_SECRET_KEY", "")

	if _, err := loadGarage(); err == nil {
		t.Error("want error when credentials are missing")
	}
}

func TestLoadGarageAppliesDefaults(t *testing.T) {
	t.Setenv("GARAGE_ENDPOINT", "garage.local:3900")
	t.Setenv("GARAGE_ACCESS_KEY", "test-key-id")
	t.Setenv("GARAGE_SECRET_KEY", "test-secret")

	garage, err := loadGarage()

	if err != nil {
		t.Fatal(err)
	}
	if garage.QuarantineBucket != "quarantine" || garage.EvidenceBucket != "evidences" || garage.Region != "garage" || garage.UseSSL {
		t.Errorf("unexpected defaults: %+v", garage)
	}
}
