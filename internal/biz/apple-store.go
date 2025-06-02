package biz

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"log"
	"math/rand"
	"os"
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

const (
	milliUnits = 1000
	centiUnits = 100
	deciUnits  = 10
)

type AppleStoreUsecase struct {
	invoices      data.InvoicesRepo
	subscriptions data.SubscriptionsRepo
	product       data.ProductRepo
	jwtParser     data.JWTParser
}

func NewAppleStoreUsecase(
	invoices data.InvoicesRepo,
	subscriptions data.SubscriptionsRepo,
	product data.ProductRepo,
	jwtParser data.JWTParser,
) *AppleStoreUsecase {
	return &AppleStoreUsecase{
		invoices:      invoices,
		subscriptions: subscriptions,
		product:       product,
		jwtParser:     jwtParser,
	}
}

func (uc *AppleStoreUsecase) ProcessPayload(ctx context.Context, payload data.Payload) error {
	var err error

	if payload.NotificationType == data.TypeTest {
		if os.Getenv("DEBUG") == "true" {
			r := rand.New(rand.NewSource(time.Now().UnixNano()))
			notificationTypes := []data.NotificationType{
				data.TypeSubscribed,
				data.TypeDidRenew,
				data.TypeOfferRedeemed,
				data.TypeExpired,
				data.TypeDidFailToRenew,
				data.TypeRevoke,
			}
			randomType := notificationTypes[r.Intn(len(notificationTypes))]

			log.Printf("DEBUG: Received TEST notification, simulating %s", randomType)

			simulatedPayload := payload
			simulatedPayload.NotificationType = randomType

			return uc.ProcessPayload(ctx, simulatedPayload)
		}

		log.Printf("Received test notification")
		return nil
	}

	if payload.NotificationType == data.TypeSubscribed ||
		payload.NotificationType == data.TypeDidRenew ||
		payload.NotificationType == data.TypeOfferRedeemed {
		err = uc.processSubscription(ctx, payload)
	}

	if payload.NotificationType == data.TypeExpired ||
		payload.NotificationType == data.TypeDidFailToRenew ||
		payload.NotificationType == data.TypeGracePeriodExpired ||
		payload.NotificationType == data.TypeRevoke {
		err = uc.processExpired(ctx, payload)
	}

	return err
}

func (uc *AppleStoreUsecase) processSubscription(ctx context.Context, payload data.Payload) error {
	var transaction data.JWSTransaction

	decodedTransaction, err := uc.jwtParser.ParseAppleSignedBody(payload.Data.SignedTransactionInfo)
	if err != nil {
		return v1.ErrorInternal("failed to parse signed transaction info")
	}

	mapClaims, ok := decodedTransaction.Claims.(jwt.MapClaims)
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

	subscription, err := uc.subscriptions.GetSubscriptionByOriginalAppleTransactionID(
		ctx,
		transaction.OriginalTransactionID, false,
	)
	if err != nil && !ent.IsNotFound(err) {
		return v1.ErrorDatabaseQuery("failed to get subscription: %s", err.Error())
	}

	if ent.IsNotFound(err) {
		subscription, err = uc.subscriptions.CreateSubscription(
			ctx, userID, tenantID, productEnt.AppID,
			data.SubscriptionDto{
				UserID:    userID,
				TenantID:  tenantID,
				AppID:     productEnt.AppID,
				ProductID: productEnt.ID,
			},
		)
		if err != nil {
			return v1.ErrorDatabaseQuery("failed to create subscription: %s", err.Error())
		}
	}

	paidAt := time.Unix(transaction.PurchaseDate/milliUnits, 0)
	paidTill := time.Unix(transaction.ExpiresDate/milliUnits, 0)
	_, err = uc.invoices.CreateInvoice(
		ctx, data.InvoiceDto{
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
			IsTrial:                 transaction.OfferDiscountType == "FREE_TRIAL",
		},
	)
	if err != nil {
		return err
	}

	return nil
}

func (uc *AppleStoreUsecase) processExpired(ctx context.Context, payload data.Payload) error {
	var transaction data.JWSTransaction

	decodedTransaction, err := uc.jwtParser.ParseAppleSignedBody(payload.Data.SignedTransactionInfo)
	if err != nil {
		return v1.ErrorInternal("failed to parse signed transaction info")
	}

	mapClaims, ok := decodedTransaction.Claims.(jwt.MapClaims)
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

	subscription, err := uc.subscriptions.GetSubscriptionByOriginalAppleTransactionID(
		ctx,
		transaction.OriginalTransactionID, false,
	)
	if err != nil {
		return v1.ErrorDatabaseQuery("failed to get subscription: %s", err.Error())
	}

	if payload.NotificationType == data.TypeRevoke {
		revocationDate := time.Unix(transaction.RevocationDate/milliUnits, 0)

		err = uc.subscriptions.RevokeActiveSubscription(ctx, subscription.ID, revocationDate)
		if err != nil {
			return err
		}
	}

	return nil
}

func extractUserIDFromUUID(uid uuid.UUID) int64 {
	reconstructedActorID := int64(

		binary.BigEndian.Uint64(
			[]byte{
				uid[8], uid[9], uid[10], uid[11], uid[12], uid[13], uid[14], uid[15],
			},
		),
	)

	return reconstructedActorID
}

func extractTenantIDFromUUID(uid uuid.UUID) int64 {
	reconstructedTenantID := int64(

		binary.BigEndian.Uint64(
			[]byte{
				uid[0], uid[1], uid[2], uid[3], uid[4], uid[5], uid[6], uid[7],
			},
		),
	)

	return reconstructedTenantID
}
