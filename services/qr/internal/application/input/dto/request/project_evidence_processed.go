package request

type ProjectEvidenceProcessed struct {
	MessageID  string
	EventType  string
	Version    int
	OccurredAt string
	Data       EvidenceProcessedData
}

type EvidenceProcessedData struct {
	EvidenceID   string `json:"evidence_id"`
	LotID        string `json:"lot_id"`
	Stage        string `json:"stage"`
	FinalKey     string `json:"final_key"`
	ThumbnailKey string `json:"thumbnail_key"`
	ContentType  string `json:"content_type"`
	ProcessedAt  string `json:"processed_at"`
}
