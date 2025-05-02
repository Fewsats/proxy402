package purchases

// DailyStats represents purchase statistics for a single day
type DailyStats struct {
	Date         string `json:"date"`
	Count        int    `json:"count"`
	Earnings     int64  `json:"earnings"`
	TestCount    int    `json:"test_count"`
	TestEarnings int64  `json:"test_earnings"`
	RealCount    int    `json:"real_count"`
	RealEarnings int64  `json:"real_earnings"`
}
