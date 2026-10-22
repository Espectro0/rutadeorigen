package request

type CreateFarm struct {
	Name     string `json:"name"`
	Altitude int    `json:"altitude"`
}
