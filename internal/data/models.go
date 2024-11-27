package data

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/shopspring/decimal"
	"gitlab.calendaria.team/services/finance/billing/ent/enum"
)

type NotificationType string
type Subtype string
type ConsumptionRequestReason string
type Environment string
type OwnershipType string
type OfferDiscountType string
type ProductType string

const (
	TYPE_SUBSCRIBED                NotificationType = "SUBSCRIBED"
	TYPE_DID_CHANGE_RENEWAL_PREF   NotificationType = "DID_CHANGE_RENEWAL_PREF"
	TYPE_DID_CHANGE_RENEWAL_STATUS NotificationType = "DID_CHANGE_RENEWAL_STATUS"
	TYPE_OFFER_REDEEMED            NotificationType = "OFFER_REDEEMED"
	TYPE_DID_RENEW                 NotificationType = "DID_RENEW"
	TYPE_EXPIRED                   NotificationType = "EXPIRED"
	TYPE_DID_FAIL_TO_RENEW         NotificationType = "DID_FAIL_TO_RENEW"
	TYPE_GRACE_PERIOD_EXPIRED      NotificationType = "GRACE_PERIOD_EXPIRED"
	TYPE_PRICE_INCREASE            NotificationType = "PRICE_INCREASE"
	TYPE_REFUND                    NotificationType = "REFUND"
	TYPE_REFUND_DECLINED           NotificationType = "REFUND_DECLINED"
	TYPE_CONSUMPTION_REQUEST       NotificationType = "CONSUMPTION_REQUEST"
	TYPE_RENEWAL_EXTENDED          NotificationType = "RENEWAL_EXTENDED"
	TYPE_REVOKE                    NotificationType = "REVOKE"
	TYPE_TEST                      NotificationType = "TEST"
	TYPE_RENEWAL_EXTENSION         NotificationType = "RENEWAL_EXTENSION"
	TYPE_REFUND_REVERSED           NotificationType = "REFUND_REVERSED"
	TYPE_EXTERNAL_PURCHASE_TOKEN   NotificationType = "EXTERNAL_PURCHASE_TOKEN"
	TYPE_ONE_TIME_CHARGE           NotificationType = "ONE_TIME_CHARGE"

	SUBTYPE_INITIAL_BUY          Subtype = "INITIAL_BUY"
	SUBTYPE_RESUBSCRIBE          Subtype = "RESUBSCRIBE"
	SUBTYPE_DOWNGRADE            Subtype = "DOWNGRADE"
	SUBTYPE_UPGRADE              Subtype = "UPGRADE"
	SUBTYPE_AUTO_RENEW_ENABLED   Subtype = "AUTO_RENEW_ENABLED"
	SUBTYPE_AUTO_RENEW_DISABLED  Subtype = "AUTO_RENEW_DISABLED"
	SUBTYPE_VOLUNTARY            Subtype = "VOLUNTARY"
	SUBTYPE_BILLING_RETRY        Subtype = "BILLING_RETRY"
	SUBTYPE_PRICE_INCREASE       Subtype = "PRICE_INCREASE"
	SUBTYPE_GRACE_PERIOD         Subtype = "GRACE_PERIOD"
	SUBTYPE_PENDING              Subtype = "PENDING"
	SUBTYPE_ACCEPTED             Subtype = "ACCEPTED"
	SUBTYPE_BILLING_RECOVERY     Subtype = "BILLING_RECOVERY"
	SUBTYPE_PRODUCT_NOT_FOR_SALE Subtype = "PRODUCT_NOT_FOR_SALE"
	SUBTYPE_SUMMARY              Subtype = "SUMMARY"
	SUBTYPE_FAILURE              Subtype = "FAILURE"
	SUBTYPE_UNREPORTED           Subtype = "UNREPORTED"

	UNINTENDED_PURCHASE       ConsumptionRequestReason = "UNINTENDED_PURCHASE"
	FULFILLMENT_ISSUE         ConsumptionRequestReason = "FULFILLMENT_ISSUE"
	UNSATISFIED_WITH_PURCHASE ConsumptionRequestReason = "UNSATISFIED_WITH_PURCHASE"
	LEGAL                     ConsumptionRequestReason = "LEGAL"
	OTHER                     ConsumptionRequestReason = "OTHER"

	Sandbox    Environment = "Sandbox"
	Production Environment = "Production"

	OWNERSHIP_FAMILY_SHARED OwnershipType = "FAMILY_SHARED"
	OWNERSHIP_PURCHASED     OwnershipType = "PURCHASED"

	DISCOUNT_FREE_TRIAL    OfferDiscountType = "FREE_TRIAL"
	DISCOUNT_PAY_AS_YOU_GO OfferDiscountType = "PAY_AS_YOU_GO"
	DISCOUNT_PAY_UP_FRONT  OfferDiscountType = "PAY_UP_FRONT"

	PRODUCT_AUTO_RENEWABLE ProductType = "Auto-Renewable Subscription"
	PRODUCT_NON_RENEWABLE  ProductType = "Non-Renewing Subscription"
	PRODUCT_NON_CONSUMABLE ProductType = "Non-Consumable"
	PRODUCT_CONSUMABLE     ProductType = "Consumable"
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

	IsRevoked           *bool
	RevokedAt           *time.Time
	PaidAt              *time.Time
	PaidTill            *time.Time
	IsRevokedProcessed  *bool
	IsPaidAtProcessed   *bool
	IsPaidTillProcessed *bool

	AppleStoreTransactionID *string
}

type InvoiceFilter struct {
	UserID         int64
	ProductID      int64
	Status         enum.InvoiceStatus
	Paid           bool
	PaidProcesses  *bool
	IsRevoked      *bool
	IsRevokedProc  *bool
	PaidTill       *time.Time
	SubscriptionID int64
}

type SubscriptionDto struct {
	UserID    int64
	TenantID  int64
	AppID     string
	ProductID int64
}

type Payload struct {
	NotificationType NotificationType
	Subtype          Subtype
	Data             struct {
		AppAppleId               int64
		BundleID                 string
		BundleVersion            string
		ConsumptionRequestReason string
		Environment              Environment
		SignedRenewalInfo        string
		SignedTransactionInfo    string
		Status                   int64
	}
	Summary struct {
		RequestIdentifier      string
		Environment            Environment
		AppAppleID             int64
		BundleID               string
		ProductID              string
		StoreFrontCountryCodes []string
		FailedCount            int64
		SucceededCount         int64
	}
	ExternalPurchaseToken struct {
		ExternalPurchaseID string
		TokenCreationDate  int64
		AppAppleID         int64
		BundleID           string
	}
	Version          string
	SignedDate       int64
	NotificationUUID string
}

type JWSTransaction struct {
	AppAccountToken             string
	BundleID                    string
	Currency                    string
	Environment                 Environment
	ExpiresDate                 int64
	InAppOwnershipType          OwnershipType
	OfferDiscountType           OfferDiscountType
	OfferIdentifier             string
	OfferType                   int64
	OriginalPurchaseDate        int64
	OriginalTransactionID       string
	Price                       int64 // holds price in milliunits
	ProductID                   string
	PurchaseDate                int64
	Quantity                    int64
	RevocationDate              int64
	RevocationReason            int64
	SignedDate                  int64
	StoreFront                  string
	StoreFrontID                string
	SubscriptionGroupIdentifier string
	TransactionID               string
	TransactionReason           string
	Type                        ProductType
	WebOrderLineItemID          string
}

type RenewalInfo struct {
	AutoRenewProductID          string
	AutoRenewStatus             int64
	Currency                    string
	EligibleWinBackOfferIDs     []string
	Environment                 Environment
	ExpirationIntent            int64
	GracePeriodExpiresDate      int64
	IsInBillingRetryPeriod      bool
	OfferDiscountType           OfferDiscountType
	OfferIdentifier             string
	OfferType                   int64
	OriginalTransactionID       string
	PriceIncreaseStatus         int64
	ProductID                   string
	RecentSubscriptionStartDate int64
	RenewalDate                 int64
	RenewalPrice                int64 // holds price in milliunits
	SignedDate                  int64
}

func ParseAppleSignedBody(signedBody string) (*jwt.Token, error) {
	parse, err := jwt.Parse(signedBody, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodECDSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}

		if t.Header["x5c"] == nil || len(t.Header["x5c"].([]interface{})) != 3 {
			return nil, errors.New("invalid x5c header")
		}

		root := fmt.Sprintf("-----BEGIN CERTIFICATE-----\n%s\n-----END CERTIFICATE-----",
			t.Header["x5c"].([]interface{})[2].(string))
		intermediate := fmt.Sprintf("-----BEGIN CERTIFICATE-----\n%s\n-----END CERTIFICATE-----",
			t.Header["x5c"].([]interface{})[1].(string))
		cert := fmt.Sprintf("-----BEGIN CERTIFICATE-----\n%s\n-----END CERTIFICATE-----",
			t.Header["x5c"].([]interface{})[0].(string))

		roots := x509.NewCertPool()
		ok := roots.AppendCertsFromPEM([]byte(root))
		if !ok {
			return nil, errors.New("failed to parse root certificate")
		}

		ok = roots.AppendCertsFromPEM([]byte(intermediate))
		if !ok {
			return nil, errors.New("failed to parse intermediate certificate")
		}

		block, _ := pem.Decode([]byte(cert))
		if block == nil {
			return nil, errors.New("failed to parse PEM block")
		}

		c, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, err
		}

		opts := x509.VerifyOptions{
			Roots: roots,
		}

		_, err = c.Verify(opts)
		if err != nil {
			return nil, err
		}

		return c.PublicKey, nil
	})
	if err != nil {
		return nil, err
	}

	return parse, nil
}
