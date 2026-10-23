package memory_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"rutadeorigen/qr/internal/domain/lotview"
	"rutadeorigen/qr/internal/infrastructure/output/repositories/semistructured/memory"
)

func TestLotViewCacheSetGetDelete(t *testing.T) {
	ctx := context.Background()
	cache := memory.NewLotViewCache()
	id := uuid.New()

	miss, err := cache.Get(ctx, id)
	if err != nil || !miss.IsDefault() {
		t.Fatalf("expected default on miss, got %v %v", miss, err)
	}

	view, err := lotview.New(lotview.NewParams{
		LotID:        id,
		Code:         "LT-001",
		Variety:      "Castillo",
		Process:      lotview.ProcessWashed,
		HarvestDate:  time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC),
		FarmID:       uuid.New(),
		FarmName:     "La Esperanza",
		FarmAltitude: 1800,
		ProducerID:   uuid.New(),
		ProducerName: "Maria Lopez",
		CustodianID:  uuid.New(),
		CurrentStage: lotview.StageNone,
		Status:       lotview.StatusActive,
		Version:      1,
		CreatedAt:    time.Now().Add(-time.Hour),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := cache.Set(ctx, view); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	hit, _ := cache.Get(ctx, id)
	if hit.IsDefault() || hit.LotID() != id {
		t.Error("expected cached view")
	}

	if err := cache.Delete(ctx, id); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	gone, _ := cache.Get(ctx, id)
	if !gone.IsDefault() {
		t.Error("expected default after delete")
	}
}
