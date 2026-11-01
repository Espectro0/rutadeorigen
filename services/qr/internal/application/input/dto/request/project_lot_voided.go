package request

type ProjectLotVoided struct {
	MessageID  string
	EventType  string
	Version    int
	OccurredAt string
	Data       LotVoidedData
}

type LotVoidedData struct {
	LotID       string `json:"lot_id"`
	VoidedBy    string `json:"voided_by"`
	Reason      string `json:"reason"`
	CustodianID string `json:"custodian_id"`
	Stage       string `json:"stage"`
	Status      string `json:"status"`
}
