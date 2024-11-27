package biz

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	v1 "gitlab.calendaria.team/services/finance/billing/api/billing/v1"
	"gitlab.calendaria.team/services/finance/billing/ent"
	"gitlab.calendaria.team/services/finance/billing/ent/enum"
	"gitlab.calendaria.team/services/finance/billing/internal/data"
)

type AppleStoreUsecase struct {
	invoices      data.InvoicesRepo
	subscriptions data.SubscriptionsRepo
	product       data.ProductRepo
}

func NewAppleStoreUsecase(invoices data.InvoicesRepo) *AppleStoreUsecase {
	return &AppleStoreUsecase{
		invoices: invoices,
	}
}

func (uc *AppleStoreUsecase) ProcessPayload(ctx context.Context, payload data.Payload) error {
	var err error

	if payload.NotificationType == data.TYPE_SUBSCRIBED ||
		payload.NotificationType == data.TYPE_DID_RENEW ||
		payload.NotificationType == data.TYPE_OFFER_REDEEMED {
		err = uc.processSubscription(ctx, payload)
	}

	if payload.NotificationType == data.TYPE_EXPIRED ||
		payload.NotificationType == data.TYPE_DID_FAIL_TO_RENEW ||
		payload.NotificationType == data.TYPE_GRACE_PERIOD_EXPIRED ||
		payload.NotificationType == data.TYPE_REVOKE {
		err = uc.processExpired(ctx, payload)
	}

	return err
}

func (uc *AppleStoreUsecase) processSubscription(ctx context.Context, payload data.Payload) error {
	var transaction data.JWSTransaction

	parse, err := data.ParseAppleSignedBody(payload.Data.SignedTransactionInfo)
	if err != nil {
		return v1.ErrorInternal("failed to parse signed transaction info")
	}

	mapClaims, ok := parse.Claims.(jwt.MapClaims)
	if !ok {
		return v1.ErrorInvalidRequest("failed to parse signed transaction: claims is not a map")
	}

	jsonBody, err := json.Marshal(mapClaims)
	if err != nil {
		return v1.ErrorInvalidRequest("failed to parse signed transaction: %s", err.Error())
	}

	err = json.Unmarshal(jsonBody, &transaction)
	if err != nil {
		return v1.ErrorInvalidRequest("failed to parse signed transaction: %s", err.Error())
	}

	uid, err := uuid.Parse(transaction.AppAccountToken)
	if err != nil {
		return v1.ErrorInvalidRequest("failed to parse app account token: %s", err.Error())
	}

	userID := extractUserIDFromUUID(uid)
	tenantID := extractTenantIDFromUUID(uid)

	productID, err := strconv.Atoi(transaction.ProductID)
	if err != nil {
		return v1.ErrorInvalidRequest("product id is not valid: %s", err.Error())
	}

	productEnt, err := uc.product.GetProduct(ctx, int64(productID))
	if err != nil {
		return v1.ErrorDatabaseQuery("failed to get product: %s", err.Error())
	}

	subscription, err := uc.subscriptions.GetSubscriptionByOriginalAppleTransactionID(ctx,
		transaction.OriginalTransactionID, false)
	if err != nil && !ent.IsNotFound(err) {
		return v1.ErrorDatabaseQuery("failed to get subscription: %s", err.Error())
	}

	if ent.IsNotFound(err) {
		subscription, err = uc.subscriptions.CreateSubscription(ctx, userID, tenantID, productEnt.AppID,
			data.SubscriptionDto{
				UserID:    userID,
				TenantID:  tenantID,
				AppID:     productEnt.AppID,
				ProductID: productEnt.ID,
			})
		if err != nil {
			return v1.ErrorDatabaseQuery("failed to create subscription: %s", err.Error())
		}
	}

	paidAt := time.Unix(transaction.PurchaseDate/1000, 0)
	paidTill := time.Unix(transaction.ExpiresDate/1000, 0)
	_, err = uc.invoices.CreateInvoice(ctx, data.InvoiceDto{
		UserID:                  subscription.UserID,
		TenantID:                subscription.TenantID,
		AppID:                   subscription.AppID,
		ProductID:               productEnt.ID,
		Amount:                  transaction.Quantity,
		Price:                   decimal.New(transaction.Price, -3),
		Status:                  enum.Paid,
		SubscriptionID:          subscription.ID,
		PaidAt:                  &paidAt,
		PaidTill:                &paidTill,
		AppleStoreTransactionID: &transaction.OriginalTransactionID,
	})
	if err != nil {
		return err
	}

	return nil
}

func (uc *AppleStoreUsecase) processExpired(ctx context.Context, payload data.Payload) error {
	var transaction data.JWSTransaction

	parse, err := data.ParseAppleSignedBody(payload.Data.SignedTransactionInfo)
	if err != nil {
		return v1.ErrorInternal("failed to parse signed transaction info")
	}

	mapClaims, ok := parse.Claims.(jwt.MapClaims)
	if !ok {
		return v1.ErrorInvalidRequest("failed to parse signed transaction: claims is not a map")
	}

	jsonBody, err := json.Marshal(mapClaims)
	if err != nil {
		return v1.ErrorInvalidRequest("failed to parse signed transaction: %s", err.Error())
	}

	err = json.Unmarshal(jsonBody, &transaction)
	if err != nil {
		return v1.ErrorInvalidRequest("failed to parse signed transaction: %s", err.Error())
	}

	subscription, err := uc.subscriptions.GetSubscriptionByOriginalAppleTransactionID(ctx,
		transaction.OriginalTransactionID, false)
	if err != nil {
		return v1.ErrorDatabaseQuery("failed to get subscription: %s", err.Error())
	}

	if payload.NotificationType == data.TYPE_REVOKE {
		revocationDate := time.Unix(transaction.RevocationDate/1000, 0)

		err = uc.subscriptions.RevokeActiveSubscription(ctx, subscription.ID, revocationDate)
		if err != nil {
			return nil
		}
	}

	return nil
}

func extractUserIDFromUUID(uid uuid.UUID) int64 {
	reconstructedActorId := int64(
		binary.BigEndian.Uint64(
			[]byte{
				uid[8], uid[9], uid[10], uid[11], uid[12], uid[13], uid[14], uid[15],
			},
		),
	)

	return reconstructedActorId
}

func extractTenantIDFromUUID(uid uuid.UUID) int64 {
	reconstructedTenantId := int64(
		binary.BigEndian.Uint64(
			[]byte{
				uid[0], uid[1], uid[2], uid[3], uid[4], uid[5], uid[6], uid[7],
			},
		),
	)

	return reconstructedTenantId
}
