package data

import (
	"time"

	"github.com/shopspring/decimal"
	"gitlab.calendaria.team/services/finance/billing/ent/enum"
)

type ItemDto struct {
	Name        string
	Description string
	TopicName   *string
}

type ProductDto struct {
	AppID        string
	Name         string
	Description  string
	Price        decimal.Decimal
	Currency     string
	IsActive     bool
	IsLimited    bool
	LimitedTill  *time.Time
	Left         int64
	IsUnique     bool
	UniqueLimit  int64
	IsExpiring   bool
	ExpiringTime *time.Time
	Bundles      []BundleDto
}

type BundleDto struct {
	ItemID int64
	Amount float64
}

type InvoiceDto struct {
	UserID         int64
	TenantID       int64
	AppID          string
	ProductID      int64
	Amount         int64
	Price          decimal.Decimal
	Status         enum.InvoiceStatus
	SubscriptionID int64

	PaidAt              *time.Time
	PaidTill            *time.Time
	IsPaidAtProcessed   *bool
	IsPaidTillProcessed *bool
}

type InvoiceFilter struct {
	ProductID      int64
	Status         enum.InvoiceStatus
	Paid           bool
	SubscriptionID int64
}

type SubscriptionDto struct {
	UserID    int64
	TenantID  int64
	AppID     string
	ProductID int64
}
