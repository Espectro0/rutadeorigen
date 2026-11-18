package entity

import (
	"time"

	"github.com/google/uuid"
)

type StageView struct {
	LotID                uuid.UUID `db:"lot_id"`
	Stage                string    `db:"stage"`
	EventDate            time.Time `db:"event_date"`
	DryingMethod         string    `db:"drying_method"`
	DryingDays           int       `db:"drying_days"`
	RoastLevel           string    `db:"roast_level"`
	RoastDurationMinutes int       `db:"roast_duration_minutes"`
	Grind                string    `db:"grind"`
	PackageWeightGrams   int       `db:"package_weight_grams"`
	BestBefore           string    `db:"best_before"`
}
