package response

type ActorSummary struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	Name string `json:"name"`
}

type ActorList struct {
	Items []ActorSummary `json:"items"`
}
