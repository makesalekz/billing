package data

import (
	"context"
	"errors"
	"os"
	"strconv"

	"github.com/go-kratos/kratos/v2/log"

	"gitlab.calendaria.team/services/finance/billing/ent"
	"gitlab.calendaria.team/services/finance/billing/ent/enum"
	"gitlab.calendaria.team/services/finance/onevisionpay"
	"gitlab.calendaria.team/services/utils/v1/config"
)

const (
	DefaultPaymentLifeTime          = 60 * 15 * 1            // 15min
	DefaultRecurrentProfileLifeTime = 60 * 60 * 24 * 365 * 4 // 4 years
	DefaultPaymentLang              = "ru"
)

type OvpClient struct {
	paymentClient      *onevisionpay.Client
	log                *log.Helper
	PaymentSuccessURL  string
	PaymentFailureURL  string
	PaymentCallbackURL string
	MerchantID         string
	MerchantName       string
	ServiceID          string
}

func NewOvpClient(config *config.Config, logger log.Logger) *OvpClient {
	uc := &OvpClient{
		log: log.NewHelper(log.With(logger, "module", "data/ovp_client")),
	}

	if err := uc.loadConfig(config); err != nil {
		uc.log.Fatalf("failed to load config: %v", err)
		return uc
	}

	client, err := uc.initPaymentClient(config)
	if err != nil {
		uc.log.Fatalf("failed to init payment client: %v", err)
		return uc
	}
	uc.paymentClient = client

	return uc
}

func (c *OvpClient) initPaymentClient(config *config.Config) (*onevisionpay.Client, error) {
	debug := os.Getenv("DEBUG") != ""
	env := onevisionpay.Production
	if debug {
		env = onevisionpay.Sandbox
	}

	secrets, err := config.ReadSecretsFor(context.Background(), "onevisionpay")
	if err != nil {
		return nil, err
	}

	apiKey, _ := secrets["api_key"].(string)
	apiSecret, _ := secrets["api_secret"].(string)

	return onevisionpay.NewClient(apiKey, apiSecret, env)
}

func (c *OvpClient) loadConfig(config *config.Config) error {
	var err error
	c.PaymentSuccessURL, err = config.Value("PAYMENT_SUCCESS_URL").String()
	if err != nil {
		return err
	}

	c.PaymentCallbackURL, err = config.Value("PAYMENT_CALLBACK_URL").String()
	if err != nil {
		return err
	}

	c.PaymentFailureURL, err = config.Value("PAYMENT_FAILURE_URL").String()
	if err != nil {
		return err
	}

	secrets, err := config.ReadSecretsFor(context.Background(), "onevisionpay")
	if err != nil {
		return err
	}

	if err = c.parseSecrets(secrets); err != nil {
		return err
	}

	return nil
}

func (c *OvpClient) parseSecrets(secrets map[string]interface{}) error {
	var ok bool
	if c.MerchantID, ok = secrets["merchant_id"].(string); !ok || c.MerchantID == "" {
		return errors.New("merchant_id not set")
	}

	if c.MerchantName, ok = secrets["merchant_name"].(string); !ok || c.MerchantName == "" {
		return errors.New("merchant_name not set")
	}

	if c.ServiceID, ok = secrets["service_id"].(string); !ok || c.ServiceID == "" {
		return errors.New("service_id not set")
	}

	return nil
}

func (c *OvpClient) CreatePayment(
	actorID int64, invoice *ent.Invoice, product *ent.Product,
) (*onevisionpay.StatusResponse, error) {
	paymentRequest := c.getPaymentPayload(actorID, invoice, product)
	if err := paymentRequest.Validate(); err != nil {
		return nil, err
	}

	payment, err := c.paymentClient.CreatePayment(paymentRequest)
	if err != nil {
		return nil, err
	}

	return payment, nil
}

func (c *OvpClient) getPaymentPayload(
	actorID int64, invoice *ent.Invoice, product *ent.Product,
) onevisionpay.PaymentRequest {
	quantity := int(invoice.Amount)
	paymentRequest := onevisionpay.PaymentRequest{
		Amount:      invoice.Price.IntPart(),
		OrderID:     strconv.FormatInt(invoice.ID, 10),
		UserID:      strconv.FormatInt(actorID, 10),
		Description: product.Description,
		Items: []onevisionpay.PaymentItem{
			{
				MerchantID:   c.MerchantID,
				ServiceID:    c.ServiceID,
				MerchantName: c.MerchantName,
				Name:         product.Name,
				Quantity:     quantity,
				AmountOnePcs: product.Price.IntPart(),
				AmountSum:    invoice.Price.IntPart(),
			},
		},
		PaymentType:            onevisionpay.Pay,
		PaymentMethod:          onevisionpay.Ecom,
		Currency:               product.Currency,
		PaymentLifetime:        DefaultPaymentLifeTime,
		Lang:                   DefaultPaymentLang,
		CreateRecurrentProfile: false,
		SuccessURL:             c.PaymentSuccessURL,
		CallbackURL:            c.PaymentCallbackURL,
		FailureURL:             c.PaymentFailureURL,
	}

	if product.PaymentModel == enum.Recurrent {
		paymentRequest.CreateRecurrentProfile = true
		paymentRequest.RecurrentProfileLifetime = DefaultRecurrentProfileLifeTime
	}

	return paymentRequest
}

func (c *OvpClient) VerifySignature(data string, sign string) bool {
	return c.paymentClient.VerifySignature(data, sign)
}

func (c *OvpClient) ParsePayload(data string) (*onevisionpay.StatusResponse, int64, error) {
	payload, err := c.paymentClient.ParsePayload(data)

	if err != nil {
		return nil, 0, err
	}

	statusResponse, err := c.getPaymentStatus(payload.PaymentID, payload.OrderID)

	if err != nil {
		return nil, 0, err
	}

	invoiceID, err := strconv.ParseInt(statusResponse.OrderID, 10, 64)
	if err != nil {
		return nil, 0, err
	}

	return statusResponse, invoiceID, nil
}

func (c *OvpClient) getPaymentStatus(
	paymentID int64, orderID string,
) (*onevisionpay.StatusResponse, error) {
	statusResponse, err := c.paymentClient.PaymentStatus(
		onevisionpay.StatusRequest{
			PaymentID: paymentID,
			OrderID:   orderID,
		},
	)
	if err != nil {
		return nil, err
	}

	return statusResponse, nil
}

func (c *OvpClient) RecurrentPayment(
	invoice *ent.Invoice, product *ent.Product, profile *ent.PaymentProfile,
) (*onevisionpay.StatusResponse, *string, error) {
	recurrentRequest := onevisionpay.RecurrentRequest{
		RecurrentToken: *profile.RecurrentToken,
		Amount:         invoice.Amount,
		OrderID:        strconv.FormatInt(invoice.ID, 10),
		Description:    product.Description,
	}

	if err := recurrentRequest.Validate(); err != nil {
		return nil, nil, err
	}

	recurrentResponse, err := c.paymentClient.RecurrentPayment(recurrentRequest)
	if err != nil {
		return nil, nil, err
	}

	transactionID := strconv.FormatInt(recurrentResponse.PaymentID, 10)

	return recurrentResponse, &transactionID, nil
}
