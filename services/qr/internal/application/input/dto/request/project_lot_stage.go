package request

type ProjectLotStage struct {
	MessageID  string
	EventType  string
	Version    int
	OccurredAt string
	Data       LotStageData
}

type LotStageData struct {
	LotID     string `json:"lot_id"`
	Stage     string `json:"stage"`
	Status    string `json:"status"`
	EventID   string `json:"event_id"`
	Sequence  int    `json:"sequence"`
	EventDate string `json:"event_date"`

	Details LotStageDetailsData `json:"details"`
}

type LotStageDetailsData struct {
	DryingMethod    string `json:"drying_method"`
	DryingDays      int    `json:"drying_days"`
	RoastLevel      string `json:"roast_level"`
	DurationMinutes int    `json:"duration_minutes"`
	Grind           string `json:"grind"`
	PackageWeightG  int    `json:"package_weight_g"`
	BestBefore      string `json:"best_before"`
}
