package postgres

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"rutadeorigen/traceability/internal/domain/lot"
)

func TestListLotsQueryWithoutFiltersOnlyOrdersAndLimits(t *testing.T) {
	query, args := listLotsQuery(lot.Filter{}, lot.Cursor{}, 21)

	if strings.Contains(query, "farm_id =") || strings.Contains(query, "status =") || strings.Contains(query, "current_stage =") {
		t.Errorf("unexpected filter in %q", query)
	}
	if !strings.HasSuffix(query, "ORDER BY created_at DESC, id DESC LIMIT $1") || !reflect.DeepEqual(args, []any{21}) {
		t.Errorf("got %q with %v", query, args)
	}
}

func TestListLotsQueryAppliesEveryFilterInOrder(t *testing.T) {
	farmID, custodianID, cursorID := uuid.New(), uuid.New(), uuid.New()
	createdAt := time.Date(2026, time.September, 20, 14, 30, 0, 0, time.UTC)

	query, args := listLotsQuery(
		lot.Filter{FarmID: farmID, CustodianID: custodianID, Status: lot.StatusActive, Stage: lot.StageRoasting},
		lot.Cursor{CreatedAt: createdAt, ID: cursorID},
		11,
	)

	for _, want := range []string{
		"AND farm_id = $1",
		"AND custodian_id = $2",
		"AND status = $3",
		"AND current_stage = $4",
		"AND (created_at, id) < ($5, $6)",
		"LIMIT $7",
	} {
		if !strings.Contains(query, want) {
			t.Errorf("missing %q in %q", want, query)
		}
	}
	wantArgs := []any{farmID, custodianID, "active", "roasting", createdAt, cursorID, 11}
	if !reflect.DeepEqual(args, wantArgs) {
		t.Errorf("got args %v, want %v", args, wantArgs)
	}
}

func TestListLotsQueryFiltersByStatusAndStageAlone(t *testing.T) {
	query, args := listLotsQuery(lot.Filter{Status: lot.StatusVoided, Stage: lot.StageNone}, lot.Cursor{}, 5)

	if !strings.Contains(query, "AND status = $1") || !strings.Contains(query, "AND current_stage = $2") || !strings.Contains(query, "LIMIT $3") {
		t.Errorf("unexpected query %q", query)
	}
	if !reflect.DeepEqual(args, []any{"voided", "none", 5}) {
		t.Errorf("got args %v", args)
	}
}
