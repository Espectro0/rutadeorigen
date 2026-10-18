package lot

import (
	"time"

	"github.com/google/uuid"

	"rutadeorigen/traceability/internal/platform/helper"
)

type Filter struct {
	FarmID      uuid.UUID
	CustodianID uuid.UUID
}

func (f Filter) HasFarm() bool {
	return !helper.IsDefaultUUID(f.FarmID)
}

func (f Filter) HasCustodian() bool {
	return !helper.IsDefaultUUID(f.CustodianID)
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
