package response

type LotPage struct {
	Items      []Lot  `json:"items"`
	NextCursor string `json:"next_cursor,omitempty"`
}
