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
}
