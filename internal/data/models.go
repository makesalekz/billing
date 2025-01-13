package data

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"time"

	"gitlab.calendaria.team/services/finance/billing/ent/enum"

	"github.com/golang-jwt/jwt/v5"
	"github.com/shopspring/decimal"
)

type NotificationType string
type NotificationSubtype string
type ConsumptionRequestReason string
type Environment string
type OwnershipType string
type OfferDiscountType string
type ProductType string

const (
	TypeSubscribed             NotificationType = "SUBSCRIBED"
	TypeDidChangeRenewalPref   NotificationType = "DID_CHANGE_RENEWAL_PREF"
	TypeDidChangeRenewalStatus NotificationType = "DID_CHANGE_RENEWAL_STATUS"
	TypeOfferRedeemed          NotificationType = "OFFER_REDEEMED"
	TypeDidRenew               NotificationType = "DID_RENEW"
	TypeExpired                NotificationType = "EXPIRED"
	TypeDidFailToRenew         NotificationType = "DID_FAIL_TO_RENEW"
	TypeGracePeriodExpired     NotificationType = "GRACE_PERIOD_EXPIRED"
	TypePriceIncrease          NotificationType = "PRICE_INCREASE"
	TypeRefund                 NotificationType = "REFUND"
	TypeRefundDeclined         NotificationType = "REFUND_DECLINED"
	TypeConsumptionRequest     NotificationType = "CONSUMPTION_REQUEST"
	TypeRenewalExtended        NotificationType = "RENEWAL_EXTENDED"
	TypeRevoke                 NotificationType = "REVOKE"
	TypeTest                   NotificationType = "TEST"
	TypeRenewalExtension       NotificationType = "RENEWAL_EXTENSION"
	TypeRefundReversed         NotificationType = "REFUND_REVERSED"

	//nolint:gosec // external purchase token is notification type and not some credential
	TypeExternalPurchaseToken NotificationType = "EXTERNAL_PURCHASE_TOKEN"
	TypeOneTimeCharge         NotificationType = "ONE_TIME_CHARGE"

	SubtypeInitialBuy        NotificationSubtype = "INITIAL_BUY"
	SubtypeResubscribe       NotificationSubtype = "RESUBSCRIBE"
	SubtypeDowngrade         NotificationSubtype = "DOWNGRADE"
	SubtypeUpgrade           NotificationSubtype = "UPGRADE"
	SubtypeAutoRenewEnabled  NotificationSubtype = "AUTO_RENEW_ENABLED"
	SubtypeAutoRenewDisabled NotificationSubtype = "AUTO_RENEW_DISABLED"
	SubtypeVoluntary         NotificationSubtype = "VOLUNTARY"
	SubtypeBillingRetry      NotificationSubtype = "BILLING_RETRY"
	SubtypePriceIncrease     NotificationSubtype = "PRICE_INCREASE"
	SubtypeGracePeriod       NotificationSubtype = "GRACE_PERIOD"
	SubtypePending           NotificationSubtype = "PENDING"
	SubtypeAccepted          NotificationSubtype = "ACCEPTED"
	SubtypeBillingRecovery   NotificationSubtype = "BILLING_RECOVERY"
	SubtypeProductNotForSale NotificationSubtype = "PRODUCT_NOT_FOR_SALE"
	SubtypeSummary           NotificationSubtype = "SUMMARY"
	SubtypeFailure           NotificationSubtype = "FAILURE"
	SubtypeUnreported        NotificationSubtype = "UNREPORTED"

	UnintendedPurchase      ConsumptionRequestReason = "UNINTENDED_PURCHASE"
	FulfillmentIssue        ConsumptionRequestReason = "FULFILLMENT_ISSUE"
	UnsatisfiedWithPurchase ConsumptionRequestReason = "UNSATISFIED_WITH_PURCHASE"
	Legal                   ConsumptionRequestReason = "LEGAL"
	Other                   ConsumptionRequestReason = "OTHER"

	Sandbox    Environment = "Sandbox"
	Production Environment = "Production"

	OwnershipFamilyShared OwnershipType = "FAMILY_SHARED"
	OwnershipPurchased    OwnershipType = "PURCHASED"

	DiscountFreeTrial  OfferDiscountType = "FREE_TRIAL"
	DiscountPayAsYouGo OfferDiscountType = "PAY_AS_YOU_GO"
	DiscountPayUpFront OfferDiscountType = "PAY_UP_FRONT"

	ProductAutoRenewable ProductType = "Auto-Renewable Subscription"
	ProductNonRenewable  ProductType = "Non-Renewing Subscription"
	ProductNonConsumable ProductType = "Non-Consumable"
	ProductConsumable    ProductType = "Consumable"
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
	RecurrentProfileID  *int64

	AppleStoreTransactionID *string
	OneVisionTransactionID  *string
	IsTrial                 bool
}

type InvoiceFilter struct {
	UserID         int64
	ProductID      int64
	Status         enum.InvoiceStatus
	Paid           bool
	PaidProcesses  *bool
	IsRevoked      *bool
	IsRevokedProc  *bool
	PaidTillProc   *bool
	SubscriptionID int64
	PaidTill       *time.Time
}

type SubscriptionDto struct {
	UserID    int64
	TenantID  int64
	AppID     string
	ProductID int64
}

type Payload struct {
	NotificationType NotificationType    `json:"notificationType,omitempty"`
	Subtype          NotificationSubtype `json:"subtype,omitempty"`
	Data             struct {
		AppAppleID               int64       `json:"appAppleID,omitempty"`
		BundleID                 string      `json:"bundleID,omitempty"`
		BundleVersion            string      `json:"bundleVersion,omitempty"`
		ConsumptionRequestReason string      `json:"consumptionRequestReason,omitempty"`
		Environment              Environment `json:"environment,omitempty"`
		SignedRenewalInfo        string      `json:"signedRenewalInfo,omitempty"`
		SignedTransactionInfo    string      `json:"signedTransactionInfo,omitempty"`
		Status                   int64       `json:"status,omitempty"`
	} `json:"data"`
	Summary struct {
		RequestIdentifier      string      `json:"requestIdentifier,omitempty"`
		Environment            Environment `json:"environment,omitempty"`
		AppAppleID             int64       `json:"appAppleID,omitempty"`
		BundleID               string      `json:"bundleID,omitempty"`
		ProductID              string      `json:"productID,omitempty"`
		StoreFrontCountryCodes []string    `json:"storeFrontCountryCodes,omitempty"`
		FailedCount            int64       `json:"failedCount,omitempty"`
		SucceededCount         int64       `json:"succeededCount,omitempty"`
	} `json:"summary"`
	ExternalPurchaseToken struct {
		ExternalPurchaseID string `json:"externalPurchaseID,omitempty"`
		TokenCreationDate  int64  `json:"tokenCreationDate,omitempty"`
		AppAppleID         int64  `json:"appAppleID,omitempty"`
		BundleID           string `json:"bundleID,omitempty"`
	} `json:"externalPurchaseToken"`
	Version          string `json:"version,omitempty"`
	SignedDate       int64  `json:"signedDate,omitempty"`
	NotificationUUID string `json:"notificationUUID,omitempty"`
}

type JWSTransaction struct {
	AppAccountToken             string            `json:"appAccountToken,omitempty"`
	BundleID                    string            `json:"bundleID,omitempty"`
	Currency                    string            `json:"currency,omitempty"`
	Environment                 Environment       `json:"environment,omitempty"`
	ExpiresDate                 int64             `json:"expiresDate,omitempty"`
	InAppOwnershipType          OwnershipType     `json:"inAppOwnershipType,omitempty"`
	OfferDiscountType           OfferDiscountType `json:"offerDiscountType,omitempty"`
	OfferIdentifier             string            `json:"offerIdentifier,omitempty"`
	OfferType                   int64             `json:"offerType,omitempty"`
	OriginalPurchaseDate        int64             `json:"originalPurchaseDate,omitempty"`
	OriginalTransactionID       string            `json:"originalTransactionID,omitempty"`
	Price                       int64             `json:"price,omitempty"` // holds price in milliunits
	ProductID                   string            `json:"productID,omitempty"`
	PurchaseDate                int64             `json:"purchaseDate,omitempty"`
	Quantity                    int64             `json:"quantity,omitempty"`
	RevocationDate              int64             `json:"revocationDate,omitempty"`
	RevocationReason            int64             `json:"revocationReason,omitempty"`
	SignedDate                  int64             `json:"signedDate,omitempty"`
	StoreFront                  string            `json:"storeFront,omitempty"`
	StoreFrontID                string            `json:"storeFrontID,omitempty"`
	SubscriptionGroupIdentifier string            `json:"subscriptionGroupIdentifier,omitempty"`
	TransactionID               string            `json:"transactionID,omitempty"`
	TransactionReason           string            `json:"transactionReason,omitempty"`
	Type                        ProductType       `json:"type,omitempty"`
	WebOrderLineItemID          string            `json:"webOrderLineItemID,omitempty"`
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
	parse, err := jwt.Parse(
		signedBody, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodECDSA); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}

			if t.Header["x5c"] == nil || len(t.Header["x5c"].([]interface{})) != 3 {
				return nil, errors.New("invalid x5c header")
			}

			root := fmt.Sprintf(
				"-----BEGIN CERTIFICATE-----\n%s\n-----END CERTIFICATE-----",
				t.Header["x5c"].([]interface{})[2].(string),
			)
			intermediate := fmt.Sprintf(
				"-----BEGIN CERTIFICATE-----\n%s\n-----END CERTIFICATE-----",
				t.Header["x5c"].([]interface{})[1].(string),
			)
			cert := fmt.Sprintf(
				"-----BEGIN CERTIFICATE-----\n%s\n-----END CERTIFICATE-----",
				t.Header["x5c"].([]interface{})[0].(string),
			)

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
		},
	)
	if err != nil {
		return nil, err
	}

	return parse, nil
}

type PaymentProfileDto struct {
	UserID         int64
	PanMasked      string
	Holder         string
	Email          *string
	Phone          *string
	UserToken      string
	RecurrentToken *string
}
