package request

type VoidLot struct {
	LotID   string `json:"-"`
	IfMatch string `json:"-"`
	Reason  string `json:"reason"`
}
