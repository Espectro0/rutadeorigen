package response

type TransferredLot struct {
	Event      LotEvent `json:"event"`
	LotVersion int      `json:"lot_version"`
}
