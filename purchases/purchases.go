package purchases

import (
	"time"
)

type Purchase struct {
	ID               uint64 `json:"-"`
	ShortCode        string `json:"short_code"`
	TargetURL        string `json:"target_url"`
	Method           string `json:"method"`
	Price            uint64 `json:"price"`
	Type             string `json:"type,omitempty"`
	CreditsAvailable uint64 `json:"credits_available,omitempty"`
	CreditsUsed      uint64 `json:"credits_used,omitempty"`
	IsTest           bool   `json:"is_test"`

	PaidRouteID   uint64 `json:"-"`
	PaidToAddress string `json:"-"`

	PaymentHeader  string `json:"-"` // Internal tracking, not usually in API response for purchase list
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
