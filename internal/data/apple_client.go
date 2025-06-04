package data

import (
	"context"
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/golang-jwt/jwt/v5"

	"gitlab.calendaria.team/services/utils/v4/config"
)

const (
	AppleStoreBaseURLProduction = "https://api.storekit.itunes.apple.com"
	AppleStoreBaseURLSandbox    = "https://api.storekit-sandbox.itunes.apple.com"
)

type AppleStoreClient interface {
	GetTransaction(ctx context.Context, transactionID string) (*AppleStoreResponse, error)
	ValidateTransaction(ctx context.Context, transactionID string) (*JWSTransaction, error)
	GetTransactionHistory(ctx context.Context, originalTransactionID string) (*AppleStoreHistoryResponse, error)
}

type appleStoreClient struct {
	httpClient  *http.Client
	environment Environment
	keyID       string
	issuerID    string
	bundleID    string
	privateKey  *ecdsa.PrivateKey
	jwtParser   JWTParser
	log         *log.Helper
}

type AppleStoreClientConfig struct {
	Environment Environment
	KeyID       string
	IssuerID    string
	BundleID    string
	PrivateKey  string // PEM-encoded private key
}

func NewAppleStoreConfig(config config.IConfig) (*AppleStoreClientConfig, error) {
	c := &AppleStoreClientConfig{}
	env, err := config.GetValue("APPLE_STORE_ENVIRONMENT")
	if err != nil {
		c.Environment = Sandbox
	}

	if env == "sandbox" {
		c.Environment = Sandbox
	}
	if env == "production" {
		c.Environment = Production
	}

	appStoreSecrets, err := config.ReadSecretsFor(context.Background(), "apple_store")
	if err != nil {
		return nil, fmt.Errorf("failed to read apple store secrets: %w", err)
	}

	if appStoreSecrets == nil {
		return nil, fmt.Errorf("apple store secrets not found")
	}

	keyId, ok := appStoreSecrets["key_id"]
	if !ok {
		return nil, fmt.Errorf("apple store key_id not found in secrets")
	}

	if keyId == "" {
		return nil, fmt.Errorf("apple store key_id is empty")
	}

	c.KeyID = keyId.(string)

	issuerId, ok := appStoreSecrets["issuer_id"]
	if !ok {
		return nil, fmt.Errorf("apple store issuer_id not found in secrets")
	}

	if issuerId == "" {
		return nil, fmt.Errorf("apple store issuer_id is empty")
	}

	c.IssuerID = issuerId.(string)

	bundleId, ok := appStoreSecrets["bundle_id"]
	if !ok {
		return nil, fmt.Errorf("apple store bundle_id not found in secrets")
	}

	if bundleId == "" {
		return nil, fmt.Errorf("apple store bundle_id is empty")
	}

	c.BundleID = bundleId.(string)

	privateKey, ok := appStoreSecrets["private_key"]
	if !ok {
		return nil, fmt.Errorf("apple store private_key not found in secrets")
	}

	if privateKey == "" {
		return nil, fmt.Errorf("apple store private_key is empty")
	}

	c.PrivateKey = privateKey.(string)

	return c, nil
}

func NewAppleStoreClient(config config.IConfig, logger log.Logger) (
	AppleStoreClient, error,
) {
	jwtParser := NewDefaultJWTParser()

	appStoreConfig, err := NewAppleStoreConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create apple store config: %w", err)
	}

	block, _ := pem.Decode([]byte(appStoreConfig.PrivateKey))
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block containing private key")
	}

	privateKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	ecdsaKey, ok := privateKey.(*ecdsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("private key is not ECDSA key")
	}

	return &appleStoreClient{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		environment: appStoreConfig.Environment,
		keyID:       appStoreConfig.KeyID,
		issuerID:    appStoreConfig.IssuerID,
		bundleID:    appStoreConfig.BundleID,
		privateKey:  ecdsaKey,
		jwtParser:   jwtParser,
		log:         log.NewHelper(logger),
	}, nil
}

func (c *appleStoreClient) getBaseURL() string {
	if c.environment == Sandbox {
		return AppleStoreBaseURLSandbox
	}
	return AppleStoreBaseURLProduction
}

func (c *appleStoreClient) generateBearerToken() (string, error) {
	now := time.Now()

	claims := jwt.MapClaims{
		"iss": c.issuerID,
		"iat": now.Unix(),
		"exp": now.Add(1 * time.Hour).Unix(),
		"aud": "appstoreconnect-v1",
		"bid": c.bundleID,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	token.Header["kid"] = c.keyID

	tokenString, err := token.SignedString(c.privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign JWT token: %w", err)
	}

	return tokenString, nil
}

func (c *appleStoreClient) GetTransaction(ctx context.Context, transactionID string) (*AppleStoreResponse, error) {
	bearerToken, err := c.generateBearerToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate bearer token: %w", err)
	}

	url := fmt.Sprintf("%s/inApps/v1/transactions/%s", c.getBaseURL(), transactionID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+bearerToken)
	req.Header.Set("Content-Type", "application/json")

	c.log.WithContext(ctx).Infof("Making request to Apple Store API: %s", url)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		c.log.WithContext(ctx).Errorf("Apple Store API error: status=%d, body=%s", resp.StatusCode, string(body))
		return nil, fmt.Errorf("apple store API returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var response AppleStoreResponse
	if err = json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &response, nil
}

func (c *appleStoreClient) ValidateTransaction(ctx context.Context, transactionID string) (*JWSTransaction, error) {
	response, err := c.GetTransaction(ctx, transactionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction from Apple: %w", err)
	}

	token, err := c.jwtParser.ParseAppleSignedBody(response.SignedTransactionInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to parse signed transaction info: %w", err)
	}

	mapClaims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims format")
	}

	claimsBytes, err := json.Marshal(mapClaims)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal claims: %w", err)
	}

	var transaction JWSTransaction
	if err = json.Unmarshal(claimsBytes, &transaction); err != nil {
		return nil, fmt.Errorf("failed to unmarshal transaction: %w", err)
	}

	return &transaction, nil
}

func (c *appleStoreClient) GetTransactionHistory(
	ctx context.Context, originalTransactionID string,
) (*AppleStoreHistoryResponse, error) {
	bearerToken, err := c.generateBearerToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate bearer token: %w", err)
	}

	url := fmt.Sprintf("%s/inApps/v1/history/%s", c.getBaseURL(), originalTransactionID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+bearerToken)
	req.Header.Set("Content-Type", "application/json")

	c.log.WithContext(ctx).Infof("Making history request to Apple Store API: %s", url)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		c.log.WithContext(ctx).Errorf(
			"Apple Store History API error: status=%d, body=%s", resp.StatusCode, string(body),
		)
		return nil, fmt.Errorf("apple store History API returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var response AppleStoreHistoryResponse
	if err = json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &response, nil
}
