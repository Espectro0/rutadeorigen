package request

import "encoding/json"

type RegisterStage struct {
	LotID     string          `json:"-"`
	IfMatch   string          `json:"-"`
	Stage     string          `json:"stage"`
	EventDate string          `json:"event_date"`
	Note      string          `json:"note"`
	Details   json.RawMessage `json:"details"`
}
