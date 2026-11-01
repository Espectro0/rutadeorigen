package response

type ScanStats struct {
	LotID       string       `json:"lot_id"`
	Total       int          `json:"total"`
	FirstScanAt *string      `json:"first_scan_at"`
	LastScanAt  *string      `json:"last_scan_at"`
	Days        int          `json:"days"`
	Daily       []DailyScans `json:"daily"`
}

type DailyScans struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}
