package request

type ListLots struct {
	FarmID      string
	CustodianID string
	Cursor      string
	Limit       string
}
