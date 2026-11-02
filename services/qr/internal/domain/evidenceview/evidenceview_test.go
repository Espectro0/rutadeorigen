package evidenceview

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"rutadeorigen/qr/internal/domain/lotview"
	"rutadeorigen/qr/internal/platform/exception"
)

func validParams() NewParams {
	return NewParams{
		EvidenceID:   uuid.New(),
		LotID:        uuid.New(),
		Stage:        lotview.StageFarm,
		FinalKey:     " lot/ev/imagen.jpg ",
		ThumbnailKey: "lot/ev/miniatura.webp",
		ContentType:  "Image/JPEG",
		ProcessedAt:  time.Date(2026, time.November, 2, 10, 0, 0, 0, time.UTC),
	}
}

func TestNewNormalizesValidParams(t *testing.T) {
	v, err := New(validParams())

	if err != nil || v.IsDefault() {
		t.Fatalf("got %v %v", v, err)
	}
	if v.FinalKey() != "lot/ev/imagen.jpg" || v.ContentType() != "image/jpeg" || v.Stage() != lotview.StageFarm {
		t.Errorf("fields were not normalized: %+v", v)
	}
}

func TestNewRejectsInvalidParams(t *testing.T) {
	cases := map[string]func(p *NewParams){
		"no evidence id":   func(p *NewParams) { p.EvidenceID = uuid.Nil },
		"no lot id":        func(p *NewParams) { p.LotID = uuid.Nil },
		"unknown stage":    func(p *NewParams) { p.Stage = lotview.StageUnknown },
		"stage none":       func(p *NewParams) { p.Stage = lotview.StageNone },
		"no final key":     func(p *NewParams) { p.FinalKey = " " },
		"no thumbnail key": func(p *NewParams) { p.ThumbnailKey = "" },
		"unsupported type": func(p *NewParams) { p.ContentType = "image/gif" },
	}

	for name, breakIt := range cases {
		t.Run(name, func(t *testing.T) {
			p := validParams()
			breakIt(&p)

			v, err := New(p)

			if !exception.HasCode(err, exception.CodeEventInvalidPayload) || !v.IsDefault() {
				t.Errorf("want invalid payload and default view, got %v %v", v, err)
			}
		})
	}
}

func TestNewFillsMissingProcessedAt(t *testing.T) {
	p := validParams()
	p.ProcessedAt = time.Time{}

	v, err := New(p)

	if err != nil || v.ProcessedAt().IsZero() {
		t.Errorf("want a processed time, got %v %v", v, err)
	}
}

func TestDefaultIsDefault(t *testing.T) {
	if !Default().IsDefault() {
		t.Error("default view must report IsDefault")
	}
}
