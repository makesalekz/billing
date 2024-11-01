package data

import (
	"time"

	"github.com/shopspring/decimal"
	"gitlab.calendaria.team/services/finance/invoices/ent/enum"
)

type ItemDto struct {
	Name        string
	Description string
}

type ProductDto struct {
	Name        string
	Description string
	Price       decimal.Decimal
	Currency    string
	IsActive    bool
	IsLimited   bool
	LimitedTill *time.Time
	Left        int64
	IsUnique    bool
	UniqueLimit int64
}

type InvoiceDto struct {
	ActorID   int64
	AppID     string
	ProductID int64
	Amount    int64
	Price     decimal.Decimal
	Status    enum.InvoiceStatus
	PaidAt    *time.Time
}

type InvoiceFilter struct {
	ProductID int64
	Status    enum.InvoiceStatus
}
