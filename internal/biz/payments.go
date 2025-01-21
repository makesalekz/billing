package biz

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/go-kratos/kratos/v2/log"

	v1 "gitlab.calendaria.team/services/finance/billing/api/billing/v1"
	"gitlab.calendaria.team/services/finance/billing/ent"
	"gitlab.calendaria.team/services/finance/billing/ent/enum"
	"gitlab.calendaria.team/services/finance/billing/internal/data"
	"gitlab.calendaria.team/services/finance/onevisionpay"
)

const (
	DefaultPriceForCardLink = 0
	PmsAppID                = "pms"
)

type PaymentUseCase struct {
	log                    *log.Helper
	paymentClient          *data.OvpClient
	invoicesRepo           data.InvoicesRepo
	productRepo            data.ProductRepo
	subscriptionRepo       data.SubscriptionsRepo
	paymentProfileRepo     data.PaymentProfileRepo
	productReservationRepo data.ProductReservationRepo
	invoiceManager         *InvoicesManager
}

func NewPaymentUsecase(
	logger log.Logger,
	paymentClient *data.OvpClient,
	invoicesRepo data.InvoicesRepo,
	productRepo data.ProductRepo,
	subscriptionRepo data.SubscriptionsRepo,
	paymentProfileRepo data.PaymentProfileRepo,
	productReservationRepo data.ProductReservationRepo,
	invoiceManager *InvoicesManager,
) (*PaymentUseCase, error) {
	return &PaymentUseCase{
		log:                    log.NewHelper(log.With(logger, "module", "usecase/payment")),
		paymentClient:          paymentClient,
		invoicesRepo:           invoicesRepo,
		productRepo:            productRepo,
		subscriptionRepo:       subscriptionRepo,
		paymentProfileRepo:     paymentProfileRepo,
		productReservationRepo: productReservationRepo,
		invoiceManager:         invoiceManager,
	}, nil
}

func (uc *PaymentUseCase) CreatePayment(
	ctx context.Context, tenantID, actorID, productID int64, appID string, amount int64,
) (int64, string, error) {
	if uc.paymentClient == nil {
		return 0, "", v1.ErrorInternal("payment client is not initialized")
	}

	invoiceDTO := data.InvoiceDto{
		TenantID:  tenantID,
		UserID:    actorID,
		AppID:     appID,
		ProductID: productID,
		Status:    enum.Created,
		Amount:    amount,
	}

	invoice, product, rollback, err := uc.invoiceManager.CreateInvoice(ctx, tenantID, actorID, invoiceDTO, productID)

	if err != nil {
		if rollback != nil {
			rollback()
		}
		return 0, "", err
	}

	payment, err := uc.paymentClient.CreatePayment(
		actorID, invoice, product,
	)

	if err != nil {
		uc.log.Errorf("Failed to create payment: %v", err)
		rollback()

		return 0, "", v1.ErrorInvalidRequest("failed to create payment %v", err)
	}

	paymentIDStr := strconv.FormatInt(payment.PaymentID, 10)
	updatedInvoice, err := uc.invoicesRepo.UpdateInvoice(
		ctx, invoice, data.InvoiceDto{
			OneVisionTransactionID: &paymentIDStr,
		},
	)
	if err != nil {
		return 0, "", v1.ErrorDatabaseQuery("failed to update invoice: %v", err)
	}

	return updatedInvoice.ID, payment.PaymentPageURL, nil
}

func (uc *PaymentUseCase) HandlePaymentCallback(
	ctx context.Context, req *v1.PaymentCallbackRequest,
) {
	uc.log.Infof("Handling payment callback: %v", req)

	if uc.paymentClient == nil {
		uc.log.Errorf("Payment client is not initialized")
		return
	}

	if !uc.paymentClient.VerifySignature(req.GetData(), req.GetSign()) {
		uc.log.Errorf("Invalid signature: %v", req.GetSign())
		return
	}

	statusResponse, invoiceID, err := uc.paymentClient.ParsePayload(req.GetData())
	if err != nil {
		uc.log.Errorf("Failed to parse payload: %v", err)
		return
	}

	invoice, err := uc.invoicesRepo.GetInvoiceByID(ctx, invoiceID)
	if err != nil {
		if ent.IsNotFound(err) {
			uc.log.Errorf("Invoice not found: %v", invoiceID)
			return
		}
		uc.log.Errorf("Failed to get invoice: %v", err)
		return
	}

	err = uc.processPaymentStatus(ctx, invoice, statusResponse)

	if err != nil {
		uc.log.Errorf("Failed to update invoice status: %v", err)
		return
	}

	uc.log.Infof("Callback processed successfully for invoice: %v", invoiceID)
}

func (uc *PaymentUseCase) processPaymentStatus(
	ctx context.Context, invoice *ent.Invoice, statusResponse *onevisionpay.StatusResponse,
) error {
	switch statusResponse.PaymentStatus {
	case onevisionpay.Created:
		return uc.handleCreatedStatus(invoice)
	case onevisionpay.Refunded:
		return uc.handleRefundedStatus(ctx, invoice, statusResponse)
	case onevisionpay.Clearing, onevisionpay.Withdraw:
		return uc.handleCompletedStatus(ctx, invoice, statusResponse)
	case onevisionpay.Canceled, onevisionpay.Error, onevisionpay.Chargeback:
		return uc.handleFailedOrCanceledStatus(ctx, invoice, statusResponse)
	case onevisionpay.PartialRefund:
		return uc.handleRefundedStatus(ctx, invoice, statusResponse)
	case onevisionpay.Processing, onevisionpay.NeedApprove, onevisionpay.Hold,
		onevisionpay.Refill, onevisionpay.Process, onevisionpay.PartialClearing:
		return uc.handleNonWidgetStatus(statusResponse, invoice)
	default:
		uc.log.Warnf("Unknown payment status: %v for invoice: %v", statusResponse.PaymentStatus, invoice.ID)
		return nil
	}
}

func (uc *PaymentUseCase) handleCreatedStatus(invoice *ent.Invoice) error {
	uc.log.Infof("Payment created for invoice: %v", invoice.ID)
	return nil
}

func (uc *PaymentUseCase) handleRefundedStatus(
	ctx context.Context, invoice *ent.Invoice,
	payment *onevisionpay.StatusResponse,
) error {
	uc.log.Infof("Processing refund for invoice: %v", invoice.ID)

	transactionID := strconv.FormatInt(payment.PaymentID, 10)

	if payment.Amount == invoice.Amount {
		return uc.processFullRefund(ctx, invoice, transactionID)
	}

	return uc.processPartialRefund(ctx, invoice, payment, transactionID)
}

func (uc *PaymentUseCase) processFullRefund(
	ctx context.Context, invoice *ent.Invoice, transactionID string,
) error {
	uc.log.Infof("Processing full refund for invoice: %v", invoice.ID)

	if invoice.SubscriptionID != nil {
		if err := uc.subscriptionRepo.RevokeActiveSubscription(ctx, *invoice.SubscriptionID, time.Now()); err != nil {
			uc.log.Errorf(
				"Failed to revoke subscription %v for invoice %v: %v", *invoice.SubscriptionID, invoice.ID, err,
			)
			return err
		}
		uc.log.Infof("Subscription %v successfully revoked for invoice %v", *invoice.SubscriptionID, invoice.ID)
	}

	revokedAt := time.Now()
	isRevoked := true

	_, err := uc.invoicesRepo.UpdateInvoice(
		ctx, invoice, data.InvoiceDto{
			Status:                 enum.CanceledByUser,
			OneVisionTransactionID: &transactionID,
			RevokedAt:              &revokedAt,
			IsRevoked:              &isRevoked,
		},
	)
	if err != nil {
		uc.log.Errorf("Failed to update invoice %d status to %v: %v", invoice.ID, enum.CanceledByUser, err)
		return err
	}

	return nil
}

func (uc *PaymentUseCase) processPartialRefund(
	ctx context.Context, invoice *ent.Invoice, payment *onevisionpay.StatusResponse, transactionID string,
) error {
	uc.log.Infof("Processing partial refund for invoice: %v", invoice.ID)

	if invoice.PaidAt != nil && invoice.PaidTill != nil {
		totalDuration := invoice.PaidTill.Sub(*invoice.PaidAt)
		remainingDuration := time.Duration(float64(totalDuration) * float64(payment.Amount) / float64(invoice.Amount))
		newRevokedAt := invoice.PaidAt.Add(remainingDuration)
		isRevoked := true

		_, err := uc.invoicesRepo.UpdateInvoice(
			ctx, invoice, data.InvoiceDto{
				Status:                 enum.CanceledByUser,
				OneVisionTransactionID: &transactionID,
				RevokedAt:              &newRevokedAt,
				IsRevoked:              &isRevoked,
			},
		)
		if err != nil {
			uc.log.Errorf("Failed to update invoice %d with partial refund: %v", invoice.ID, err)
			return err
		}
	} else {
		uc.log.Warnf("Invoice %v does not have valid PaidAt or PaidTill for partial refund calculation", invoice.ID)
	}

	return nil
}

func (uc *PaymentUseCase) handleCompletedStatus(
	ctx context.Context, invoice *ent.Invoice, paymentStatus *onevisionpay.StatusResponse,
) error {
	uc.log.Infof("Payment completed for invoice: %v", invoice.ID)

	if invoice.Status == enum.Paid && invoice.SubscriptionID != nil {
		uc.log.Infof("Invoice %v already paid", invoice.ID)
		return nil
	}

	recurrentProfile, err := uc.saveRecurrentProfile(ctx, invoice.UserID, paymentStatus)
	if err != nil {
		uc.log.Errorf("Failed to save payment profile for user %v: %v", invoice.UserID, err)
		return err
	}

	uc.log.Infof("Payment profile successfully saved for user %v", invoice.UserID)

	paidAt := time.Now()

	updateInvoiceDto := data.InvoiceDto{
		Status:             enum.Paid,
		RecurrentProfileID: &recurrentProfile.ID,
		PaidAt:             &paidAt,
	}

	if invoice.SubscriptionID == nil {
		uc.log.Infof(
			"Activating subscription for invoice: %v, subscription ID: %v", invoice.ID, *invoice.SubscriptionID,
		)

		newSubscription, createSubErr := uc.subscriptionRepo.CreateSubscription(
			ctx, invoice.UserID, invoice.TenantID, invoice.AppID, data.SubscriptionDto{
				TenantID:  invoice.TenantID,
				UserID:    invoice.UserID,
				AppID:     invoice.AppID,
				ProductID: invoice.ProductID,
			},
		)

		if createSubErr != nil {
			uc.log.Errorf("Failed to create subscription for invoice %v: %v", invoice.ID, createSubErr)
			return createSubErr
		}

		updateInvoiceDto.SubscriptionID = newSubscription.ID
	} else {
		uc.log.Infof(
			"User alrady had subscription for invoice: %v, subscription ID: %v", invoice.ID, *invoice.SubscriptionID,
		)
	}

	_, err = uc.invoicesRepo.UpdateInvoice(ctx, invoice, updateInvoiceDto)
	if err != nil {
		uc.log.Errorf("Failed to update invoice status to Paid for invoice %v: %v", invoice.ID, err)
		return err
	}

	err = uc.productReservationRepo.UpdateReservationStatusByInvoiceID(ctx, invoice.ID, enum.Completed)
	if err != nil {
		uc.log.Errorf("Failed to update reservation status for invoice %v: %v", invoice.ID, err)
		return err
	}

	uc.log.Infof("Invoice %v successfully updated with status Paid", invoice.ID)

	return nil
}

func (uc *PaymentUseCase) handleFailedOrCanceledStatus(
	ctx context.Context, invoice *ent.Invoice,
	statusResponse *onevisionpay.StatusResponse,
) error {
	uc.log.Infof("Payment failed or canceled for invoice: %v", invoice.ID)

	now := time.Now()
	isRevoked := true
	transactionID := strconv.FormatInt(statusResponse.PaymentID, 10)

	updateDto := data.InvoiceDto{
		Status:                 enum.CanceledByUser,
		RevokedAt:              &now,
		IsRevoked:              &isRevoked,
		OneVisionTransactionID: &transactionID,
	}

	_, err := uc.invoicesRepo.UpdateInvoice(ctx, invoice, updateDto)
	if err != nil {
		uc.log.Errorf("Failed to update invoice status to Canceled for invoice %v: %v", invoice.ID, err)
		return err
	}

	uc.log.Infof("Invoice %v successfully updated with status Canceled", invoice.ID)

	err = uc.productReservationRepo.CancelReservationByInvoiceID(ctx, invoice.ID)
	if err != nil {
		uc.log.Errorf("Failed to update reservation status for invoice %v: %v", invoice.ID, err)
		return err
	}

	return nil
}

func (uc *PaymentUseCase) handleNonWidgetStatus(
	status *onevisionpay.StatusResponse, invoice *ent.Invoice,
) error {
	uc.log.Infof("Unsupported payment status: %v for invoice: %v", status.PaymentStatus, invoice.ID)
	return nil
}

func (uc *PaymentUseCase) CancelSubscription(
	ctx context.Context, subscriptionID int64,
) error {
	err := uc.subscriptionRepo.RevokeActiveSubscription(ctx, subscriptionID, time.Now())
	if err != nil {
		if ent.IsNotFound(err) {
			return v1.ErrorNotFound("subscription not found")
		}
		return v1.ErrorDatabaseQuery("failed to cancel subscription: %v", err)
	}
	return nil
}

func (uc *PaymentUseCase) checkSubscriptionStatus(
	ctx context.Context, tenantID, actorID, productID int64,
) (bool, bool, error) {
	// fetch all paid invoices for the user in the app
	filter := data.InvoiceFilter{
		TenantID:  tenantID,
		UserID:    actorID,
		Status:    enum.Paid,
		Paid:      true,
		ProductID: productID,
	}

	invoices, err := uc.invoicesRepo.ListInvoices(ctx, filter, nil)
	if err != nil {
		uc.log.Errorf("Failed to list invoices: %v", err)
		return false, false, err
	}

	hasActive := false

	for _, invoice := range invoices {
		if invoice.PaidTill != nil && !invoice.IsRevoked {
			if invoice.PaidTill.After(time.Now()) {
				hasActive = true
			}
		}
	}

	isFirst := len(invoices) == 0

	return hasActive, isFirst, nil
}

func (uc *PaymentUseCase) saveRecurrentProfile(
	ctx context.Context, userID int64, statusResponse *onevisionpay.StatusResponse,
) (*ent.PaymentProfile, error) {
	panMasked := statusResponse.PayerInfo.PanMasked
	holder := statusResponse.PayerInfo.Holder
	email := statusResponse.PayerInfo.Email
	phone := statusResponse.PayerInfo.Phone
	userToken := statusResponse.PayerInfo.UserToken
	recurrentToken := statusResponse.RecurrentToken

	existingProfile, err := uc.paymentProfileRepo.GetProfileByUserID(ctx, userID)
	if err != nil && !ent.IsNotFound(err) {
		return nil, fmt.Errorf("failed to check existing payment profile: %w", err)
	}

	if existingProfile == nil {
		uc.log.Infof("Creating new payment profile for user %v.", userID)

		existingProfile, err = uc.paymentProfileRepo.CreateProfile(
			ctx, data.PaymentProfileDto{
				UserID:         userID,
				PanMasked:      panMasked,
				Holder:         holder,
				UserToken:      userToken,
				Email:          &email,
				Phone:          &phone,
				RecurrentToken: &recurrentToken,
			},
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create payment profile: %w", err)
		}
	} else {
		uc.log.Infof("Payment profile already exists for user %v. Updating profile.", userID)

		err = uc.paymentProfileRepo.UpdateProfile(
			ctx, existingProfile.ID, data.PaymentProfileDto{
				Email:          &email,
				Phone:          &phone,
				RecurrentToken: &recurrentToken,
			},
		)
		if err != nil {
			return nil, fmt.Errorf("failed to update payment profile: %w", err)
		}
	}

	return existingProfile, nil
}

// ProcessExpiredPayments processes expired payments for recurrent profiles.
func (uc *PaymentUseCase) ProcessExpiredPayments(ctx context.Context) {
	uc.log.Info("Processing expired payments for recurrent profiles")

	now := time.Now().Add(time.Hour) // give one hour to renew subscription

	expiredPayments, err := uc.invoicesRepo.GetInvoicesToExpire(ctx, &now)
	if err != nil {
		uc.log.Errorf("failed to list invoices: %s", err.Error())
	}

	for _, payment := range expiredPayments {
		uc.log.Infof("Processing expired payment for user %v, invoice ID %v", payment.UserID, payment.ID)

		uc.createRecurrentPayment(ctx, payment)
	}
}

// createRecurrentPayment creates a recurrent payment for the given invoice.
func (uc *PaymentUseCase) createRecurrentPayment(ctx context.Context, invoice *ent.Invoice) {
	if invoice.AppID != PmsAppID {
		uc.log.Infof("Skipping recurrent payment for app %v", invoice.AppID)
		return
	}

	if uc.paymentClient == nil {
		uc.log.Error("Payment client is not initialized")
		return
	}

	if invoice.PaymentProfileID == nil {
		uc.log.Errorf("Payment profile not found for invoice %v", invoice.ID)
		return
	}
	if invoice.Edges.PaymentProfile == nil {
		uc.log.Errorf("Payment profile not found for invoice %v", invoice.ID)
		return
	}

	newInvoiceDTO := data.InvoiceDto{
		TenantID:           invoice.TenantID,
		UserID:             invoice.UserID,
		AppID:              invoice.AppID,
		ProductID:          invoice.ProductID,
		Status:             enum.Created,
		Amount:             invoice.Amount,
		RecurrentProfileID: invoice.PaymentProfileID,
	}

	newInvoice, product, rollback, err := uc.invoiceManager.CreateInvoice(
		ctx, invoice.TenantID, invoice.UserID,
		newInvoiceDTO,
		invoice.ProductID,
	)

	if err != nil {
		if rollback != nil {
			rollback()
		}
		uc.log.Errorf("Failed to create new invoice for recurrent payment: %v", err)
		return
	}

	uc.log.Infof("New invoice created for recurrent payment, invoice ID: %v", newInvoice.ID)

	response, transactionID, err := uc.paymentClient.RecurrentPayment(newInvoice, product, invoice.Edges.PaymentProfile)
	if err != nil {
		uc.log.Errorf("Failed to create recurrent payment: %v", err)
		return
	}

	_, err = uc.invoicesRepo.UpdateInvoice(
		ctx, invoice, data.InvoiceDto{
			OneVisionTransactionID: transactionID,
		},
	)
	if err != nil {
		uc.log.Errorf("Failed to update invoice %d status to %v: %v", invoice.ID, enum.Created, err)
	}

	uc.log.Infof("Recurrent payment successfully created, payment ID: %v", response.PaymentID)
}

func (uc *PaymentUseCase) isProductAvailable(product *ent.Product, amount int64) error {
	if !product.IsActive {
		return v1.ErrorInvalidRequest("product is not active")
	}

	if product.IsLimited {
		if product.LimitedTill != nil && time.Now().After(*product.LimitedTill) {
			return v1.ErrorInvalidRequest("product is not available")
		}

		if product.Left == 0 || product.Left < amount {
			return v1.ErrorInvalidRequest("Product amount ")
		}
	}

	if product.IsExpiring && product.ExpiringTime != nil && time.Now().After(*product.ExpiringTime) {
		return v1.ErrorInvalidRequest("product is not available")
	}

	if product.IsUnique && product.UniqueLimit < amount {
		return v1.ErrorInvalidRequest("product is not available")
	}

	return nil
}

func (uc *PaymentUseCase) reserveProduct(
	ctx context.Context, product *ent.Product, invoice *ent.Invoice, amount int64,
) error {
	if product.IsLimited {
		_, err := uc.productReservationRepo.CreateReservation(
			ctx, data.ProductReservationDto{
				ProductID:           product.ID,
				InvoiceID:           invoice.ID,
				UserID:              invoice.UserID,
				ReservationQuantity: amount,
				Status:              enum.Pending,
			},
		)

		if err != nil {
			return err
		}
	}
	return nil
}

func (uc *PaymentUseCase) CancelReservations(ctx context.Context) {
	uc.log.Info("Processing expired reservations")
	err := uc.productReservationRepo.ProcessExpiredReservations(ctx)
	if err != nil {
		uc.log.Errorf("Failed to process expired reservations: %v", err)
	}
	uc.log.Info("Expired reservations processed")
}
