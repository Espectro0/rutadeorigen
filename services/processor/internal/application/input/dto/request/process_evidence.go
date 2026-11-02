package request

type ProcessEvidence struct {
	MessageID       string
	EventType       string
	EvidenceID      string
	LotID           string
	LotCode         string
	Stage           string
	FileName        string
	QuarantineKey   string
	SizeBytes       int64
	UploaderActorID string
	UploaderRole    string
	UploaderName    string
	UploaderEmail   string
}
