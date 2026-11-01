package request

type ListLots struct {
	FarmID      string
	CustodianID string
	Status      string
	Stage       string
	Cursor      string
	Limit       string
}
