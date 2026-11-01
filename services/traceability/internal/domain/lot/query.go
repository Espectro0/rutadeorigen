package lot

import (
	"time"

	"github.com/google/uuid"

	"rutadeorigen/traceability/internal/platform/helper"
)

type Filter struct {
	FarmID      uuid.UUID
	CustodianID uuid.UUID
	Status      Status
	Stage       Stage
}

func (f Filter) HasFarm() bool {
	return !helper.IsDefaultUUID(f.FarmID)
}

func (f Filter) HasCustodian() bool {
	return !helper.IsDefaultUUID(f.CustodianID)
}

func (f Filter) HasStatus() bool {
	return f.Status != ""
}

func (f Filter) HasStage() bool {
	return f.Stage != ""
}

func (f Filter) Matches(l *Lot) bool {
	return (!f.HasFarm() || l.FarmID() == f.FarmID) &&
		(!f.HasCustodian() || l.CustodianID() == f.CustodianID) &&
		(!f.HasStatus() || l.Status() == f.Status) &&
		(!f.HasStage() || l.CurrentStage() == f.Stage)
}

type Cursor struct {
	CreatedAt time.Time
	ID        uuid.UUID
}

func (c Cursor) IsDefault() bool {
	return helper.IsDefaultUUID(c.ID)
}

func CursorOf(l *Lot) Cursor {
	return Cursor{CreatedAt: l.CreatedAt(), ID: l.ID()}
}

type Page struct {
	Lots []*Lot
	Next Cursor
}
