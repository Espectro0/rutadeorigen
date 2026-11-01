package response

type LotDetail struct {
	Lot
	AllowedActions []string `json:"allowed_actions"`
}
