package request

type SendEvidenceNotification struct {
	MessageID     string
	EventType     string
	EvidenceID    string
	LotID         string
	LotCode       string
	Stage         string
	FileName      string
	Reason        string
	UploaderName  string
	UploaderEmail string
}
