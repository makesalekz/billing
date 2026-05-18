package biz

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/go-kratos/kratos/v2/log"

	v1 "github.com/makesalekz/billing/api/billing/v1"
	"github.com/makesalekz/billing/ent"
	"github.com/makesalekz/billing/ent/enum"
	"github.com/makesalekz/billing/internal/data"
	utils_v1 "github.com/makesalekz/utils/api/utils/v1"
)

const (
	DefaultPriceForCardLink = 0
	PmsAppID                = "pms"

	// TTP webhook statuses
	TtpStatusAuthorized = "Authorized"
	TtpStatusCompleted  = "Completed"
	TtpStatusDeclined   = "Declined"
)

type PaymentUseCase struct {
	log                    *log.Helper
	paymentClient          data.PaymentClient
	invoicesRepo           data.InvoicesRepo
	productRepo            data.ProductRepo
	subscriptionRepo       data.SubscriptionsRepo
	paymentProfileRepo     data.PaymentProfileRepo
	productReservationRepo data.ProductReservationRepo
	invoiceManager         *InvoicesManager
}

func NewPaymentUsecase(
	logger log.Logger,
	paymentClient data.PaymentClient,
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

// CreatePayment processes a card payment via TipTopPay cryptogram.
func (uc *PaymentUseCase) CreatePayment(
	ctx context.Context,
	tenantID, actorID, productID int64,
	appID string,
	amount int64,
	cryptogram, ipAddress, name, email string,
) (*v1.CreatePaymentResponse, error) {
	if uc.paymentClient == nil {
		return nil, v1.ErrorInternal("payment client is not initialized")
	}

	invoiceDTO := data.InvoiceDto{
		TenantID:  tenantID,
		UserID:    actorID,
		AppID:     appID,
		ProductID: productID,
		Status:    enum.Created,
		Amount:    amount,
	}

	invoice, product, rollback, err := uc.invoiceManager.CreateInvoice(ctx, invoiceDTO)
	if err != nil {
		if rollback != nil {
			rollback()
		}
		return nil, err
	}

	chargeReq := data.TtpChargeRequest{
		Amount:               product.Price.InexactFloat64() * float64(amount),
		Currency:             product.Currency,
		CardCryptogramPacket: cryptogram,
		IpAddress:            ipAddress,
		Name:                 name,
		InvoiceId:            strconv.FormatInt(invoice.ID, 10),
		Description:          product.Description,
		AccountId:            strconv.FormatInt(actorID, 10),
		Email:                email,
	}

	ctx = data.WithRequestID(ctx, strconv.FormatInt(invoice.ID, 10))
	resp, err := uc.paymentClient.Charge(ctx, chargeReq)
	if err != nil {
		uc.log.Errorf("Failed to charge via TTP: %v", err)
		rollback()
		return nil, v1.ErrorInvalidRequest("failed to create payment: %v", err)
	}

	// Save transaction ID immediately
	txID := strconv.FormatInt(resp.Model.TransactionId, 10)
	_, err = uc.invoicesRepo.UpdateInvoice(ctx, invoice, data.InvoiceDto{
		TtpTransactionID: &txID,
	})
	if err != nil {
		uc.log.Errorf("Failed to save transaction ID for invoice %v: %v", invoice.ID, err)
	}

	// 3DS required — frontend redirects to AcsUrl
	if !resp.Success && resp.Model.AcsUrl != "" {
		return &v1.CreatePaymentResponse{
			InvoiceId:     invoice.ID,
			Success:       false,
			TransactionId: &resp.Model.TransactionId,
			PaReq:         &resp.Model.PaReq,
			AcsUrl:        &resp.Model.AcsUrl,
		}, nil
	}

	// Immediate success — payment confirmed, webhook Pay will handle the rest
	// (subscription creation, RBAC activation, etc.)
	if resp.Success {
		return &v1.CreatePaymentResponse{
			InvoiceId: invoice.ID,
			Success:   true,
		}, nil
	}

	// Declined
	uc.log.Errorf("Payment declined for invoice %v: %s (code %d)", invoice.ID, resp.Message, resp.Model.ReasonCode)
	rollback()
	return nil, v1.ErrorInvalidRequest("payment declined: %s", resp.Message)
}

// Complete3DS completes a 3D Secure payment.
func (uc *PaymentUseCase) Complete3DS(
	ctx context.Context, transactionID int64, paRes string,
) (*v1.Complete3DSResponse, error) {
	if uc.paymentClient == nil {
		return nil, v1.ErrorInternal("payment client is not initialized")
	}

	resp, err := uc.paymentClient.Post3ds(ctx, data.TtpPost3dsRequest{
		TransactionId: transactionID,
		PaRes:         paRes,
	})
	if err != nil {
		return nil, v1.ErrorInternal("3DS completion failed: %v", err)
	}

	txID := strconv.FormatInt(transactionID, 10)
	invoice, err := uc.invoicesRepo.FindByExternalTransactionID(ctx, txID)
	if err != nil {
		return nil, v1.ErrorNotFound("invoice not found for transaction %d", transactionID)
	}

	// 3DS success — webhook Pay will handle subscription/RBAC
	if resp.Success {
		return &v1.Complete3DSResponse{
			Success:   true,
			InvoiceId: &invoice.ID,
		}, nil
	}

	// 3DS failed
	uc.handleFailedPayment(ctx, invoice, txID)
	return &v1.Complete3DSResponse{Success: false}, nil
}

// handleCompletedPayment processes a successful payment.
func (uc *PaymentUseCase) handleCompletedPayment(
	ctx context.Context,
	invoice *ent.Invoice,
	product *ent.Product,
	token, panMasked, holder, email string,
) error {
	if invoice.Status == enum.Paid && invoice.SubscriptionID != nil {
		uc.log.Infof("Invoice %v already paid", invoice.ID)
		return nil
	}

	// Save payment profile (card token)
	profile, err := uc.savePaymentProfile(ctx, invoice.UserID, token, panMasked, holder, email)
	if err != nil {
		uc.log.Errorf("Failed to save payment profile for user %v: %v", invoice.UserID, err)
		return err
	}

	paidAt := time.Now()
	paidTill := paidAt.Add(calculateDuration(product.ProductPeriod))

	updateDto := data.InvoiceDto{
		Status:             enum.Paid,
		PaidAt:             &paidAt,
		PaidTill:           &paidTill,
		RecurrentProfileID: &profile.ID,
	}

	if invoice.SubscriptionID == nil {
		uc.log.Infof("Creating subscription for invoice: %v", invoice.ID)

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

		updateDto.SubscriptionID = newSubscription.ID
	}

	_, err = uc.invoicesRepo.UpdateInvoice(ctx, invoice, updateDto)
	if err != nil {
		uc.log.Errorf("Failed to update invoice %v: %v", invoice.ID, err)
		return err
	}

	// Update reservation
	err = uc.productReservationRepo.UpdateReservationStatusByInvoiceID(ctx, invoice.ID, enum.Completed)
	if err != nil {
		uc.log.Errorf("Failed to update reservation for invoice %v: %v", invoice.ID, err)
	}

	if product.PaymentModel == enum.Recurrent && token != "" {
		uc.createTtpSubscription(ctx, invoice, product, token, paidTill)
	}

	uc.log.Infof("Invoice %v successfully paid", invoice.ID)
	return nil
}

// createTtpSubscription creates a recurring subscription on TTP side.
func (uc *PaymentUseCase) createTtpSubscription(
	ctx context.Context,
	invoice *ent.Invoice,
	product *ent.Product,
	token string,
	startDate time.Time,
) {
	period, interval := mapProductPeriodToTtp(product.ProductPeriod)
	if interval == "" {
		uc.log.Warnf("Cannot create TTP subscription: unsupported period %v", product.ProductPeriod)
		return
	}

	ttpSub, err := uc.paymentClient.CreateSubscription(ctx, data.TtpRecurrentCreateRequest{
		AccountId:   strconv.FormatInt(invoice.UserID, 10),
		Amount:      product.Price.InexactFloat64(),
		Currency:    product.Currency,
		Token:       token,
		Period:      period,
		Interval:    interval,
		StartDate:   startDate.Format(time.RFC3339),
		Description: product.Description,
	})
	if err != nil {
		uc.log.Errorf("Failed to create TTP subscription for invoice %v: %v", invoice.ID, err)
		return
	}

	if ttpSub.Success {
		_, err = uc.invoicesRepo.UpdateInvoice(ctx, invoice, data.InvoiceDto{
			TtpSubscriptionID: &ttpSub.Model.SubscriptionId,
		})
		if err != nil {
			uc.log.Errorf("Failed to save TTP subscription ID for invoice %v: %v", invoice.ID, err)
		}
		uc.log.Infof("TTP subscription %s created for invoice %v", ttpSub.Model.SubscriptionId, invoice.ID)
	} else {
		uc.log.Errorf("TTP subscription creation failed for invoice %v: %s", invoice.ID, ttpSub.Message)
	}
}

// handleFailedPayment marks an invoice as failed.
func (uc *PaymentUseCase) handleFailedPayment(ctx context.Context, invoice *ent.Invoice, transactionID string) {
	now := time.Now()
	isRevoked := true
	_, err := uc.invoicesRepo.UpdateInvoice(ctx, invoice, data.InvoiceDto{
		Status:           enum.CanceledByUser,
		RevokedAt:        &now,
		IsRevoked:        &isRevoked,
		TtpTransactionID: &transactionID,
	})
	if err != nil {
		uc.log.Errorf("Failed to update invoice %v: %v", invoice.ID, err)
	}

	err = uc.productReservationRepo.CancelReservationByInvoiceID(ctx, invoice.ID)
	if err != nil {
		uc.log.Errorf("Failed to cancel reservation for invoice %v: %v", invoice.ID, err)
	}
}

// HandleCheckWebhook validates a payment before TTP processes it.
func (uc *PaymentUseCase) HandleCheckWebhook(ctx context.Context, p *v1.WebhookPayload) (int, string) {
	uc.log.Infof("Check webhook: InvoiceId=%s, Amount=%.2f", p.GetInvoiceId(), p.GetAmount())

	invoiceID, err := strconv.ParseInt(p.GetInvoiceId(), 10, 64)
	if err != nil {
		return 100, "Invalid InvoiceId"
	}

	invoice, err := uc.invoicesRepo.GetInvoiceByID(ctx, invoiceID)
	if err != nil {
		uc.log.Errorf("Invoice not found for Check: %d", invoiceID)
		return 100, "Invoice not found"
	}

	if invoice.Status != enum.Created {
		uc.log.Warnf("Check for non-CREATED invoice %d (status=%s)", invoiceID, invoice.Status)
		return 100, "Invoice already processed"
	}

	return 0, "OK"
}

// HandlePaymentWebhook processes TTP Pay/Fail webhook.
func (uc *PaymentUseCase) HandlePaymentWebhook(ctx context.Context, p *v1.WebhookPayload) (int, string) {
	uc.log.Infof("Payment webhook: TransactionId=%d, Status=%s, InvoiceId=%s",
		p.GetTransactionId(), p.GetStatus(), p.GetInvoiceId())

	invoiceID, err := strconv.ParseInt(p.GetInvoiceId(), 10, 64)
	if err != nil {
		return 13, "Invalid InvoiceId"
	}

	invoice, err := uc.invoicesRepo.GetInvoiceByID(ctx, invoiceID)
	if err != nil {
		if ent.IsNotFound(err) {
			return 13, "Invoice not found"
		}
		return 13, "Internal error"
	}

	switch p.GetStatus() {
	case TtpStatusAuthorized, TtpStatusCompleted:
		product, prodErr := uc.productRepo.GetProduct(ctx, invoice.ProductID)
		if prodErr != nil {
			uc.log.Errorf("Failed to get product for invoice %v: %v", invoice.ID, prodErr)
			return 13, "Product not found"
		}
		err = uc.handleCompletedPayment(ctx, invoice, product,
			p.GetToken(), p.GetCardFirstSix()+p.GetCardLastFour(),
			p.GetName(), p.GetEmail())
		if err != nil {
			uc.log.Errorf("Failed to process pay webhook for invoice %v: %v", invoice.ID, err)
		}
	case TtpStatusDeclined, "Failed":
		txID := strconv.FormatInt(p.GetTransactionId(), 10)
		uc.handleFailedPayment(ctx, invoice, txID)
	default:
		uc.log.Warnf("Unknown webhook status: %s for invoice: %d", p.GetStatus(), invoiceID)
	}

	return 0, "OK"
}

// HandleRecurrentWebhook processes TTP recurrent payment webhook.
func (uc *PaymentUseCase) HandleRecurrentWebhook(ctx context.Context, p *v1.WebhookPayload) (int, string) {
	uc.log.Infof("Recurrent webhook: SubscriptionId=%s, TransactionId=%d, Amount=%.2f",
		p.GetSubscriptionId(), p.GetTransactionId(), p.GetAmount())

	originalInvoice, err := uc.invoicesRepo.FindByTtpSubscriptionID(ctx, p.GetSubscriptionId())
	if err != nil {
		uc.log.Errorf("Failed to find invoice for TTP subscription %s: %v", p.GetSubscriptionId(), err)
		return 13, "Subscription not found"
	}

	product, err := uc.productRepo.GetProduct(ctx, originalInvoice.ProductID)
	if err != nil {
		return 13, "Product not found"
	}

	txID := strconv.FormatInt(p.GetTransactionId(), 10)
	existing, _ := uc.invoicesRepo.FindByExternalTransactionID(ctx, txID)
	if existing != nil {
		uc.log.Infof("Recurrent invoice for tx %s already exists", txID)
		return 0, "OK"
	}

	paidAt := time.Now()
	paidTill := paidAt.Add(calculateDuration(product.ProductPeriod))

	var subscriptionID int64
	if originalInvoice.SubscriptionID != nil {
		subscriptionID = *originalInvoice.SubscriptionID
	}

	subID := p.GetSubscriptionId()
	newInvoice, err := uc.invoicesRepo.CreateInvoice(ctx, data.InvoiceDto{
		TenantID:           originalInvoice.TenantID,
		UserID:             originalInvoice.UserID,
		AppID:              originalInvoice.AppID,
		ProductID:          originalInvoice.ProductID,
		Amount:             originalInvoice.Amount,
		Status:             enum.Paid,
		SubscriptionID:     subscriptionID,
		PaidAt:             &paidAt,
		PaidTill:           &paidTill,
		TtpTransactionID:   &txID,
		TtpSubscriptionID:  &subID,
		RecurrentProfileID: originalInvoice.PaymentProfileID,
	})
	if err != nil {
		uc.log.Errorf("Failed to create recurrent invoice: %v", err)
		return 13, "Failed to create invoice"
	}

	uc.log.Infof("Recurrent invoice %d created for subscription %s", newInvoice.ID, p.GetSubscriptionId())
	return 0, "OK"
}


func (uc *PaymentUseCase) GetPaymentStatus(ctx context.Context, txID string, actorID int64) (*v1.GetPaymentStatusResponse, error) {
	invoice, err := uc.invoicesRepo.FindByExternalTransactionID(ctx, txID)
	if err != nil {
		if ent.IsNotFound(err) {
			return &v1.GetPaymentStatusResponse{Found: false}, nil
		}
		return nil, v1.ErrorDatabaseQuery("failed to find invoice: %v", err)
	}

	if invoice.UserID != actorID {
		return &v1.GetPaymentStatusResponse{Found: false}, nil
	}

	// If invoice is still CREATED, poll TTP for actual status
	if invoice.Status == enum.Created && uc.paymentClient != nil {
		txIDInt, _ := strconv.ParseInt(txID, 10, 64)
		if txIDInt > 0 {
			// Use background context to avoid gRPC deadline cancellation
			bgCtx := context.Background()
			uc.pollAndProcessTransaction(bgCtx, invoice, txIDInt)
			// Re-read invoice after potential update
			if updated, rerr := uc.invoicesRepo.FindByExternalTransactionID(bgCtx, txID); rerr == nil && updated != nil {
				invoice = updated
			}
		}
	}

	resp := &v1.GetPaymentStatusResponse{
		Found:     true,
		Status:    string(invoice.Status),
		InvoiceId: invoice.ID,
		ProductId: invoice.ProductID,
		Price:     invoice.Price.String(),
		Currency:  invoice.Currency,
	}

	product, err := uc.productRepo.GetProduct(ctx, invoice.ProductID)
	if err == nil {
		resp.ProductName = product.Name
	}

	if invoice.PaidAt != nil {
		resp.PaidAt = invoice.PaidAt.Format(time.RFC3339)
	}
	if invoice.PaidTill != nil {
		resp.PaidTill = invoice.PaidTill.Format(time.RFC3339)
	}

	return resp, nil
}

// pollAndProcessTransaction checks TTP for transaction status and processes if completed.
func (uc *PaymentUseCase) pollAndProcessTransaction(ctx context.Context, invoice *ent.Invoice, txID int64) {
	ttpResp, err := uc.paymentClient.GetTransaction(ctx, txID)
	if err != nil {
		uc.log.Errorf("Failed to poll TTP for tx %d: %v", txID, err)
		return
	}

	if !ttpResp.Success {
		return
	}

	switch ttpResp.Model.Status {
	case TtpStatusCompleted, TtpStatusAuthorized:
		product, err := uc.productRepo.GetProduct(ctx, invoice.ProductID)
		if err != nil {
			uc.log.Errorf("Failed to get product %d: %v", invoice.ProductID, err)
			return
		}
		err = uc.handleCompletedPayment(ctx, invoice, product,
			ttpResp.Model.Token,
			ttpResp.Model.CardFirstSix+ttpResp.Model.CardLastFour,
			"", "")
		if err != nil {
			uc.log.Errorf("Failed to process polled payment for invoice %d: %v", invoice.ID, err)
		}
	case TtpStatusDeclined, "Failed":
		txIDStr := strconv.FormatInt(txID, 10)
		uc.handleFailedPayment(ctx, invoice, txIDStr)
	}
}

func (uc *PaymentUseCase) CancelSubscription(ctx context.Context, subscriptionID int64) error {
	// Find the latest invoice to get TTP subscription ID
	invoices, err := uc.invoicesRepo.ListInvoices(ctx, data.InvoiceFilter{
		SubscriptionID: subscriptionID,
	}, &utils_v1.PaginateRequest{Limit: 100})
	if err != nil {
		return v1.ErrorDatabaseQuery("failed to find invoices: %v", err)
	}

	// Cancel TTP subscription (stop auto-billing)
	for _, inv := range invoices {
		if inv.TtpSubscriptionID != nil && *inv.TtpSubscriptionID != "" {
			_, cancelErr := uc.paymentClient.CancelSubscription(ctx, *inv.TtpSubscriptionID)
			if cancelErr != nil {
				uc.log.Errorf("Failed to cancel TTP subscription %s: %v", *inv.TtpSubscriptionID, cancelErr)
			} else {
				uc.log.Infof("TTP subscription %s cancelled", *inv.TtpSubscriptionID)
			}
			break
		}
	}

	// Don't revoke access immediately — user keeps access until paid_till expires.
	// ExpireResources cron will handle RBAC downgrade when paid_till < now.
	// Just mark subscription as cancelled so we don't renew.
	err = uc.subscriptionRepo.RevokeActiveSubscription(ctx, subscriptionID, time.Now())
	if err != nil {
		if ent.IsNotFound(err) {
			return v1.ErrorNotFound("subscription not found")
		}
		return v1.ErrorDatabaseQuery("failed to cancel subscription: %v", err)
	}

	return nil
}

func (uc *PaymentUseCase) savePaymentProfile(
	ctx context.Context, userID int64, token, panMasked, holder, email string,
) (*ent.PaymentProfile, error) {
	existingProfile, err := uc.paymentProfileRepo.GetProfileByUserID(ctx, userID)
	if err != nil && !ent.IsNotFound(err) {
		return nil, fmt.Errorf("failed to check existing payment profile: %w", err)
	}

	if existingProfile == nil {
		existingProfile, err = uc.paymentProfileRepo.CreateProfile(
			ctx, data.PaymentProfileDto{
				UserID:         userID,
				PanMasked:      panMasked,
				Holder:         holder,
				Email:          &email,
				RecurrentToken: &token,
			},
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create payment profile: %w", err)
		}
	} else {
		err = uc.paymentProfileRepo.UpdateProfile(
			ctx, existingProfile.ID, data.PaymentProfileDto{
				Email:          &email,
				RecurrentToken: &token,
			},
		)
		if err != nil {
			return nil, fmt.Errorf("failed to update payment profile: %w", err)
		}
	}

	return existingProfile, nil
}

// CancelReservations cancels expired product reservations.
func (uc *PaymentUseCase) CancelReservations(ctx context.Context) {
	uc.log.Info("Processing expired reservations")
	err := uc.productReservationRepo.ProcessExpiredReservations(ctx)
	if err != nil {
		uc.log.Errorf("Failed to process expired reservations: %v", err)
	}
}

// HandlePaymentCallback handles legacy OVP callbacks (no-op, logs warning).
func (uc *PaymentUseCase) HandlePaymentCallback(ctx context.Context, req *v1.PaymentCallbackRequest) {
	uc.log.Warnf("Received legacy OVP callback — ignoring (OVP has been replaced by TipTopPay)")
}

// --- Helpers ---

func calculateDuration(period enum.ProductPeriod) time.Duration {
	switch period {
	case enum.ProductPeriodDay:
		return HoursInDay * time.Hour
	case enum.ProductPeriodWeek:
		return DaysInWeek * HoursInDay * time.Hour
	case enum.ProductPeriodMonth:
		return DaysInMonth * HoursInDay * time.Hour
	case enum.ProductPeriodYear:
		return DaysInYear * HoursInDay * time.Hour
	case enum.ProductPeriodUnlimited:
		return UnlimitedYears * DaysInYear * HoursInDay * time.Hour
	default:
		return DaysInMonth * HoursInDay * time.Hour
	}
}

// TTP interval constants
const (
	TtpIntervalDay   = "Day"
	TtpIntervalWeek  = "Week"
	TtpIntervalMonth = "Month"
)

func mapProductPeriodToTtp(period enum.ProductPeriod) (int, string) {
	switch period {
	case enum.ProductPeriodDay:
		return 1, TtpIntervalDay
	case enum.ProductPeriodWeek:
		return 1, TtpIntervalWeek
	case enum.ProductPeriodMonth:
		return 1, TtpIntervalMonth
	case enum.ProductPeriodYear:
		return 12, TtpIntervalMonth
	default:
		return 0, ""
	}
}
