package response

type PublicLot struct {
	ID          string         `json:"id"`
	Code        string         `json:"code"`
	Variety     string         `json:"variety"`
	Process     string         `json:"process"`
	HarvestDate string         `json:"harvest_date"`
	Farm        PublicFarm     `json:"farm"`
	Producer    PublicProducer `json:"producer"`
	Stage       string         `json:"stage"`
	Status      string         `json:"status"`
	VoidReason  string         `json:"void_reason,omitempty"`
	QRAvailable bool           `json:"qr_available"`
	UpdatedAt   string         `json:"updated_at"`

	Evidences []PublicEvidence `json:"evidences,omitempty"`
	Stages    []PublicStage    `json:"stages,omitempty"`
}

type PublicStage struct {
	Stage                string `json:"stage"`
	EventDate            string `json:"event_date"`
	DryingMethod         string `json:"drying_method,omitempty"`
	DryingDays           int    `json:"drying_days,omitempty"`
	RoastLevel           string `json:"roast_level,omitempty"`
	RoastDurationMinutes int    `json:"roast_duration_minutes,omitempty"`
	Grind                string `json:"grind,omitempty"`
	PackageWeightGrams   int    `json:"package_weight_grams,omitempty"`
	BestBefore           string `json:"best_before,omitempty"`
}

type PublicEvidence struct {
	ID           string `json:"id"`
	Stage        string `json:"stage"`
	ThumbnailURL string `json:"thumbnail_url"`
	ImageURL     string `json:"image_url"`
}

type PublicFarm struct {
	Name     string `json:"name"`
	Altitude int    `json:"altitude"`
}

type PublicProducer struct {
	Name string `json:"name"`
}
