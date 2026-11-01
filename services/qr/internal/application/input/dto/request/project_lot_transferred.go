package request

type ProjectLotTransferred struct {
	MessageID  string
	EventType  string
	Version    int
	OccurredAt string
	Data       LotTransferredData
}

type LotTransferredData struct {
	LotID       string `json:"lot_id"`
	FromActorID string `json:"from_actor_id"`
	CustodianID string `json:"custodian_id"`
	Stage       string `json:"stage"`
	Status      string `json:"status"`
}
