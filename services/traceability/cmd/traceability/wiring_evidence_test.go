package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	libmessaging "rutadeorigen/lib/messaging"
	"rutadeorigen/traceability/internal/application/output/ports"
	"rutadeorigen/traceability/internal/infrastructure/input/messaging"
	evidencestorage "rutadeorigen/traceability/internal/infrastructure/output/external/storage"
	"rutadeorigen/traceability/internal/platform/catalog"
	"rutadeorigen/traceability/internal/platform/exception"
)

type evidenceServer struct {
	mux     *http.ServeMux
	logs    *bytes.Buffer
	storage *evidencestorage.Memory
	results *messaging.EvidenceResultsHandler
}

func newEvidenceServer(t *testing.T, extra ...routeOption) evidenceServer {
	t.Helper()

	messages, err := catalog.New()
	if err != nil {
		t.Fatalf("catalog.New: %v", err)
	}

	var logs bytes.Buffer
	storage := evidencestorage.NewMemory()
	mux := http.NewServeMux()
	options := append([]routeOption{withEvidenceStorage(storage)}, extra...)
	results := registerRoutes(mux, slog.New(slog.NewJSONHandler(&logs, nil)), nil, messages,
		authn{mode: authModeDevHeader}, options...)

	return evidenceServer{mux: mux, logs: &logs, storage: storage, results: results}
}

func evidenceRequest(mux http.Handler, method, actorID, target, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if actorID != "" {
		req.Header.Set("X-Actor-ID", actorID)
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

const evidenceBody = `{"file_name":"secado.jpg","content_type":"image/jpeg","size_bytes":2048}`

type evidenceView struct {
	ID     string `json:"id"`
	LotID  string `json:"lot_id"`
	Stage  string `json:"stage"`
	Status string `json:"status"`
	SHA256 string `json:"sha256"`

	ThumbnailURL string `json:"thumbnail_url"`
	ImageURL     string `json:"image_url"`
	URLExpiresAt string `json:"url_expires_at"`
}

type evidenceUploadView struct {
	Evidence evidenceView `json:"evidence"`
	Upload   struct {
		URL     string            `json:"url"`
		Method  string            `json:"method"`
		Headers map[string]string `json:"headers"`
	} `json:"upload"`
}

func mustRequestUpload(t *testing.T, s evidenceServer, actorID, lotID string) evidenceUploadView {
	t.Helper()

	rec := evidenceRequest(s.mux, http.MethodPost, actorID, "/lots/"+lotID+"/evidences", evidenceBody)
	if rec.Code != http.StatusCreated {
		t.Fatalf("request upload: status %d (%s)", rec.Code, rec.Body.String())
	}

	var view evidenceUploadView
	if err := json.NewDecoder(rec.Body).Decode(&view); err != nil {
		t.Fatalf("body is not JSON: %v", err)
	}
	return view
}

func completeUpload(s evidenceServer, actorID, lotID, evidenceID string) *httptest.ResponseRecorder {
	return evidenceRequest(s.mux, http.MethodPost, actorID, "/lots/"+lotID+"/evidences/"+evidenceID+"/complete", "")
}

func quarantineKey(v evidenceUploadView) string { return v.Evidence.LotID + "/" + v.Evidence.ID }

func TestEvidenceUploadEndToEnd(t *testing.T) {
	s := newEvidenceServer(t)
	lot, _ := lotWithFarmStage(t, s.mux)

	requested := mustRequestUpload(t, s, producerHeader, lot.ID)

	if requested.Evidence.Status != "pending_upload" || requested.Evidence.Stage != "farm" {
		t.Errorf("got status %q and stage %q, want pending_upload and farm", requested.Evidence.Status, requested.Evidence.Stage)
	}
	if requested.Upload.Method != http.MethodPut || requested.Upload.URL == "" ||
		requested.Upload.Headers["Content-Type"] != "image/jpeg" {
		t.Errorf("unexpected upload target %+v", requested.Upload)
	}

	rec := completeUpload(s, producerHeader, lot.ID, requested.Evidence.ID)
	assertErrorCode(t, rec, http.StatusConflict, string(exception.CodeEvidenceFileMissing))

	s.storage.Put(quarantineKey(requested), 2048)
	rec = completeUpload(s, producerHeader, lot.ID, requested.Evidence.ID)
	if rec.Code != http.StatusOK {
		t.Fatalf("complete: status %d (%s)", rec.Code, rec.Body.String())
	}
	var completed evidenceView
	if err := json.NewDecoder(rec.Body).Decode(&completed); err != nil || completed.Status != "uploaded" {
		t.Fatalf("got %+v (%v), want an uploaded evidence", completed, err)
	}

	if rec := completeUpload(s, producerHeader, lot.ID, requested.Evidence.ID); rec.Code != http.StatusOK {
		t.Fatalf("completing twice must be harmless, status %d", rec.Code)
	}
	if n := strings.Count(s.logs.String(), `"event_type":"evidence.uploaded"`); n != 1 {
		t.Errorf("evidence.uploaded was published %d times, want 1", n)
	}

	body := `{"data":{"evidence_id":"` + requested.Evidence.ID +
		`","sha256":"abc123","final_key":"` + lot.ID + `/f.webp","thumbnail_key":"` + lot.ID + `/t.webp","processed_at":"2026-11-02T10:00:00Z"}}`
	if err := s.results.Handle(context.Background(), libmessaging.Delivery{Type: "evidence.processed", Body: []byte(body)}); err != nil {
		t.Fatalf("processed result: %v", err)
	}

	rec = get(s.mux, producerHeader, "/lots/"+lot.ID+"/evidences")
	var list struct {
		Items []evidenceView `json:"items"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&list); err != nil {
		t.Fatalf("body is not JSON: %v", err)
	}
	if len(list.Items) != 1 || list.Items[0].Status != "processed" || list.Items[0].SHA256 != "abc123" {
		t.Errorf("got %+v, want one processed evidence", list.Items)
	}
	if got := list.Items[0]; !strings.HasSuffix(got.ThumbnailURL, "/evidences/"+lot.ID+"/t.webp") ||
		!strings.HasSuffix(got.ImageURL, "/evidences/"+lot.ID+"/f.webp") || got.URLExpiresAt == "" {
		t.Errorf("got %+v, want view urls for the thumbnail and the image", got)
	}
}

func TestEvidenceListHidesViewURLsUntilProcessed(t *testing.T) {
	s := newEvidenceServer(t)
	lot, _ := lotWithFarmStage(t, s.mux)
	mustRequestUpload(t, s, producerHeader, lot.ID)

	rec := get(s.mux, producerHeader, "/lots/"+lot.ID+"/evidences")

	var list struct {
		Items []evidenceView `json:"items"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&list); err != nil || len(list.Items) != 1 {
		t.Fatalf("got %+v (%v), want one evidence", list.Items, err)
	}
	if got := list.Items[0]; got.ThumbnailURL != "" || got.ImageURL != "" || got.URLExpiresAt != "" {
		t.Errorf("got %+v, want no view urls for a pending evidence", got)
	}
}

type failingViewer struct{}

func (failingViewer) PresignView(context.Context, string, time.Duration) (ports.ViewURL, error) {
	return ports.ViewURL{}, errors.New("garage down")
}

func TestEvidenceListSurvivesAViewerFailure(t *testing.T) {
	s := newEvidenceServer(t, withEvidenceViewer(failingViewer{}))
	lot, _ := lotWithFarmStage(t, s.mux)
	requested := mustRequestUpload(t, s, producerHeader, lot.ID)
	s.storage.Put(quarantineKey(requested), 2048)
	completeUpload(s, producerHeader, lot.ID, requested.Evidence.ID)
	body := `{"data":{"evidence_id":"` + requested.Evidence.ID +
		`","sha256":"abc123","final_key":"` + lot.ID + `/f.webp","thumbnail_key":"` + lot.ID + `/t.webp","processed_at":"2026-11-02T10:00:00Z"}}`
	if err := s.results.Handle(context.Background(), libmessaging.Delivery{Type: "evidence.processed", Body: []byte(body)}); err != nil {
		t.Fatal(err)
	}

	rec := get(s.mux, producerHeader, "/lots/"+lot.ID+"/evidences")

	var list struct {
		Items []evidenceView `json:"items"`
	}
	if rec.Code != http.StatusOK || json.NewDecoder(rec.Body).Decode(&list) != nil || len(list.Items) != 1 {
		t.Fatalf("status %d, items %+v, want the list to still be served", rec.Code, list.Items)
	}
	if got := list.Items[0]; got.Status != "processed" || got.ImageURL != "" {
		t.Errorf("got %+v, want a processed evidence without view urls", got)
	}
}

func TestEvidenceUploadEndToEndErrors(t *testing.T) {
	tests := []struct {
		name     string
		actorID  string
		body     string
		want     int
		wantCode string
	}{
		{"missing actor", "", evidenceBody, http.StatusUnauthorized, "UNAUTHORIZED"},
		{"not the custodian", roasterHeader, evidenceBody, http.StatusForbidden, "FORBIDDEN"},
		{"not an image", producerHeader, `{"file_name":"a.pdf","content_type":"application/pdf","size_bytes":10}`,
			http.StatusBadRequest, string(exception.CodeEvidenceInvalidContentType)},
		{"empty file", producerHeader, `{"file_name":"a.jpg","content_type":"image/jpeg","size_bytes":0}`,
			http.StatusBadRequest, string(exception.CodeEvidenceEmptyFile)},
		{"too large", producerHeader, `{"file_name":"a.jpg","content_type":"image/jpeg","size_bytes":99999999}`,
			http.StatusBadRequest, string(exception.CodeEvidenceTooLarge)},
		{"unknown field", producerHeader, `{"file_name":"a.jpg","x":1}`, http.StatusBadRequest, "INVALID_REQUEST_BODY"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newEvidenceServer(t)
			lot, _ := lotWithFarmStage(t, s.mux)

			rec := evidenceRequest(s.mux, http.MethodPost, tt.actorID, "/lots/"+lot.ID+"/evidences", tt.body)

			assertErrorCode(t, rec, tt.want, tt.wantCode)
		})
	}
}

func TestEvidenceUploadEndToEndRequiresAStage(t *testing.T) {
	s := newEvidenceServer(t)
	lot := mustCreateLot(t, s.mux, "LOT-001")

	rec := evidenceRequest(s.mux, http.MethodPost, producerHeader, "/lots/"+lot.ID+"/evidences", evidenceBody)

	assertErrorCode(t, rec, http.StatusConflict, string(exception.CodeEvidenceLotWithoutStage))
}

func TestEvidenceUploadEndToEndOnlyTheUploaderCompletes(t *testing.T) {
	s := newEvidenceServer(t)
	lot, _ := lotWithFarmStage(t, s.mux)
	requested := mustRequestUpload(t, s, producerHeader, lot.ID)
	s.storage.Put(quarantineKey(requested), 2048)

	rec := completeUpload(s, roasterHeader, lot.ID, requested.Evidence.ID)

	assertErrorCode(t, rec, http.StatusForbidden, "FORBIDDEN")
}

func TestEvidenceUploadEndToEndRejectsAFileThatIsNotWhatWasAnnounced(t *testing.T) {
	s := newEvidenceServer(t)
	lot, _ := lotWithFarmStage(t, s.mux)
	requested := mustRequestUpload(t, s, producerHeader, lot.ID)
	s.storage.Put(quarantineKey(requested), 999999)

	rec := completeUpload(s, producerHeader, lot.ID, requested.Evidence.ID)

	assertErrorCode(t, rec, http.StatusConflict, string(exception.CodeEvidenceFileMismatch))
	if info, _ := s.storage.Stat(context.Background(), quarantineKey(requested)); info.Exists {
		t.Error("the mismatched file must be removed from quarantine")
	}
	if strings.Contains(s.logs.String(), `"event_type":"evidence.uploaded"`) {
		t.Error("nothing must be published for a mismatched file")
	}
}

func TestEvidenceResultsHandlerDoesNotRetryWhatCanNeverSucceed(t *testing.T) {
	s := newEvidenceServer(t)

	tests := []struct {
		name string
		body string
	}{
		{"malformed json", `{`},
		{"unknown evidence", `{"data":{"evidence_id":"6f1a7a5e-2f0b-4a58-9f55-2f8e1f6f2c11","sha256":"a","final_key":"f","thumbnail_key":"t","processed_at":"2026-11-02T10:00:00Z"}}`},
		{"invalid id", `{"data":{"evidence_id":"nope","processed_at":"2026-11-02T10:00:00Z"}}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := s.results.Handle(context.Background(), libmessaging.Delivery{Type: "evidence.processed", Body: []byte(tt.body)})

			if !libmessaging.IsPermanent(err) {
				t.Errorf("got %v, want a permanent error", err)
			}
		})
	}
}
