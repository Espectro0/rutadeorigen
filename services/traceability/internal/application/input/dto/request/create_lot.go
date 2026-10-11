package request

type CreateLot struct {
	FarmID      string `json:"farm_id"`
	Code        string `json:"code"`
	Variety     string `json:"variety"`
	Process     string `json:"process"`
	HarvestDate string `json:"harvest_date"`
}
