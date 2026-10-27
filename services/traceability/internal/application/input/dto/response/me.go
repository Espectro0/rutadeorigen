package response

type Me struct {
	Actor
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}
