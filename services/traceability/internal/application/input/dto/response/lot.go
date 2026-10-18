package response

type Lot struct {
	ID          string `json:"id"`
	Version     int    `json:"version"`
	FarmID      string `json:"farm_id"`
	CustodianID string `json:"custodian_id"`
	Code        string `json:"code"`
	Variety     string `json:"variety"`
	Process     string `json:"process"`
	HarvestDate string `json:"harvest_date"`
	Stage       string `json:"stage"`
	Status      string `json:"status"`
	VoidReason  string `json:"void_reason,omitempty"`
	CreatedAt   string `json:"created_at"`
}
