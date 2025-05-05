package purchases

import (
	"time"
)

type Purchase struct {
	ID        uint64 `json:"-"`
	ShortCode string `json:"short_code"`
	TargetURL string `json:"target_url"`
	Method    string `json:"method"`
	Price     uint64 `json:"price"`
	IsTest    bool   `json:"is_test"`

	PaidRouteID   uint64 `json:"-"`
	PaidToAddress string `json:"-"`

	PaymentPayload []byte `json:"-"`
	SettleResponse []byte `json:"-"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// DailyStats represents purchase statistics for a single day
type DailyStats struct {
	Date         string `json:"date"`
	Count        uint64 `json:"count"`
	Earnings     uint64 `json:"earnings"`
	TestCount    uint64 `json:"test_count"`
	TestEarnings uint64 `json:"test_earnings"`
	RealCount    uint64 `json:"real_count"`
	RealEarnings uint64 `json:"real_earnings"`
}
