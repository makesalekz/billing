package data

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/go-kratos/kratos/v2/log"

	"gitlab.calendaria.team/services/utils/v1/config"
)

type contextKey string

const ctxKeyRequestID contextKey = "ttp_request_id"

// WithRequestID adds idempotency key to context for TTP API calls.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, ctxKeyRequestID, id)
}

const (
	TtpBaseURLProduction = "https://api.tiptoppay.kz"
	TtpDefaultTimeout    = 30 * time.Second
)

// --- Request types ---

type TtpChargeRequest struct {
	Amount               float64 `json:"Amount"`
	Currency             string  `json:"Currency"`
	CardCryptogramPacket string  `json:"CardCryptogramPacket"`
	IpAddress            string  `json:"IpAddress"`
	Name                 string  `json:"Name,omitempty"`
	InvoiceId            string  `json:"InvoiceId,omitempty"`
	Description          string  `json:"Description,omitempty"`
	AccountId            string  `json:"AccountId,omitempty"`
	Email                string  `json:"Email,omitempty"`
	JsonData             any     `json:"JsonData,omitempty"`
}

type TtpPost3dsRequest struct {
	TransactionId int64  `json:"TransactionId"`
	PaRes         string `json:"PaRes"`
}

type TtpTokenChargeRequest struct {
	Amount      float64 `json:"Amount"`
	Currency    string  `json:"Currency"`
	Token       string  `json:"Token"`
	AccountId   string  `json:"AccountId"`
	InvoiceId   string  `json:"InvoiceId,omitempty"`
	Description string  `json:"Description,omitempty"`
	IpAddress   string  `json:"IpAddress,omitempty"`
}

type TtpRecurrentCreateRequest struct {
	AccountId   string  `json:"AccountId"`
	Amount      float64 `json:"Amount"`
	Currency    string  `json:"Currency"`
	Token       string  `json:"Token"`
	Period      int     `json:"Period"`
	Interval    string  `json:"Interval"`
	StartDate   string  `json:"StartDate"`
	Description string  `json:"Description,omitempty"`
	MaxPeriods  int     `json:"MaxPeriods,omitempty"`
}

type TtpRefundRequest struct {
	TransactionId int64   `json:"TransactionId"`
	Amount        float64 `json:"Amount"`
}

// --- Response types ---

type TtpResponse struct {
	Success bool                `json:"Success"`
	Message string              `json:"Message"`
	Model   TtpTransactionModel `json:"Model"`
}

type TtpSubscriptionResponse struct {
	Success bool                 `json:"Success"`
	Message string               `json:"Message"`
	Model   TtpSubscriptionModel `json:"Model"`
}

type TtpTransactionModel struct {
	TransactionId int64   `json:"TransactionId"`
	Amount        float64 `json:"Amount"`
	Currency      string  `json:"Currency"`
	Status        string  `json:"Status"`
	StatusCode    int     `json:"StatusCode"`
	ReasonCode    int     `json:"ReasonCode"`
	Token         string  `json:"Token"`
	AccountId     string  `json:"AccountId"`
	InvoiceId     string  `json:"InvoiceId"`
	CardFirstSix  string  `json:"CardFirstSix"`
	CardLastFour  string  `json:"CardLastFour"`
	CardExpDate   string  `json:"CardExpDate"`
	CardType      string  `json:"CardType"`
	AuthCode      string  `json:"AuthCode"`
	Rrn           string  `json:"Rrn"`
	TestMode      bool    `json:"TestMode"`
	// 3DS fields
	PaReq  string `json:"PaReq,omitempty"`
	AcsUrl string `json:"AcsUrl,omitempty"`
}

type TtpSubscriptionModel struct {
	SubscriptionId  string  `json:"SubscriptionId"`
	AccountId       string  `json:"AccountId"`
	Status          string  `json:"Status"`
	Amount          float64 `json:"Amount"`
	Currency        string  `json:"Currency"`
	Period          int     `json:"Period"`
	Interval        string  `json:"Interval"`
	NextPaymentDate string  `json:"NextPaymentDate"`
}

// --- Client ---

type PaymentClient interface {
	Charge(ctx context.Context, req TtpChargeRequest) (*TtpResponse, error)
	Post3ds(ctx context.Context, req TtpPost3dsRequest) (*TtpResponse, error)
	TokenCharge(ctx context.Context, req TtpTokenChargeRequest) (*TtpResponse, error)
	CreateSubscription(ctx context.Context, req TtpRecurrentCreateRequest) (*TtpSubscriptionResponse, error)
	CancelSubscription(ctx context.Context, subscriptionId string) (*TtpSubscriptionResponse, error)
	Refund(ctx context.Context, req TtpRefundRequest) (*TtpResponse, error)
	GetTransaction(ctx context.Context, transactionId int64) (*TtpResponse, error)
}

const maxResponseSize = 1 << 20 // 1 MB

type TtpClient struct {
	httpClient     *http.Client
	baseURL        string
	publicID       string
	apiSecret      string
	authHeaderVal  string // cached Basic auth header
	log            *log.Helper
	SuccessURL     string
	FailureURL     string
}

func NewTtpClient(cfg *config.Config, logger log.Logger) (*TtpClient, error) {
	uc := &TtpClient{
		log:     log.NewHelper(log.With(logger, "module", "data/ttp_client")),
		baseURL: TtpBaseURLProduction,
		httpClient: &http.Client{
			Timeout: TtpDefaultTimeout,
		},
	}

	if err := uc.loadConfig(cfg); err != nil {
		return nil, fmt.Errorf("failed to load TTP config: %w", err)
	}

	if os.Getenv("DEBUG") != "" {
		uc.log.Info("TTP client initialized in DEBUG mode")
	}

	return uc, nil
}

func (c *TtpClient) loadConfig(cfg *config.Config) error {
	var err error
	c.SuccessURL, err = cfg.Value("PAYMENT_SUCCESS_URL").String()
	if err != nil {
		return fmt.Errorf("PAYMENT_SUCCESS_URL not set: %w", err)
	}

	c.FailureURL, err = cfg.Value("PAYMENT_FAILURE_URL").String()
	if err != nil {
		return fmt.Errorf("PAYMENT_FAILURE_URL not set: %w", err)
	}

	secrets, err := cfg.ReadSecretsFor(context.Background(), "tiptoppay")
	if err != nil {
		return fmt.Errorf("failed to read TTP secrets: %w", err)
	}

	var ok bool
	if c.publicID, ok = secrets["public_id"].(string); !ok || c.publicID == "" {
		return fmt.Errorf("tiptoppay public_id not set")
	}
	if c.apiSecret, ok = secrets["api_secret"].(string); !ok || c.apiSecret == "" {
		return fmt.Errorf("tiptoppay api_secret not set")
	}

	creds := c.publicID + ":" + c.apiSecret
	c.authHeaderVal = "Basic " + base64.StdEncoding.EncodeToString([]byte(creds))

	return nil
}

func (c *TtpClient) doRequest(ctx context.Context, method, path string, body any) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request: %w", err)
		}
		reqBody = bytes.NewReader(jsonData)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", c.authHeaderVal)
	req.Header.Set("Content-Type", "application/json")

	if requestID := ctx.Value(ctxKeyRequestID); requestID != nil {
		req.Header.Set("X-Request-ID", requestID.(string))
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize))
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, fmt.Errorf("TTP rate limit exceeded")
	}

	if resp.StatusCode >= http.StatusInternalServerError {
		return nil, fmt.Errorf("TTP server error: %d", resp.StatusCode)
	}

	return respBody, nil
}

// --- PaymentClient interface implementation ---

func (c *TtpClient) Charge(ctx context.Context, req TtpChargeRequest) (*TtpResponse, error) {
	body, err := c.doRequest(ctx, http.MethodPost, "/payments/cards/charge", req)
	if err != nil {
		return nil, err
	}

	var resp TtpResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal charge response: %w", err)
	}

	return &resp, nil
}

func (c *TtpClient) Post3ds(ctx context.Context, req TtpPost3dsRequest) (*TtpResponse, error) {
	body, err := c.doRequest(ctx, http.MethodPost, "/payments/cards/post3ds", req)
	if err != nil {
		return nil, err
	}

	var resp TtpResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal post3ds response: %w", err)
	}

	return &resp, nil
}

func (c *TtpClient) TokenCharge(ctx context.Context, req TtpTokenChargeRequest) (*TtpResponse, error) {
	body, err := c.doRequest(ctx, http.MethodPost, "/payments/tokens/charge", req)
	if err != nil {
		return nil, err
	}

	var resp TtpResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal token charge response: %w", err)
	}

	return &resp, nil
}

func (c *TtpClient) CreateSubscription(ctx context.Context, req TtpRecurrentCreateRequest) (*TtpSubscriptionResponse, error) {
	body, err := c.doRequest(ctx, http.MethodPost, "/payments/recurrents/create", req)
	if err != nil {
		return nil, err
	}

	var resp TtpSubscriptionResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal subscription response: %w", err)
	}

	return &resp, nil
}

func (c *TtpClient) CancelSubscription(ctx context.Context, subscriptionId string) (*TtpSubscriptionResponse, error) {
	reqBody := map[string]string{"SubscriptionId": subscriptionId}
	body, err := c.doRequest(ctx, http.MethodPost, "/payments/recurrents/cancel", reqBody)
	if err != nil {
		return nil, err
	}

	var resp TtpSubscriptionResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal cancel subscription response: %w", err)
	}

	return &resp, nil
}

func (c *TtpClient) Refund(ctx context.Context, req TtpRefundRequest) (*TtpResponse, error) {
	body, err := c.doRequest(ctx, http.MethodPost, "/payments/refund", req)
	if err != nil {
		return nil, err
	}

	var resp TtpResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal refund response: %w", err)
	}

	return &resp, nil
}

func (c *TtpClient) GetTransaction(ctx context.Context, transactionId int64) (*TtpResponse, error) {
	reqBody := map[string]int64{"TransactionId": transactionId}
	body, err := c.doRequest(ctx, http.MethodPost, "/payments/get", reqBody)
	if err != nil {
		return nil, err
	}

	var resp TtpResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal transaction response: %w", err)
	}

	return &resp, nil
}
