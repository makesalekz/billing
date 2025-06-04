package biz

import (
	"context"
	"fmt"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/golang-jwt/jwt/v5"

	"gitlab.calendaria.team/services/finance/billing/ent"
	"gitlab.calendaria.team/services/finance/billing/internal/data"
)

type AppleStoreReliabilityUseCase struct {
	subscriptions data.SubscriptionsRepo
	invoices      data.InvoicesRepo
	appleClient   data.AppleStoreClient
	jwtParser     data.JWTParser
	log           *log.Helper
}

func NewAppleStoreReliabilityUseCase(
	subscriptions data.SubscriptionsRepo,
	invoices data.InvoicesRepo,
	appleClient data.AppleStoreClient,
	jwtParser data.JWTParser,
	logger log.Logger,
) *AppleStoreReliabilityUseCase {
	return &AppleStoreReliabilityUseCase{
		subscriptions: subscriptions,
		invoices:      invoices,
		appleClient:   appleClient,
		jwtParser:     jwtParser,
		log:           log.NewHelper(log.With(logger, "module", "biz/apple-store-reliability")),
	}
}

// CheckMissedS2SNotifications checks for missed S2S notifications from Apple.
func (uc *AppleStoreReliabilityUseCase) CheckMissedS2SNotifications(ctx context.Context) error {
	uc.log.Info("Starting check for missed S2S notifications")

	activeSubscriptions, err := uc.subscriptions.GetActiveAppleSubscriptions(ctx)
	if err != nil {
		uc.log.Errorf("Failed to get active Apple subscriptions: %v", err)
		return fmt.Errorf("failed to get active Apple subscriptions: %w", err)
	}

	for _, subscription := range activeSubscriptions {
		if err = uc.checkSubscriptionHistory(ctx, subscription); err != nil {
			uc.log.Errorf("Failed to check history for subscription %d: %v", subscription.ID, err)
			continue
		}
	}

	return nil
}

func (uc *AppleStoreReliabilityUseCase) checkSubscriptionHistory(
	ctx context.Context, subscription *ent.Subscriptions,
) error {
	if len(subscription.Edges.Invoices) == 0 {
		uc.log.Warnf("No invoices found for subscription %d", subscription.ID)
		return nil
	}

	var lastInvoice *ent.Invoice
	for _, invoice := range subscription.Edges.Invoices {
		if invoice.OriginalAppleTransactionID != nil {
			if lastInvoice == nil || invoice.ID > lastInvoice.ID {
				lastInvoice = invoice
			}
		}
	}

	if lastInvoice == nil || lastInvoice.OriginalAppleTransactionID == nil {
		uc.log.Warnf("No original transaction ID found for subscription %d", subscription.ID)
		return nil
	}

	historyResponse, err := uc.appleClient.GetTransactionHistory(ctx, *lastInvoice.OriginalAppleTransactionID)
	if err != nil {
		return fmt.Errorf("failed to get transaction history from Apple: %w", err)
	}

	for _, signedTransaction := range historyResponse.SignedTransactions {
		if err = uc.processHistoryTransaction(ctx, signedTransaction, subscription); err != nil {
			uc.log.Errorf("Failed to process history transaction: %v", err)
			continue
		}
	}

	return nil
}

func (uc *AppleStoreReliabilityUseCase) processHistoryTransaction(
	ctx context.Context, signedTransaction string, subscription *ent.Subscriptions,
) error {
	token, err := uc.jwtParser.ParseAppleSignedBody(signedTransaction)
	if err != nil {
		return fmt.Errorf("failed to parse signed transaction: %w", err)
	}

	mapClaims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return fmt.Errorf("invalid token claims format")
	}

	transactionID, ok := mapClaims["transactionId"].(string)
	if !ok {
		return fmt.Errorf("transaction ID not found in claims")
	}

	existingInvoices, err := uc.invoices.ListInvoices(
		ctx, data.InvoiceFilter{
			SubscriptionID: subscription.ID,
		}, nil,
	)
	if err != nil {
		return fmt.Errorf("failed to check existing invoices: %w", err)
	}

	for _, invoice := range existingInvoices {
		if invoice.ExternalTransactionID != nil && *invoice.ExternalTransactionID == transactionID {
			// Транзакция уже существует, пропускаем
			return nil
		}
	}

	uc.log.Warnf(
		"Found missing transaction %s for subscription %d, potential missed S2S notification", transactionID,
		subscription.ID,
	)

	// TODO: Здесь можно добавить логику для обработки пропущенной транзакции

	return nil
}
