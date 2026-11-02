package listpublicevidences_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"rutadeorigen/qr/internal/application/output/ports"
	"rutadeorigen/qr/internal/application/usecases/listpublicevidences"
	"rutadeorigen/qr/internal/domain/evidenceview"
	"rutadeorigen/qr/internal/domain/lotview"
)

type fakeRepository struct {
	items    []*evidenceview.EvidenceView
	err      error
	gotLimit int
}

func (r *fakeRepository) Save(context.Context, *evidenceview.EvidenceView) error { return nil }

func (r *fakeRepository) ListByLotID(_ context.Context, _ uuid.UUID, limit int) ([]*evidenceview.EvidenceView, error) {
	r.gotLimit = limit
	return r.items, r.err
}

type fakeViewer struct {
	failFor map[string]bool
	signed  []string
}

func (v *fakeViewer) PresignView(_ context.Context, key string, ttl time.Duration) (ports.ViewURL, error) {
	if v.failFor[key] {
		return ports.ViewURL{}, errors.New("garage down")
	}
	v.signed = append(v.signed, key)
	return ports.ViewURL{URL: "https://signed/" + key, ExpiresAt: time.Now().Add(ttl)}, nil
}

func lotWithStatus(status lotview.Status) *lotview.LotView {
	return lotview.Restore(lotview.RestoreParams{
		LotID: uuid.New(), Code: "LT-001", Variety: "Castillo", Process: lotview.ProcessWashed,
		HarvestDate: time.Now(), FarmName: "La Esperanza", FarmAltitude: 1800, ProducerName: "Maria",
		CurrentStage: lotview.StageFarm, Status: status, Version: 1,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	})
}

func storedEvidence(t *testing.T, name string) *evidenceview.EvidenceView {
	t.Helper()
	view, err := evidenceview.New(evidenceview.NewParams{
		EvidenceID: uuid.New(), LotID: uuid.New(), Stage: lotview.StageFarm,
		FinalKey: name + "/imagen.jpg", ThumbnailKey: name + "/miniatura.webp", ContentType: "image/jpeg",
	})
	if err != nil {
		t.Fatal(err)
	}
	return view
}

var settings = listpublicevidences.Settings{ViewTTL: time.Hour, MaxPerLot: 12}

func TestExecuteSignsThumbnailAndImage(t *testing.T) {
	repository := &fakeRepository{items: []*evidenceview.EvidenceView{storedEvidence(t, "a")}}

	items, err := listpublicevidences.New(repository, &fakeViewer{}, settings).
		Execute(context.Background(), lotWithStatus(lotview.StatusActive))

	if err != nil || len(items) != 1 {
		t.Fatalf("got %+v %v", items, err)
	}
	if items[0].ThumbnailURL != "https://signed/a/miniatura.webp" || items[0].ImageURL != "https://signed/a/imagen.jpg" {
		t.Errorf("unexpected urls %+v", items[0])
	}
	if repository.gotLimit != 12 {
		t.Errorf("the per-lot limit must reach the repository, got %d", repository.gotLimit)
	}
}

func TestExecuteReturnsNothingForVoidedOrUnknownLots(t *testing.T) {
	repository := &fakeRepository{items: []*evidenceview.EvidenceView{storedEvidence(t, "a")}}
	useCase := listpublicevidences.New(repository, &fakeViewer{}, settings)

	for name, lot := range map[string]*lotview.LotView{
		"voided":  lotWithStatus(lotview.StatusVoided),
		"default": lotview.Default(),
	} {
		items, err := useCase.Execute(context.Background(), lot)

		if err != nil || len(items) != 0 {
			t.Errorf("%s lot: want no evidences, got %+v %v", name, items, err)
		}
	}
	if repository.gotLimit != 0 {
		t.Error("the repository must not be queried for a lot without public photos")
	}
}

func TestExecuteSkipsEvidenceWhoseUrlCannotBeSigned(t *testing.T) {
	repository := &fakeRepository{items: []*evidenceview.EvidenceView{storedEvidence(t, "a"), storedEvidence(t, "b")}}
	viewer := &fakeViewer{failFor: map[string]bool{"a/miniatura.webp": true}}

	items, err := listpublicevidences.New(repository, viewer, settings).
		Execute(context.Background(), lotWithStatus(lotview.StatusActive))

	if err != nil || len(items) != 1 || items[0].ThumbnailURL != "https://signed/b/miniatura.webp" {
		t.Errorf("want only the evidence that could be signed, got %+v %v", items, err)
	}
}

func TestExecuteReturnsRepositoryErrors(t *testing.T) {
	failure := errors.New("db down")
	repository := &fakeRepository{err: failure}

	items, err := listpublicevidences.New(repository, &fakeViewer{}, settings).
		Execute(context.Background(), lotWithStatus(lotview.StatusActive))

	if !errors.Is(err, failure) || len(items) != 0 {
		t.Errorf("want the repository error and no items, got %+v %v", items, err)
	}
}
