package request

type ProjectLotCreated struct {
	MessageID  string
	EventType  string
	Version    int
	OccurredAt string
	Data       LotCreatedData
}

type LotCreatedData struct {
	LotID       string          `json:"lot_id"`
	Farm        LotCreatedFarm  `json:"farm"`
	Producer    LotCreatedActor `json:"producer"`
	CustodianID string          `json:"custodian_id"`
	Code        string          `json:"code"`
	Variety     string          `json:"variety"`
	Process     string          `json:"process"`
	HarvestDate string          `json:"harvest_date"`
	Stage       string          `json:"stage"`
	Status      string          `json:"status"`
}

type LotCreatedFarm struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Altitude int    `json:"altitude"`
}

type LotCreatedActor struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
