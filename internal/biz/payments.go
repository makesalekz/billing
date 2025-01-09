package biz

import (
	"context"
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

type PaymentUsecase struct {
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
) (*PaymentUsecase, error) {
	helper := log.NewHelper(log.With(logger, "module", "usecase/payment"))
	helper.Info("creating onevisionpay client")

	uc := &PaymentUsecase{
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

func (uc *PaymentUsecase) loadConfig(config *config.Config) error {
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

	if err := uc.parseSecrets(secrets); err != nil {
		return fmt.Errorf("invalid secrets: %w", err)
	}

	return nil
}

func (uc *PaymentUsecase) parseSecrets(secrets map[string]interface{}) error {
	var ok bool
	if uc.MerchantID, ok = secrets["merchant_id"].(string); !ok || uc.MerchantID == "" {
		return fmt.Errorf("merchant_id not set")
	}

	if uc.MerchantName, ok = secrets["merchant_name"].(string); !ok || uc.MerchantName == "" {
		return fmt.Errorf("merchant_name not set")
	}

	if uc.ServiceID, ok = secrets["service_id"].(string); !ok || uc.ServiceID == "" {
		return fmt.Errorf("service_id not set")
	}

	return nil
}

func (uc *PaymentUsecase) initPaymentClient(config *config.Config) (*onevisionpay.Client, error) {
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

func (u *PaymentUsecase) CreatePayment(
	ctx context.Context, tenantID, actorID, productID int64, appID string,
) (int64, string, error) {
	product, err := u.productRepo.GetProduct(ctx, productID)
	if err != nil {
		if ent.IsNotFound(err) {
			u.log.Errorf("Product not found: %d", productID)
			return 0, "", v1.ErrorNotFound("product not found")
		}
		u.log.Errorf("Failed to get product %d: %v", productID, err)
		return 0, "", v1.ErrorDatabaseQuery("failed to get product: %v", err)
	}

	hasActive, isFirst, err := u.checkSubscriptionStatus(ctx, tenantID, actorID, productID)
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

	invoice, err := u.invoicesRepo.CreateInvoice(ctx, invoiceDTO)
	if err != nil {
		u.log.Errorf("Failed to create invoice: %+v, error: %v", invoiceDTO, err)
		return 0, "", v1.ErrorDatabaseQuery("failed to create invoice: %v", err)
	}

	if u.paymentClient == nil {
		return 0, "", v1.ErrorInternal("payment client is not initialized")
	}

	paymentRequest := u.getPaymentPayload(actorID, invoice, product, invoiceDTO.Price)
	if err := paymentRequest.Validate(); err != nil {
		return 0, "", v1.ErrorInvalidRequest("invalid payment payload: %v", err)
	}

	payment, err := u.paymentClient.CreatePayment(paymentRequest)
	if err != nil {
		u.log.Errorf("Failed to create payment: %v", err)
		return 0, "", v1.ErrorInvalidRequest("failed to create payment", err)
	}

	paymentIDStr := strconv.FormatInt(payment.PaymentID, 10)
	updatedInvoice, err := u.invoicesRepo.UpdateInvoice(
		ctx, invoice, data.InvoiceDto{
			AppleStoreTransactionID: &paymentIDStr,
		},
	)
	if err != nil {
		return 0, "", v1.ErrorDatabaseQuery("failed to update invoice: %v", err)
	}

	return updatedInvoice.ID, payment.PaymentPageURL, nil
}

func (u *PaymentUsecase) HandlePaymentCallback(ctx context.Context, req *v1.PaymentCallbackRequest) {
	u.log.Infof("Handling payment callback: %v", req)

	if u.paymentClient == nil {
		u.log.Errorf("Payment client is not initialized")
		return
	}

	if !u.paymentClient.VerifySignature(req.Data, req.Sign) {
		u.log.Errorf("Invalid signature: %v", req.Sign)
		return
	}

	payload, err := u.paymentClient.ParsePayload(req.Data)
	if err != nil {
		u.log.Errorf("Failed to parse payload: %v", err)
		return
	}

	paymentStatus, err := u.getPaymentStatus(payload.PaymentID, payload.OrderID)
	if err != nil {
		u.log.Errorf("Failed to check payment status: %v", err)
		return
	}

	invoiceID, err := strconv.ParseInt(paymentStatus.OrderID, 10, 64)
	if err != nil {
		u.log.Errorf("Failed to parse invoice ID: %v", err)
		return
	}

	invoice, err := u.invoicesRepo.GetInvoiceById(ctx, invoiceID)
	if err != nil {
		if ent.IsNotFound(err) {
			u.log.Errorf("Invoice not found: %v", invoiceID)
			return
		}
		u.log.Errorf("Failed to get invoice: %v", err)
		return
	}

	switch paymentStatus.PaymentStatus {
	case onevisionpay.Created:
		u.log.Infof("Payment created for invoice: %v", invoiceID)

	case onevisionpay.Refunded:
		u.log.Infof("Full refund processed for invoice: %v", invoiceID)
		err = u.updateInvoiceStatus(ctx, invoice, enum.CanceledByUser)

	case onevisionpay.Clearing, onevisionpay.Withdraw:
		u.log.Infof("Payment completed for invoice: %v", invoiceID)

		recurrentProfile, err := u.saveRecurrentProfile(
			ctx, invoice.UserID, paymentStatus.PayerInfo.PanMasked, paymentStatus.PayerInfo.Holder,
			paymentStatus.PayerInfo.Email,
			paymentStatus.PayerInfo.Phone, paymentStatus.PayerInfo.UserToken, paymentStatus.RecurrentToken,
		)
		if err != nil {
			u.log.Errorf("Failed to save payment profile for user %v: %v", invoice.UserID, err)
			return
		}

		u.log.Infof("Payment profile successfully saved for user %v", invoice.UserID)

		if invoice.SubscriptionID != nil {
			u.log.Infof(
				"Extending subscription for invoice: %v, subscription ID: %v", invoiceID, *invoice.SubscriptionID,
			)

			err = u.createOrExtendSubscription(ctx, *invoice.SubscriptionID)
			if err != nil {
				u.log.Errorf("Failed to extend subscription for subscription ID %v: %v", *invoice.SubscriptionID, err)
				return
			}

			u.log.Infof("Subscription %v successfully extended for invoice %v", *invoice.SubscriptionID, invoiceID)
		} else {
			u.log.Infof("No subscription linked to invoice %v. Creating a new subscription.", invoiceID)

			newSubscription, err := u.subscriptionRepo.CreateSubscription(
				ctx, invoice.UserID, invoice.TenantID, invoice.AppID, data.SubscriptionDto{
					ProductID: invoice.ProductID,
				},
			)
			if err != nil {
				u.log.Errorf("Failed to create subscription for invoice %v: %v", invoiceID, err)
				return
			}

			_, err = u.invoicesRepo.UpdateInvoice(
				ctx, invoice, data.InvoiceDto{
					SubscriptionID:     newSubscription.ID,
					RecurrentProfileId: &recurrentProfile.ID,
				},
			)
			if err != nil {
				u.log.Errorf(
					"Failed to update invoice %v with new subscription ID %v: %v", invoiceID, newSubscription.ID, err,
				)
				return
			}

			u.log.Infof(
				"New subscription %v successfully created and linked to invoice %v", newSubscription.ID, invoiceID,
			)
		}

	case onevisionpay.Canceled, onevisionpay.Error, onevisionpay.Chargeback:
		u.log.Infof("Payment failed or canceled for invoice: %v", invoiceID)

		err = u.updateInvoiceStatus(ctx, invoice, enum.CanceledByUser)
		if err != nil {
			u.log.Errorf("Failed to update invoice status to Canceled for invoice %v: %v", invoiceID, err)
			return
		}

		if invoice.SubscriptionID != nil {
			u.log.Infof(
				"Canceling subscription for invoice: %v, subscription ID: %v", invoiceID, *invoice.SubscriptionID,
			)

			err = u.subscriptionRepo.RevokeActiveSubscription(ctx, *invoice.SubscriptionID, time.Now())
			if err != nil {
				u.log.Errorf(
					"Failed to cancel subscription %v for invoice %v: %v", *invoice.SubscriptionID, invoiceID, err,
				)
				return
			}

			u.log.Infof("Subscription %v successfully canceled for invoice %v", *invoice.SubscriptionID, invoiceID)
		}

	case onevisionpay.PartialRefund:
		u.log.Infof("Partial refund for invoice: %v", invoiceID)

	default:
		u.log.Warnf("Unknown payment status: %v for invoice: %v", paymentStatus.PaymentStatus, invoiceID)
	}

	if err != nil {
		u.log.Errorf("Failed to update invoice status: %v", err)
		return
	}

	u.log.Infof("Callback processed successfully for invoice: %v", invoiceID)
}

func (u *PaymentUsecase) CancelSubscription(
	ctx context.Context, subscriptionID int64,
) error {
	err := u.subscriptionRepo.RevokeActiveSubscription(ctx, subscriptionID, time.Now())
	if err != nil {
		if ent.IsNotFound(err) {
			return v1.ErrorNotFound("subscription not found")
		}
		return v1.ErrorDatabaseQuery("failed to cancel subscription: %v", err)
	}
	return nil
}

func (u *PaymentUsecase) updateInvoiceStatus(
	ctx context.Context, invoice *ent.Invoice, status enum.InvoiceStatus,
) error {
	_, err := u.invoicesRepo.UpdateInvoice(
		ctx, invoice, data.InvoiceDto{
			Status: status,
		},
	)
	if err != nil {
		u.log.Errorf("Failed to update invoice %d status to %v: %v", invoice.ID, status, err)
	}
	return err
}

func (u *PaymentUsecase) getPaymentPayload(
	actorID int64, invoice *ent.Invoice, product *ent.Product, price decimal.Decimal,
) onevisionpay.PaymentRequest {
	paymentRequest := onevisionpay.PaymentRequest{
		Amount:      price.IntPart(),
		OrderID:     strconv.FormatInt(invoice.ID, 10),
		UserID:      strconv.FormatInt(actorID, 10),
		Description: product.Description,
		Items: []onevisionpay.PaymentItem{
			{
				MerchantID:   u.MerchantID,
				ServiceID:    u.ServiceID,
				MerchantName: u.MerchantName,
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
		SuccessURL:               u.PaymentSuccessURL,
		CallbackURL:              u.PaymentCallbackURL,
		FailureURL:               u.PaymentFailureURL,
	}
	return paymentRequest
}

func (u *PaymentUsecase) checkSubscriptionStatus(
	ctx context.Context, tenantID, actorID, productID int64,
) (hasActive bool, isFirst bool, err error) {
	subscriptions, err := u.subscriptionRepo.ListSubscriptions(ctx, actorID, false, nil)
	if err != nil {
		u.log.Errorf("Failed to list subscriptions: %v", err)
		return false, false, err
	}

	// Проверяем наличие активных подписок для продукта
	for _, sub := range subscriptions {
		if sub.ProductID == productID && sub.TenantID == tenantID {
			hasActive = true
			break
		}
	}

	// Проверяем, была ли это первая подписка
	isFirst = len(subscriptions) == 0

	return hasActive, isFirst, nil
}

func (u *PaymentUsecase) saveRecurrentProfile(
	ctx context.Context, userID int64, panMasked, holder, email, phone, userToken, recurrentToken string,
) (*ent.PaymentProfile, error) {
	// Проверяем, существует ли профиль
	existingProfile, err := u.paymentProfileRepo.GetProfileByUserID(ctx, userID)
	if err != nil && !ent.IsNotFound(err) {
		return nil, fmt.Errorf("failed to check existing payment profile: %w", err)
	}

	if existingProfile != nil {
		u.log.Infof("Payment profile already exists for user %v. Updating profile.", userID)

		// Обновляем существующий профиль
		err = u.paymentProfileRepo.UpdateProfile(
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
		u.log.Infof("Creating new payment profile for user %v.", userID)

		// Создаём новый профиль
		existingProfile, err = u.paymentProfileRepo.CreateProfile(
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

func (u *PaymentUsecase) getPaymentStatus(
	paymentID int64, orderID string,
) (*onevisionpay.StatusResponse, error) {
	// Запрос статуса платежа через API
	statusResponse, err := u.paymentClient.PaymentStatus(
		onevisionpay.StatusRequest{
			PaymentID: paymentID,
			OrderID:   orderID,
		},
	)
	if err != nil {
		u.log.Errorf("Failed to fetch payment status for payment ID %v: %v", paymentID, err)
		return nil, err
	}
	return statusResponse, nil
}

func (u *PaymentUsecase) createOrExtendSubscription(ctx context.Context, subscriptionID int64) error {
	err := u.subscriptionRepo.CreateOrExtendSubscription(ctx, subscriptionID)
	if err != nil {
		return fmt.Errorf("failed to create or extend subscription: %w", err)
	}
	return nil
}
