package request

type ProcessEvidence struct {
	MessageID       string
	EventType       string
	EvidenceID      string
	LotID           string
	Stage           string
	QuarantineKey   string
	SizeBytes       int64
	UploaderActorID string
	UploaderRole    string
}
