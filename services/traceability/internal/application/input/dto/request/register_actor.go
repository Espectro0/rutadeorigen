package request

type RegisterActor struct {
	Type string `json:"type"`
	Name string `json:"name"`
}
