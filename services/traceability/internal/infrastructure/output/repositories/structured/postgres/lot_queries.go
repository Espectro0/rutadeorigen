package postgres

import (
	"fmt"
	"strings"

	"rutadeorigen/traceability/internal/domain/lot"
)

const selectLot = `
	SELECT id, farm_id, custodian_id, code, variety, process,
	       harvest_date, current_stage, status, void_reason, version, created_at
	FROM lot`

func listLotsQuery(f lot.Filter, after lot.Cursor, limit int) (string, []any) {
	var sb strings.Builder
	args := make([]any, 0, 5)

	sb.WriteString(selectLot)
	sb.WriteString(" WHERE TRUE")

	if f.HasFarm() {
		args = append(args, f.FarmID)
		fmt.Fprintf(&sb, " AND farm_id = $%d", len(args))
	}
	if f.HasCustodian() {
		args = append(args, f.CustodianID)
		fmt.Fprintf(&sb, " AND custodian_id = $%d", len(args))
	}
	if !after.IsDefault() {
		args = append(args, after.CreatedAt, after.ID)
		fmt.Fprintf(&sb, " AND (created_at, id) < ($%d, $%d)", len(args)-1, len(args))
	}

	args = append(args, limit)
	fmt.Fprintf(&sb, " ORDER BY created_at DESC, id DESC LIMIT $%d", len(args))
	return sb.String(), args
}
