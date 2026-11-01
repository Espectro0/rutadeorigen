package request

type TransferLot struct {
	LotID         string `json:"-"`
	IfMatch       string `json:"-"`
	TargetActorID string `json:"target_actor_id"`
	Note          string `json:"note"`
}
