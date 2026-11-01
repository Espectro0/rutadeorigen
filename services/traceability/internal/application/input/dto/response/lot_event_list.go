package response

type LotEventList struct {
	Items      []LotEvent `json:"items"`
	ChainValid bool       `json:"chain_valid"`
}
