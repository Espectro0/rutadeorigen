package lot_test

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"rutadeorigen/traceability/internal/domain/lot"
)

func filterLot(stage lot.Stage, status lot.Status) *lot.Lot {
	return lot.Restore(lot.RestoreParams{
		ID:           uuid.New(),
		FarmID:       uuid.New(),
		CustodianID:  uuid.New(),
		Code:         "LOT-001",
		Variety:      "Castillo",
		Process:      lot.ProcessWashed,
		HarvestDate:  time.Date(2026, time.September, 15, 0, 0, 0, 0, time.UTC),
		CurrentStage: stage,
		Status:       status,
		Version:      1,
		CreatedAt:    time.Date(2026, time.September, 20, 14, 30, 0, 0, time.UTC),
	})
}

func TestFilterMatches(t *testing.T) {
	l := filterLot(lot.StageRoasting, lot.StatusActive)

	tests := []struct {
		name   string
		filter lot.Filter
		want   bool
	}{
		{"empty filter matches everything", lot.Filter{}, true},
		{"same farm", lot.Filter{FarmID: l.FarmID()}, true},
		{"other farm", lot.Filter{FarmID: uuid.New()}, false},
		{"same custodian", lot.Filter{CustodianID: l.CustodianID()}, true},
		{"other custodian", lot.Filter{CustodianID: uuid.New()}, false},
		{"same status", lot.Filter{Status: lot.StatusActive}, true},
		{"other status", lot.Filter{Status: lot.StatusVoided}, false},
		{"same stage", lot.Filter{Stage: lot.StageRoasting}, true},
		{"other stage", lot.Filter{Stage: lot.StageFarm}, false},
		{"every criterion", lot.Filter{FarmID: l.FarmID(), CustodianID: l.CustodianID(), Status: lot.StatusActive, Stage: lot.StageRoasting}, true},
		{"one criterion fails", lot.Filter{FarmID: l.FarmID(), Status: lot.StatusActive, Stage: lot.StagePackaging}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.filter.Matches(l); got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLotIsCompleted(t *testing.T) {
	tests := []struct {
		name   string
		stage  lot.Stage
		status lot.Status
		want   bool
	}{
		{"packaged and active", lot.StagePackaging, lot.StatusActive, true},
		{"packaged and voided", lot.StagePackaging, lot.StatusVoided, false},
		{"roasted and active", lot.StageRoasting, lot.StatusActive, false},
		{"new and active", lot.StageNone, lot.StatusActive, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := filterLot(tt.stage, tt.status).IsCompleted(); got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}
