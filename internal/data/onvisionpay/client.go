package onevisionpay

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type ClientInterface interface {
	// Methods for payment operations
	CreatePayment(req PaymentRequest) (*StatusResponse, error)
	RefundPayment(req RefundRequest) (*StatusResponse, error)
	CancelPayment(req CancelRequest) (*StatusResponse, error)
	ClearingRequest(req ClearingRequest) (*StatusResponse, error)
	ConfirmPayment(req ConfirmRequest) (*StatusResponse, error)
	RecurrentPayment(req RecurrentRequest) (*StatusResponse, error)
	PaymentStatus(req StatusRequest) (*StatusResponse, error)

	// Methods for other operations
	PayoutBalanceInfo(req BalanceRequest) (*BalanceResponse, error)
	GetReceipt(req ReceiptRequest) (*ReceiptResponse, error)

	// Utility methods
	VerifySignature(data, sign string) bool
	ParsePayload(data string) (*StatusResponse, error)
}

type Environment string

const (
	Sandbox    Environment = "sandbox"
	Production Environment = "production"
)

var baseURLs = map[Environment]string{
	Sandbox:    "https://api.onevisionpay.com",
	Production: "https://api.onevisionpay.com",
}

type Client struct {
	BaseURL     string       // Базовый URL API OneVisionPay
	APIKey      string       // Ключ для аутентификации
	SecretKey   string       // Секретный ключ для подписи запросов
	HTTPClient  *http.Client // HTTP клиент для выполнения запросов
	Environment Environment  // Окружение (sandbox или production)
}

// NewClient create new client for OneVisionPay API
func NewClient(apiKey, secretKey string, env Environment) (*Client, error) {
	baseURL, exists := baseURLs[env]
	if !exists {
		return nil, fmt.Errorf("unknown environment: %s", env)
	}

	return &Client{
		BaseURL:     baseURL,
		APIKey:      apiKey,
		SecretKey:   secretKey,
		Environment: env,
		HTTPClient:  &http.Client{},
	}, nil
}

// Request base method for sending requests to OneVisionPay API with HMAC SHA-512 signature
func (c *Client) Request(method, endpoint string, requestBody interface{}) ([]byte, error) {
	dataJson, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}

	dataBase64 := base64.StdEncoding.EncodeToString(dataJson)

	mac := hmac.New(sha512.New, []byte(c.SecretKey))
	mac.Write([]byte(dataBase64))
	signature := fmt.Sprintf("%x", mac.Sum(nil))

	signedRequest := map[string]interface{}{
		"data": dataBase64,
		"sign": signature,
	}

	signedJsonData, err := json.Marshal(signedRequest)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s%s", c.BaseURL, endpoint)

	httpReq, err := http.NewRequest(method, url, bytes.NewBuffer(signedJsonData))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	apiKeyBase64 := base64.StdEncoding.EncodeToString([]byte(c.APIKey))
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKeyBase64))

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status code: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var apiResp APIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %s", string(body))
	}

	if !apiResp.Success {
		return nil, fmt.Errorf("api error: %s, message: %s", apiResp.ErrorCode, apiResp.ErrorMsg)
	}

	var decodedData []byte
	if apiResp.Data != "" {
		decodedData, err = base64.StdEncoding.DecodeString(apiResp.Data)
		if err != nil {
			return nil, fmt.Errorf("failed to decode data: %s", err)
		}
	}

	return decodedData, nil
}

// CreatePayment create new payment with specified parameters
func (c *Client) CreatePayment(req PaymentRequest) (*StatusResponse, error) {
	endpoint := "/payment/create"

	// Используем метод Request для выполнения запроса
	responseBytes, err := c.Request("POST", endpoint, req)
	if err != nil {
		return nil, err
	}

	// Раскодируем тело ответа в структуру PaymentResponse
	var paymentResp StatusResponse
	if err := json.Unmarshal(responseBytes, &paymentResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	return &paymentResp, nil
}

// RefundPayment refund payment with specified parameters
func (c *Client) RefundPayment(req RefundRequest) (*StatusResponse, error) {
	endpoint := "/payment/refund"

	responseBytes, err := c.Request("POST", endpoint, req)
	if err != nil {
		return nil, err
	}

	var refundResp StatusResponse
	if err := json.Unmarshal(responseBytes, &refundResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	return &refundResp, nil
}

// CancelPayment cancel payment with specified parameters
func (c *Client) CancelPayment(req CancelRequest) (*StatusResponse, error) {
	endpoint := "/payment/cancel"

	if err := req.Validate(); err != nil {
		return nil, err
	}

	responseBytes, err := c.Request("POST", endpoint, req)
	if err != nil {
		return nil, err
	}

	var cancelResp StatusResponse
	if err := json.Unmarshal(responseBytes, &cancelResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	return &cancelResp, nil
}

// ClearingRequest clears payment with specified parameters
func (c *Client) ClearingRequest(req ClearingRequest) (*StatusResponse, error) {
	endpoint := "/payment/capture"

	if err := req.Validate(); err != nil {
		return nil, err
	}
	responseBytes, err := c.Request("POST", endpoint, req)
	if err != nil {
		return nil, err
	}

	var clearingResp StatusResponse
	if err := json.Unmarshal(responseBytes, &clearingResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	return &clearingResp, nil
}

// ConfirmPayment confirms payment with specified parameters
func (c *Client) ConfirmPayment(req ConfirmRequest) (*StatusResponse, error) {
	endpoint := "/payment/confirm"

	if err := req.Validate(); err != nil {
		return nil, err
	}
	responseBytes, err := c.Request("POST", endpoint, req)
	if err != nil {
		return nil, err
	}

	var confirmResp StatusResponse
	if err := json.Unmarshal(responseBytes, &confirmResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	return &confirmResp, nil
}

// RecurrentPayment make recurrent payment with specified parameters
func (c *Client) RecurrentPayment(req RecurrentRequest) (*StatusResponse, error) {
	endpoint := "/payment/recurrent"

	if err := req.Validate(); err != nil {
		return nil, err
	}

	responseBytes, err := c.Request("POST", endpoint, req)
	if err != nil {
		return nil, err
	}

	var recurrentResp StatusResponse
	if err := json.Unmarshal(responseBytes, &recurrentResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	return &recurrentResp, nil
}

// PaymentStatus get payment status with specified parameters
func (c *Client) PaymentStatus(req StatusRequest) (*StatusResponse, error) {
	endpoint := "/payment/status"

	if err := req.Validate(); err != nil {
		return nil, err
	}

	responseBytes, err := c.Request("POST", endpoint, req)
	if err != nil {
		return nil, err
	}

	var statusResp StatusResponse
	if err := json.Unmarshal(responseBytes, &statusResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	return &statusResp, nil
}

// PayoutBalanceInfo get payout balance info with specified parameters
func (c *Client) PayoutBalanceInfo(req BalanceRequest) (*BalanceResponse, error) {
	endpoint := "/payout/payout_balance"

	responseBytes, err := c.Request("POST", endpoint, req)
	if err != nil {
		return nil, err
	}

	var balanceResp BalanceResponse
	if err := json.Unmarshal(responseBytes, &balanceResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	return &balanceResp, nil
}

// GetReceipt get receipt with specified parameters
func (c *Client) GetReceipt(req ReceiptRequest) (*ReceiptResponse, error) {
	endpoint := "/get-receipt"

	if err := req.Validate(); err != nil {
		return nil, err
	}

	responseBytes, err := c.Request("POST", endpoint, req)
	if err != nil {
		return nil, err
	}

	var receiptResp ReceiptResponse
	if err := json.Unmarshal(responseBytes, &receiptResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	return &receiptResp, nil
}

// VerifySignature checks if the provided signature is valid for the given data and secret key.
func (c *Client) VerifySignature(data, sign string) bool {
	h := hmac.New(sha512.New, []byte(c.SecretKey))
	h.Write([]byte(data))
	computedHMAC := h.Sum(nil)
	computedSignature := fmt.Sprintf("%x", computedHMAC)
	return hmac.Equal([]byte(computedSignature), []byte(sign))
}

// ParsePayload decodes base64 encoded data and unmarshals it into SuccessResponse struct
func (c *Client) ParsePayload(data string) (*StatusResponse, error) {
	decodedData, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64 data: %v", err)
	}

	var resp StatusResponse
	if err := json.Unmarshal(decodedData, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %v", err)
	}

	return &resp, nil
}
