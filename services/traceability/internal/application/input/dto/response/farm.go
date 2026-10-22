package response

type Farm struct {
	ID         string `json:"id"`
	ProducerID string `json:"producer_id"`
	Name       string `json:"name"`
	Altitude   int    `json:"altitude"`
	CreatedAt  string `json:"created_at"`
}
