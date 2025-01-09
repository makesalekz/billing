package biz

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/shopspring/decimal"

	v1 "gitlab.calendaria.team/services/finance/billing/api/billing/v1"
	"gitlab.calendaria.team/services/finance/billing/ent"
	"gitlab.calendaria.team/services/finance/billing/ent/enum"
	"gitlab.calendaria.team/services/finance/billing/internal/data"
	"gitlab.calendaria.team/services/finance/onevisionpay"
	"gitlab.calendaria.team/services/utils/v1/config"
)

type PaymentUseCase struct {
	paymentClient      *onevisionpay.Client
	invoicesRepo       data.InvoicesRepo
	productRepo        data.ProductRepo
	subscriptionRepo   data.SubscriptionsRepo
	paymentProfileRepo data.PaymentProfileRepo
	log                *log.Helper
	PaymentSuccessURL  string
	PaymentFailureURL  string
	PaymentCallbackURL string
	MerchantID         string
	MerchantName       string
	ServiceID          string
}

func NewPaymentUsecase(
	config *config.Config,
	logger log.Logger,
	invoicesRepo data.InvoicesRepo,
	productRepo data.ProductRepo,
	subscriptionRepo data.SubscriptionsRepo,
	paymentProfileRepo data.PaymentProfileRepo,
) (*PaymentUseCase, error) {
	helper := log.NewHelper(log.With(logger, "module", "usecase/payment"))
	helper.Info("creating onevisionpay client")

	uc := &PaymentUseCase{
		log:                helper,
		invoicesRepo:       invoicesRepo,
		productRepo:        productRepo,
		subscriptionRepo:   subscriptionRepo,
		paymentProfileRepo: paymentProfileRepo,
	}

	if err := uc.loadConfig(config); err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	client, err := uc.initPaymentClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize payment client: %w", err)
	}
	uc.paymentClient = client

	return uc, nil
}

func (uc *PaymentUseCase) loadConfig(config *config.Config) error {
	var err error
	uc.PaymentSuccessURL, err = config.Value("PAYMENT_SUCCESS_URL").String()
	if err != nil {
		return fmt.Errorf("missing PAYMENT_SUCCESS_URL: %w", err)
	}

	uc.PaymentCallbackURL, err = config.Value("PAYMENT_CALLBACK_URL").String()
	if err != nil {
		return fmt.Errorf("missing PAYMENT_CALLBACK_URL: %w", err)
	}

	uc.PaymentFailureURL, err = config.Value("PAYMENT_FAILURE_URL").String()
	if err != nil {
		return fmt.Errorf("missing PAYMENT_FAILURE_URL: %w", err)
	}

	secrets, err := config.ReadSecretsFor(context.Background(), "onevisionpay")
	if err != nil {
		return fmt.Errorf("failed to read secrets: %w", err)
	}

	if err = uc.parseSecrets(secrets); err != nil {
		return fmt.Errorf("invalid secrets: %w", err)
	}

	return nil
}

func (uc *PaymentUseCase) parseSecrets(secrets map[string]interface{}) error {
	var ok bool
	if uc.MerchantID, ok = secrets["merchant_id"].(string); !ok || uc.MerchantID == "" {
		return errors.New("merchant_id not set")
	}

	if uc.MerchantName, ok = secrets["merchant_name"].(string); !ok || uc.MerchantName == "" {
		return errors.New("merchant_name not set")
	}

	if uc.ServiceID, ok = secrets["service_id"].(string); !ok || uc.ServiceID == "" {
		return errors.New("service_id not set")
	}

	return nil
}

func (uc *PaymentUseCase) initPaymentClient(config *config.Config) (*onevisionpay.Client, error) {
	debug := os.Getenv("DEBUG") != ""
	env := onevisionpay.Production
	if debug {
		env = onevisionpay.Sandbox
	}

	secrets, err := config.ReadSecretsFor(context.Background(), "onevisionpay")
	if err != nil {
		return nil, fmt.Errorf("failed to read onevisionpay secrets: %w", err)
	}

	apiKey, _ := secrets["api_key"].(string)
	apiSecret, _ := secrets["api_secret"].(string)

	return onevisionpay.NewClient(apiKey, apiSecret, env)
}

func (uc *PaymentUseCase) CreatePayment(
	ctx context.Context, tenantID, actorID, productID int64, appID string,
) (int64, string, error) {
	product, err := uc.productRepo.GetProduct(ctx, productID)
	if err != nil {
		if ent.IsNotFound(err) {
			uc.log.Errorf("Product not found: %d", productID)
			return 0, "", v1.ErrorNotFound("product not found")
		}
		uc.log.Errorf("Failed to get product %d: %v", productID, err)
		return 0, "", v1.ErrorDatabaseQuery("failed to get product: %v", err)
	}

	hasActive, isFirst, err := uc.checkSubscriptionStatus(ctx, tenantID, actorID, appID)
	if err != nil {
		return 0, "", err
	}

	if hasActive {
		return 0, "", v1.ErrorInvalidRequest("user already has active subscription")
	}

	invoiceDTO := data.InvoiceDto{
		TenantID:  tenantID,
		UserID:    actorID,
		AppID:     appID,
		ProductID: productID,
		Status:    enum.Created,
		Amount:    product.Price.IntPart(),
		Price:     product.Price,
	}

	if isFirst {
		invoiceDTO.Amount = 0
		invoiceDTO.Price = decimal.NewFromInt(0)
		invoiceDTO.IsTrial = true
	}

	invoice, err := uc.invoicesRepo.CreateInvoice(ctx, invoiceDTO)
	if err != nil {
		uc.log.Errorf("Failed to create invoice: %+v, error: %v", invoiceDTO, err)
		return 0, "", v1.ErrorDatabaseQuery("failed to create invoice: %v", err)
	}

	if uc.paymentClient == nil {
		return 0, "", v1.ErrorInternal("payment client is not initialized")
	}

	paymentRequest := uc.getPaymentPayload(actorID, invoice, product, invoiceDTO.Price)
	if err = paymentRequest.Validate(); err != nil {
		return 0, "", v1.ErrorInvalidRequest("invalid payment payload: %v", err)
	}

	payment, err := uc.paymentClient.CreatePayment(paymentRequest)
	if err != nil {
		uc.log.Errorf("Failed to create payment: %v", err)
		return 0, "", v1.ErrorInvalidRequest("failed to create payment %v", err)
	}

	paymentIDStr := strconv.FormatInt(payment.PaymentID, 10)
	updatedInvoice, err := uc.invoicesRepo.UpdateInvoice(
		ctx, invoice, data.InvoiceDto{
			AppleStoreTransactionID: &paymentIDStr,
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

	payload, err := uc.paymentClient.ParsePayload(req.GetData())
	if err != nil {
		uc.log.Errorf("Failed to parse payload: %v", err)
		return
	}

	paymentStatus, err := uc.getPaymentStatus(payload.PaymentID, payload.OrderID)
	if err != nil {
		uc.log.Errorf("Failed to check payment status: %v", err)
		return
	}

	invoiceID, err := strconv.ParseInt(paymentStatus.OrderID, 10, 64)
	if err != nil {
		uc.log.Errorf("Failed to parse invoice ID: %v", err)
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

	err = uc.processPaymentStatus(ctx, invoice, paymentStatus)

	if err != nil {
		uc.log.Errorf("Failed to update invoice status: %v", err)
		return
	}

	uc.log.Infof("Callback processed successfully for invoice: %v", invoiceID)
}

func (uc *PaymentUseCase) processPaymentStatus(
	ctx context.Context, invoice *ent.Invoice, paymentStatus *onevisionpay.StatusResponse,
) error {
	transactionID := strconv.FormatInt(paymentStatus.PaymentID, 10)
	switch paymentStatus.PaymentStatus {
	case onevisionpay.Created:
		return uc.handleCreatedStatus(invoice)
	case onevisionpay.Refunded:
		return uc.handleRefundedStatus(ctx, invoice, &transactionID)
	case onevisionpay.Clearing, onevisionpay.Withdraw:
		return uc.handleCompletedStatus(ctx, invoice, paymentStatus)
	case onevisionpay.Canceled, onevisionpay.Error, onevisionpay.Chargeback:
		return uc.handleFailedOrCanceledStatus(ctx, invoice, &transactionID)
	case onevisionpay.PartialRefund:
		return uc.handlePartialRefundStatus(invoice)
	case onevisionpay.Processing, onevisionpay.NeedApprove, onevisionpay.Hold,
		onevisionpay.Refill, onevisionpay.Process, onevisionpay.PartialClearing:
		return uc.handleNonWidgetStatus(paymentStatus, invoice)
	default:
		uc.log.Warnf("Unknown payment status: %v for invoice: %v", paymentStatus.PaymentStatus, invoice.ID)
		return nil
	}
}

func (uc *PaymentUseCase) handleCreatedStatus(invoice *ent.Invoice) error {
	uc.log.Infof("Payment created for invoice: %v", invoice.ID)
	return nil
}

func (uc *PaymentUseCase) handleRefundedStatus(ctx context.Context, invoice *ent.Invoice, transactionID *string) error {
	uc.log.Infof("Full refund processed for invoice: %v", invoice.ID)

	_, err := uc.invoicesRepo.UpdateInvoice(
		ctx, invoice, data.InvoiceDto{
			Status:                  enum.CanceledByUser,
			AppleStoreTransactionID: transactionID,
		},
	)
	if err != nil {
		uc.log.Errorf("Failed to update invoice %d status to %v: %v", invoice.ID, enum.CanceledByUser, err)
		return err
	}
	return nil
}

func (uc *PaymentUseCase) handleCompletedStatus(
	ctx context.Context, invoice *ent.Invoice, paymentStatus *onevisionpay.StatusResponse,
) error {
	uc.log.Infof("Payment completed for invoice: %v", invoice.ID)

	recurrentProfile, err := uc.saveRecurrentProfile(
		ctx, invoice.UserID, paymentStatus.PayerInfo.PanMasked, paymentStatus.PayerInfo.Holder,
		paymentStatus.PayerInfo.Email, paymentStatus.PayerInfo.Phone,
		paymentStatus.PayerInfo.UserToken, paymentStatus.RecurrentToken,
	)
	if err != nil {
		uc.log.Errorf("Failed to save payment profile for user %v: %v", invoice.UserID, err)
		return err
	}

	uc.log.Infof("Payment profile successfully saved for user %v", invoice.UserID)
	if invoice.SubscriptionID != nil {
		return uc.extendSubscription(ctx, invoice)
	}
	return uc.createNewSubscription(ctx, invoice, recurrentProfile)
}

// Продление подписки.
func (uc *PaymentUseCase) extendSubscription(ctx context.Context, invoice *ent.Invoice) error {
	uc.log.Infof("Extending subscription for invoice: %v, subscription ID: %v", invoice.ID, *invoice.SubscriptionID)
	err := uc.createOrExtendSubscription(ctx, *invoice.SubscriptionID)
	if err != nil {
		uc.log.Errorf("Failed to extend subscription for subscription ID %v: %v", *invoice.SubscriptionID, err)
		return err
	}
	uc.log.Infof("Subscription %v successfully extended for invoice %v", *invoice.SubscriptionID, invoice.ID)
	return nil
}

func (uc *PaymentUseCase) createNewSubscription(
	ctx context.Context, invoice *ent.Invoice, profile *ent.PaymentProfile,
) error {
	uc.log.Infof("No subscription linked to invoice %v. Creating a new subscription.", invoice.ID)

	newSubscription, err := uc.subscriptionRepo.CreateSubscription(
		ctx, invoice.UserID, invoice.TenantID, invoice.AppID, data.SubscriptionDto{
			ProductID: invoice.ProductID,
		},
	)
	if err != nil {
		uc.log.Errorf("Failed to create subscription for invoice %v: %v", invoice.ID, err)
		return err
	}

	now := time.Now()
	paidTill := now.AddDate(0, 1, 0) // Добавляем 1 месяц к текущей дате (пример)

	updateDto := data.InvoiceDto{
		Amount:             invoice.Amount,
		Price:              invoice.Price,
		Status:             enum.Paid,
		SubscriptionID:     newSubscription.ID,
		RecurrentProfileID: &profile.ID,
		PaidAt:             &now,
		PaidTill:           &paidTill,
	}

	_, updateErr := uc.invoicesRepo.UpdateInvoice(
		ctx, invoice, updateDto,
	)
	if updateErr != nil {
		uc.log.Errorf(
			"Failed to update invoice %v with new subscription ID %v: %v", invoice.ID, newSubscription.ID, updateErr,
		)
		return updateErr
	}

	uc.log.Infof(
		"New subscription %v successfully created and linked to invoice %v", newSubscription.ID, invoice.ID,
	)
	return nil
}

func (uc *PaymentUseCase) handleFailedOrCanceledStatus(
	ctx context.Context, invoice *ent.Invoice,
	transactionID *string,
) error {
	uc.log.Infof("Payment failed or canceled for invoice: %v", invoice.ID)

	now := time.Now()
	isRevoked := true

	updateDto := data.InvoiceDto{
		Status:                  enum.CanceledByUser,
		RevokedAt:               &now,
		IsRevoked:               &isRevoked,
		AppleStoreTransactionID: transactionID,
	}

	_, err := uc.invoicesRepo.UpdateInvoice(ctx, invoice, updateDto)
	if err != nil {
		uc.log.Errorf("Failed to update invoice status to Canceled for invoice %v: %v", invoice.ID, err)
		return err
	}

	uc.log.Infof("Invoice %v successfully updated with status Canceled", invoice.ID)

	if invoice.SubscriptionID != nil {
		uc.log.Infof(
			"Canceling subscription for invoice: %v, subscription ID: %v", invoice.ID, *invoice.SubscriptionID,
		)

		err = uc.subscriptionRepo.RevokeActiveSubscription(ctx, *invoice.SubscriptionID, now)
		if err != nil {
			uc.log.Errorf(
				"Failed to cancel subscription %v for invoice %v: %v", *invoice.SubscriptionID, invoice.ID, err,
			)
			return err
		}

		uc.log.Infof("Subscription %v successfully canceled for invoice %v", *invoice.SubscriptionID, invoice.ID)
	}

	return nil
}

func (uc *PaymentUseCase) handlePartialRefundStatus(invoice *ent.Invoice) error {
	uc.log.Infof("Partial refund for invoice: %v", invoice.ID)
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

func (uc *PaymentUseCase) getPaymentPayload(
	actorID int64, invoice *ent.Invoice, product *ent.Product, price decimal.Decimal,
) onevisionpay.PaymentRequest {
	paymentRequest := onevisionpay.PaymentRequest{
		Amount:      price.IntPart(),
		OrderID:     strconv.FormatInt(invoice.ID, 10),
		UserID:      strconv.FormatInt(actorID, 10),
		Description: product.Description,
		Items: []onevisionpay.PaymentItem{
			{
				MerchantID:   uc.MerchantID,
				ServiceID:    uc.ServiceID,
				MerchantName: uc.MerchantName,
				Name:         product.Name,
				Quantity:     1,
				AmountOnePcs: price.IntPart(),
				AmountSum:    price.IntPart(),
			},
		},
		PaymentType:              onevisionpay.Pay,
		PaymentMethod:            onevisionpay.Ecom,
		Currency:                 DefaultPaymentCurrency,
		PaymentLifetime:          DefaultPaymentLifeTime,
		RecurrentProfileLifetime: DefaultRecurrentProfileLifeTime,
		Lang:                     DefaultPaymentLang,
		CreateRecurrentProfile:   true,
		SuccessURL:               uc.PaymentSuccessURL,
		CallbackURL:              uc.PaymentCallbackURL,
		FailureURL:               uc.PaymentFailureURL,
	}
	return paymentRequest
}

func (uc *PaymentUseCase) checkSubscriptionStatus(
	ctx context.Context, tenantID, actorID int64, appID string,
) (bool, bool, error) {
	subscriptions, err := uc.subscriptionRepo.ListSubscriptions(ctx, actorID, true, nil)
	if err != nil {
		uc.log.Errorf("Failed to list subscriptions: %v", err)
		return false, false, err
	}

	hasActive := false
	for _, sub := range subscriptions {
		if sub.TenantID == tenantID && sub.AppID == appID {
			for _, invoice := range sub.Edges.Invoices {
				if invoice.IsRevoked {
					continue
				}
				hasActive = true
				break
			}
		}
	}

	isFirst := len(subscriptions) == 0
	return hasActive, isFirst, nil
}

func (uc *PaymentUseCase) saveRecurrentProfile(
	ctx context.Context, userID int64, panMasked, holder, email, phone, userToken, recurrentToken string,
) (*ent.PaymentProfile, error) {
	existingProfile, err := uc.paymentProfileRepo.GetProfileByUserID(ctx, userID)
	if err != nil && !ent.IsNotFound(err) {
		return nil, fmt.Errorf("failed to check existing payment profile: %w", err)
	}

	if existingProfile != nil {
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
	} else {
		uc.log.Infof("Creating new payment profile for user %v.", userID)

		existingProfile, err = uc.paymentProfileRepo.CreateProfile(
			ctx, data.PaymentProfileDto{
				UserID:         userID,
				PanMasked:      panMasked,
				Holder:         holder,
				Email:          &email,
				Phone:          &phone,
				UserToken:      userToken,
				RecurrentToken: &recurrentToken,
			},
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create payment profile: %w", err)
		}
	}

	return existingProfile, nil
}

func (uc *PaymentUseCase) getPaymentStatus(
	paymentID int64, orderID string,
) (*onevisionpay.StatusResponse, error) {
	statusResponse, err := uc.paymentClient.PaymentStatus(
		onevisionpay.StatusRequest{
			PaymentID: paymentID,
			OrderID:   orderID,
		},
	)
	if err != nil {
		uc.log.Errorf("Failed to fetch payment status for payment ID %v: %v", paymentID, err)
		return nil, err
	}
	return statusResponse, nil
}

func (uc *PaymentUseCase) createOrExtendSubscription(ctx context.Context, subscriptionID int64) error {
	err := uc.subscriptionRepo.CreateOrExtendSubscription(ctx, subscriptionID)
	if err != nil {
		return fmt.Errorf("failed to create or extend subscription: %w", err)
	}
	return nil
}

// ProcessExpiredPayments processes expired payments for recurrent profiles
func (uc *PaymentUseCase) ProcessExpiredPayments(ctx context.Context) error {
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

	return nil
}

// createRecurrentPayment creates a recurrent payment for the given invoice
func (uc *PaymentUseCase) createRecurrentPayment(ctx context.Context, invoice *ent.Invoice) {
	if invoice.AppID != PmsAppID {
		uc.log.Infof("Skipping recurrent payment for app %v", invoice.AppID)
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

	product, err := uc.productRepo.GetProduct(ctx, invoice.ProductID)
	if err != nil {
		uc.log.Errorf("Failed to get product for invoice %v: %v", invoice.ID, err)
		return
	}

	newInvoiceDTO := data.InvoiceDto{
		TenantID:           invoice.TenantID,
		UserID:             invoice.UserID,
		AppID:              invoice.AppID,
		ProductID:          invoice.ProductID,
		Status:             enum.Created,
		Amount:             product.Price.IntPart(),
		Price:              product.Price,
		RecurrentProfileID: invoice.PaymentProfileID,
	}

	newInvoice, err := uc.invoicesRepo.CreateInvoice(ctx, newInvoiceDTO)
	if err != nil {
		uc.log.Errorf("Failed to create new invoice for recurrent payment: %v", err)
		return
	}

	uc.log.Infof("New invoice created for recurrent payment, invoice ID: %v", newInvoice.ID)

	recurrentRequest := onevisionpay.RecurrentRequest{
		RecurrentToken: *invoice.Edges.PaymentProfile.RecurrentToken,
		Amount:         product.Price.IntPart(),
		OrderID:        strconv.FormatInt(newInvoice.ID, 10),
		Description:    product.Description,
	}

	if uc.paymentClient == nil {
		uc.log.Error("Payment client is not initialized")
		return
	}

	if err := recurrentRequest.Validate(); err != nil {
		uc.log.Errorf("Invalid recurrent payment request: %v", err)
		return
	}

	response, err := uc.paymentClient.RecurrentPayment(recurrentRequest)
	if err != nil {
		uc.log.Errorf("Failed to create recurrent payment: %v", err)
		return
	}

	transactionID := strconv.FormatInt(response.PaymentID, 10)

	_, err = uc.invoicesRepo.UpdateInvoice(
		ctx, invoice, data.InvoiceDto{
			Status:                  enum.Created,
			AppleStoreTransactionID: &transactionID,
		},
	)
	if err != nil {
		uc.log.Errorf("Failed to update invoice %d status to %v: %v", invoice.ID, enum.Created, err)
	}

	uc.log.Infof("Recurrent payment successfully created, payment ID: %v", response.PaymentID)
	return
}
