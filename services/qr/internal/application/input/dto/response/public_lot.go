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
