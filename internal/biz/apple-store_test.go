package biz

import (
	"context"
	"encoding/binary"
	"errors"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"gitlab.calendaria.team/services/finance/billing/ent"
	"gitlab.calendaria.team/services/finance/billing/ent/enum"
	"gitlab.calendaria.team/services/finance/billing/internal/data"
	"gitlab.calendaria.team/services/finance/billing/internal/data/mock"
)

func createTestUUID(tenantID, userID int64) string {
	u := make([]byte, 16)

	binary.BigEndian.PutUint64(u[:8], uint64(tenantID))
	binary.BigEndian.PutUint64(u[8:], uint64(userID))
	return uuid.UUID(u).String()
}

func TestAppleStoreUsecase_ProcessPayload(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockInvoices := mock.NewMockInvoicesRepo(ctrl)
	mockSubscriptions := mock.NewMockSubscriptionsRepo(ctrl)
	mockProduct := mock.NewMockProductRepo(ctrl)
	mockJwtParser := mock.NewMockJWTParser(ctrl)

	uc := NewAppleStoreUsecase(
		mockInvoices,
		mockSubscriptions,
		mockProduct,
		mockJwtParser,
		nil,
	)

	ctx := context.Background()

	testUserID := int64(1)
	testTenantID := int64(2)
	testTransactionID := "test_transaction_id"
	testAppID := "test_app_id"

	testSignedTransactionInfo := "test_jwt_token"
	testJWTClaims := jwt.MapClaims{
		"appAccountToken":       createTestUUID(testTenantID, testUserID),
		"productID":             "123",
		"purchaseDate":          int64(1640995200000), // 2022-01-01 00:00:00
		"expiresDate":           int64(1643673600000), // 2022-02-01 00:00:00
		"originalTransactionID": testTransactionID,
		"quantity":              int64(1),
		"price":                 int64(999),
		"type":                  "Auto-Renewable Subscription",
		"revocationDate":        int64(1642608000000), // 2022-01-20 00:00:00
	}
	testJWTToken := &jwt.Token{Claims: testJWTClaims}

	tests := []struct {
		name    string
		payload data.Payload
		setup   func()
		wantErr bool
	}{
		{
			name: "successful_new_subscription_processing",
			payload: data.Payload{
				NotificationType: data.TypeSubscribed,
				Data: struct {
					AppAppleID               int64            `json:"appAppleID,omitempty"`
					BundleID                 string           `json:"bundleID,omitempty"`
					BundleVersion            string           `json:"bundleVersion,omitempty"`
					ConsumptionRequestReason string           `json:"consumptionRequestReason,omitempty"`
					Environment              data.Environment `json:"environment,omitempty"`
					SignedRenewalInfo        string           `json:"signedRenewalInfo,omitempty"`
					SignedTransactionInfo    string           `json:"signedTransactionInfo,omitempty"`
					Status                   int64            `json:"status,omitempty"`
				}{
					SignedTransactionInfo: testSignedTransactionInfo,
				},
			},
			setup: func() {
				gomock.InOrder(
					mockJwtParser.EXPECT().
						ParseAppleSignedBody(testSignedTransactionInfo).
						Return(testJWTToken, nil),

					mockProduct.EXPECT().
						GetProduct(gomock.Any(), int64(123)).
						Return(
							&ent.Product{
								ID:    123,
								AppID: testAppID,
							}, nil,
						),

					mockSubscriptions.EXPECT().
						GetSubscriptionByOriginalAppleTransactionID(gomock.Any(), testTransactionID, false).
						Return(nil, &ent.NotFoundError{}),

					mockSubscriptions.EXPECT().
						CreateSubscription(
							gomock.Any(),
							testUserID,
							testTenantID,
							testAppID,
							data.SubscriptionDto{
								UserID:    testUserID,
								TenantID:  testTenantID,
								AppID:     testAppID,
								ProductID: int64(123),
							},
						).
						Return(
							&ent.Subscriptions{
								ID:       1,
								UserID:   testUserID,
								TenantID: testTenantID,
								AppID:    testAppID,
							}, nil,
						),

					mockInvoices.EXPECT().
						CreateInvoice(gomock.Any(), gomock.Any()).
						DoAndReturn(
							func(_ context.Context, dto data.InvoiceDto) (*ent.Invoice, error) {
								assert.Equal(t, testUserID, dto.UserID)
								assert.Equal(t, testTenantID, dto.TenantID)
								assert.Equal(t, testAppID, dto.AppID)
								assert.Equal(t, int64(123), dto.ProductID)
								assert.Equal(t, int64(1), dto.Amount)
								assert.Equal(t, enum.Paid, dto.Status)
								assert.Equal(t, int64(1), dto.SubscriptionID)
								assert.Equal(t, testTransactionID, *dto.AppleStoreTransactionID)
								return &ent.Invoice{}, nil
							},
						),
				)
			},
			wantErr: false,
		},
		{
			name: "subscription_revocation_processing",
			payload: data.Payload{
				NotificationType: data.TypeRevoke,
				Data: struct {
					AppAppleID               int64            `json:"appAppleID,omitempty"`
					BundleID                 string           `json:"bundleID,omitempty"`
					BundleVersion            string           `json:"bundleVersion,omitempty"`
					ConsumptionRequestReason string           `json:"consumptionRequestReason,omitempty"`
					Environment              data.Environment `json:"environment,omitempty"`
					SignedRenewalInfo        string           `json:"signedRenewalInfo,omitempty"`
					SignedTransactionInfo    string           `json:"signedTransactionInfo,omitempty"`
					Status                   int64            `json:"status,omitempty"`
				}{
					SignedTransactionInfo: testSignedTransactionInfo,
				},
			},
			setup: func() {
				gomock.InOrder(
					mockJwtParser.EXPECT().
						ParseAppleSignedBody(testSignedTransactionInfo).
						Return(testJWTToken, nil),

					mockSubscriptions.EXPECT().
						GetSubscriptionByOriginalAppleTransactionID(gomock.Any(), testTransactionID, false).
						Return(
							&ent.Subscriptions{
								ID:       1,
								UserID:   testUserID,
								TenantID: testTenantID,
								AppID:    testAppID,
							}, nil,
						),

					mockSubscriptions.EXPECT().
						RevokeActiveSubscription(gomock.Any(), int64(1), gomock.Any()).
						Return(nil),
				)
			},
			wantErr: false,
		},
		{
			name: "subscription_renewal_processing",
			payload: data.Payload{
				NotificationType: data.TypeDidRenew,
				Data: struct {
					AppAppleID               int64            `json:"appAppleID,omitempty"`
					BundleID                 string           `json:"bundleID,omitempty"`
					BundleVersion            string           `json:"bundleVersion,omitempty"`
					ConsumptionRequestReason string           `json:"consumptionRequestReason,omitempty"`
					Environment              data.Environment `json:"environment,omitempty"`
					SignedRenewalInfo        string           `json:"signedRenewalInfo,omitempty"`
					SignedTransactionInfo    string           `json:"signedTransactionInfo,omitempty"`
					Status                   int64            `json:"status,omitempty"`
				}{
					SignedTransactionInfo: testSignedTransactionInfo,
				},
			},
			setup: func() {
				gomock.InOrder(
					mockJwtParser.EXPECT().
						ParseAppleSignedBody(testSignedTransactionInfo).
						Return(testJWTToken, nil),

					mockProduct.EXPECT().
						GetProduct(gomock.Any(), int64(123)).
						Return(
							&ent.Product{
								ID:    123,
								AppID: testAppID,
							}, nil,
						),

					mockSubscriptions.EXPECT().
						GetSubscriptionByOriginalAppleTransactionID(gomock.Any(), testTransactionID, false).
						Return(
							&ent.Subscriptions{
								ID:       1,
								UserID:   testUserID,
								TenantID: testTenantID,
								AppID:    testAppID,
							}, nil,
						),

					mockInvoices.EXPECT().
						CreateInvoice(gomock.Any(), gomock.Any()).
						Return(&ent.Invoice{}, nil),
				)
			},
			wantErr: false,
		},
		{
			name: "offer_redeemed_processing",
			payload: data.Payload{
				NotificationType: data.TypeOfferRedeemed,
				Data: struct {
					AppAppleID               int64            `json:"appAppleID,omitempty"`
					BundleID                 string           `json:"bundleID,omitempty"`
					BundleVersion            string           `json:"bundleVersion,omitempty"`
					ConsumptionRequestReason string           `json:"consumptionRequestReason,omitempty"`
					Environment              data.Environment `json:"environment,omitempty"`
					SignedRenewalInfo        string           `json:"signedRenewalInfo,omitempty"`
					SignedTransactionInfo    string           `json:"signedTransactionInfo,omitempty"`
					Status                   int64            `json:"status,omitempty"`
				}{
					SignedTransactionInfo: testSignedTransactionInfo,
				},
			},
			setup: func() {
				gomock.InOrder(
					mockJwtParser.EXPECT().
						ParseAppleSignedBody(testSignedTransactionInfo).
						Return(testJWTToken, nil),

					mockProduct.EXPECT().
						GetProduct(gomock.Any(), int64(123)).
						Return(
							&ent.Product{
								ID:    123,
								AppID: testAppID,
							}, nil,
						),

					mockSubscriptions.EXPECT().
						GetSubscriptionByOriginalAppleTransactionID(gomock.Any(), testTransactionID, false).
						Return(
							&ent.Subscriptions{
								ID:       1,
								UserID:   testUserID,
								TenantID: testTenantID,
								AppID:    testAppID,
							}, nil,
						),

					mockInvoices.EXPECT().
						CreateInvoice(gomock.Any(), gomock.Any()).
						Return(&ent.Invoice{}, nil),
				)
			},
			wantErr: false,
		},
		{
			name: "subscription_expiration_processing",
			payload: data.Payload{
				NotificationType: data.TypeExpired,
				Data: struct {
					AppAppleID               int64            `json:"appAppleID,omitempty"`
					BundleID                 string           `json:"bundleID,omitempty"`
					BundleVersion            string           `json:"bundleVersion,omitempty"`
					ConsumptionRequestReason string           `json:"consumptionRequestReason,omitempty"`
					Environment              data.Environment `json:"environment,omitempty"`
					SignedRenewalInfo        string           `json:"signedRenewalInfo,omitempty"`
					SignedTransactionInfo    string           `json:"signedTransactionInfo,omitempty"`
					Status                   int64            `json:"status,omitempty"`
				}{
					SignedTransactionInfo: testSignedTransactionInfo,
				},
			},
			setup: func() {
				gomock.InOrder(
					mockJwtParser.EXPECT().
						ParseAppleSignedBody(testSignedTransactionInfo).
						Return(testJWTToken, nil),

					mockSubscriptions.EXPECT().
						GetSubscriptionByOriginalAppleTransactionID(gomock.Any(), testTransactionID, false).
						Return(
							&ent.Subscriptions{
								ID:       1,
								UserID:   testUserID,
								TenantID: testTenantID,
								AppID:    testAppID,
							}, nil,
						),
				)
			},
			wantErr: false,
		},
		{
			name: "failed_renewal_subscription_processing",
			payload: data.Payload{
				NotificationType: data.TypeDidFailToRenew,
				Data: struct {
					AppAppleID               int64            `json:"appAppleID,omitempty"`
					BundleID                 string           `json:"bundleID,omitempty"`
					BundleVersion            string           `json:"bundleVersion,omitempty"`
					ConsumptionRequestReason string           `json:"consumptionRequestReason,omitempty"`
					Environment              data.Environment `json:"environment,omitempty"`
					SignedRenewalInfo        string           `json:"signedRenewalInfo,omitempty"`
					SignedTransactionInfo    string           `json:"signedTransactionInfo,omitempty"`
					Status                   int64            `json:"status,omitempty"`
				}{
					SignedTransactionInfo: testSignedTransactionInfo,
				},
			},
			setup: func() {
				gomock.InOrder(
					mockJwtParser.EXPECT().
						ParseAppleSignedBody(testSignedTransactionInfo).
						Return(testJWTToken, nil),

					mockSubscriptions.EXPECT().
						GetSubscriptionByOriginalAppleTransactionID(gomock.Any(), testTransactionID, false).
						Return(
							&ent.Subscriptions{
								ID:       1,
								UserID:   testUserID,
								TenantID: testTenantID,
								AppID:    testAppID,
							}, nil,
						),
				)
			},
			wantErr: false,
		},
		{
			name: "grace_period_expiration_processing",
			payload: data.Payload{
				NotificationType: data.TypeGracePeriodExpired,
				Data: struct {
					AppAppleID               int64            `json:"appAppleID,omitempty"`
					BundleID                 string           `json:"bundleID,omitempty"`
					BundleVersion            string           `json:"bundleVersion,omitempty"`
					ConsumptionRequestReason string           `json:"consumptionRequestReason,omitempty"`
					Environment              data.Environment `json:"environment,omitempty"`
					SignedRenewalInfo        string           `json:"signedRenewalInfo,omitempty"`
					SignedTransactionInfo    string           `json:"signedTransactionInfo,omitempty"`
					Status                   int64            `json:"status,omitempty"`
				}{
					SignedTransactionInfo: testSignedTransactionInfo,
				},
			},
			setup: func() {
				gomock.InOrder(
					mockJwtParser.EXPECT().
						ParseAppleSignedBody(testSignedTransactionInfo).
						Return(testJWTToken, nil),

					mockSubscriptions.EXPECT().
						GetSubscriptionByOriginalAppleTransactionID(gomock.Any(), testTransactionID, false).
						Return(
							&ent.Subscriptions{
								ID:       1,
								UserID:   testUserID,
								TenantID: testTenantID,
								AppID:    testAppID,
							}, nil,
						),
				)
			},
			wantErr: false,
		},
		{
			name: "uuid_parsing_error",
			payload: data.Payload{
				NotificationType: data.TypeSubscribed,
				Data: struct {
					AppAppleID               int64            `json:"appAppleID,omitempty"`
					BundleID                 string           `json:"bundleID,omitempty"`
					BundleVersion            string           `json:"bundleVersion,omitempty"`
					ConsumptionRequestReason string           `json:"consumptionRequestReason,omitempty"`
					Environment              data.Environment `json:"environment,omitempty"`
					SignedRenewalInfo        string           `json:"signedRenewalInfo,omitempty"`
					SignedTransactionInfo    string           `json:"signedTransactionInfo,omitempty"`
					Status                   int64            `json:"status,omitempty"`
				}{
					SignedTransactionInfo: testSignedTransactionInfo,
				},
			},
			setup: func() {
				mockJwtParser.EXPECT().
					ParseAppleSignedBody(testSignedTransactionInfo).
					Return(nil, errors.New("invalid token"))
			},
			wantErr: true,
		},
		{
			name: "error_mapping_claims_to_mapclaims",
			payload: data.Payload{
				NotificationType: data.TypeSubscribed,
				Data: struct {
					AppAppleID               int64            `json:"appAppleID,omitempty"`
					BundleID                 string           `json:"bundleID,omitempty"`
					BundleVersion            string           `json:"bundleVersion,omitempty"`
					ConsumptionRequestReason string           `json:"consumptionRequestReason,omitempty"`
					Environment              data.Environment `json:"environment,omitempty"`
					SignedRenewalInfo        string           `json:"signedRenewalInfo,omitempty"`
					SignedTransactionInfo    string           `json:"signedTransactionInfo,omitempty"`
					Status                   int64            `json:"status,omitempty"`
				}{
					SignedTransactionInfo: testSignedTransactionInfo,
				},
			},
			setup: func() {
				mockJwtParser.EXPECT().
					ParseAppleSignedBody(testSignedTransactionInfo).
					Return(&jwt.Token{Claims: jwt.MapClaims{"invalid": true}}, nil)
			},
			wantErr: true,
		},
		{
			name: "product_id_parsing_error",
			payload: data.Payload{
				NotificationType: data.TypeSubscribed,
				Data: struct {
					AppAppleID               int64            `json:"appAppleID,omitempty"`
					BundleID                 string           `json:"bundleID,omitempty"`
					BundleVersion            string           `json:"bundleVersion,omitempty"`
					ConsumptionRequestReason string           `json:"consumptionRequestReason,omitempty"`
					Environment              data.Environment `json:"environment,omitempty"`
					SignedRenewalInfo        string           `json:"signedRenewalInfo,omitempty"`
					SignedTransactionInfo    string           `json:"signedTransactionInfo,omitempty"`
					Status                   int64            `json:"status,omitempty"`
				}{
					SignedTransactionInfo: testSignedTransactionInfo,
				},
			},
			setup: func() {
				invalidJWTClaims := jwt.MapClaims{
					"appAccountToken":       createTestUUID(testTenantID, testUserID),
					"productID":             "invalid_product_id",
					"purchaseDate":          int64(1640995200000),
					"expiresDate":           int64(1643673600000),
					"originalTransactionID": testTransactionID,
					"quantity":              int64(1),
					"price":                 int64(999),
				}
				invalidJWTToken := &jwt.Token{Claims: invalidJWTClaims}

				mockJwtParser.EXPECT().
					ParseAppleSignedBody(testSignedTransactionInfo).
					Return(invalidJWTToken, nil)
			},
			wantErr: true,
		},
		{
			name: "error_getting_product",
			payload: data.Payload{
				NotificationType: data.TypeSubscribed,
				Data: struct {
					AppAppleID               int64            `json:"appAppleID,omitempty"`
					BundleID                 string           `json:"bundleID,omitempty"`
					BundleVersion            string           `json:"bundleVersion,omitempty"`
					ConsumptionRequestReason string           `json:"consumptionRequestReason,omitempty"`
					Environment              data.Environment `json:"environment,omitempty"`
					SignedRenewalInfo        string           `json:"signedRenewalInfo,omitempty"`
					SignedTransactionInfo    string           `json:"signedTransactionInfo,omitempty"`
					Status                   int64            `json:"status,omitempty"`
				}{
					SignedTransactionInfo: testSignedTransactionInfo,
				},
			},
			setup: func() {
				gomock.InOrder(
					mockJwtParser.EXPECT().
						ParseAppleSignedBody(testSignedTransactionInfo).
						Return(testJWTToken, nil),

					mockProduct.EXPECT().
						GetProduct(gomock.Any(), int64(123)).
						Return(nil, errors.New("product not found")),
				)
			},
			wantErr: true,
		},
		{
			name: "error_getting_subscription_not_notfound",
			payload: data.Payload{
				NotificationType: data.TypeSubscribed,
				Data: struct {
					AppAppleID               int64            `json:"appAppleID,omitempty"`
					BundleID                 string           `json:"bundleID,omitempty"`
					BundleVersion            string           `json:"bundleVersion,omitempty"`
					ConsumptionRequestReason string           `json:"consumptionRequestReason,omitempty"`
					Environment              data.Environment `json:"environment,omitempty"`
					SignedRenewalInfo        string           `json:"signedRenewalInfo,omitempty"`
					SignedTransactionInfo    string           `json:"signedTransactionInfo,omitempty"`
					Status                   int64            `json:"status,omitempty"`
				}{
					SignedTransactionInfo: testSignedTransactionInfo,
				},
			},
			setup: func() {
				gomock.InOrder(
					mockJwtParser.EXPECT().
						ParseAppleSignedBody(testSignedTransactionInfo).
						Return(testJWTToken, nil),

					mockProduct.EXPECT().
						GetProduct(gomock.Any(), int64(123)).
						Return(
							&ent.Product{
								ID:    123,
								AppID: testAppID,
							}, nil,
						),

					mockSubscriptions.EXPECT().
						GetSubscriptionByOriginalAppleTransactionID(gomock.Any(), testTransactionID, false).
						Return(nil, errors.New("database error")),
				)
			},
			wantErr: true,
		},
		{
			name: "error_creating_subscription",
			payload: data.Payload{
				NotificationType: data.TypeSubscribed,
				Data: struct {
					AppAppleID               int64            `json:"appAppleID,omitempty"`
					BundleID                 string           `json:"bundleID,omitempty"`
					BundleVersion            string           `json:"bundleVersion,omitempty"`
					ConsumptionRequestReason string           `json:"consumptionRequestReason,omitempty"`
					Environment              data.Environment `json:"environment,omitempty"`
					SignedRenewalInfo        string           `json:"signedRenewalInfo,omitempty"`
					SignedTransactionInfo    string           `json:"signedTransactionInfo,omitempty"`
					Status                   int64            `json:"status,omitempty"`
				}{
					SignedTransactionInfo: testSignedTransactionInfo,
				},
			},
			setup: func() {
				gomock.InOrder(
					mockJwtParser.EXPECT().
						ParseAppleSignedBody(testSignedTransactionInfo).
						Return(testJWTToken, nil),

					mockProduct.EXPECT().
						GetProduct(gomock.Any(), int64(123)).
						Return(
							&ent.Product{
								ID:    123,
								AppID: testAppID,
							}, nil,
						),

					mockSubscriptions.EXPECT().
						GetSubscriptionByOriginalAppleTransactionID(gomock.Any(), testTransactionID, false).
						Return(nil, &ent.NotFoundError{}),

					mockSubscriptions.EXPECT().
						CreateSubscription(
							gomock.Any(),
							testUserID,
							testTenantID,
							testAppID,
							data.SubscriptionDto{
								UserID:    testUserID,
								TenantID:  testTenantID,
								AppID:     testAppID,
								ProductID: int64(123),
							},
						).
						Return(nil, errors.New("failed to create subscription")),
				)
			},
			wantErr: true,
		},
		{
			name: "error_creating_invoice",
			payload: data.Payload{
				NotificationType: data.TypeSubscribed,
				Data: struct {
					AppAppleID               int64            `json:"appAppleID,omitempty"`
					BundleID                 string           `json:"bundleID,omitempty"`
					BundleVersion            string           `json:"bundleVersion,omitempty"`
					ConsumptionRequestReason string           `json:"consumptionRequestReason,omitempty"`
					Environment              data.Environment `json:"environment,omitempty"`
					SignedRenewalInfo        string           `json:"signedRenewalInfo,omitempty"`
					SignedTransactionInfo    string           `json:"signedTransactionInfo,omitempty"`
					Status                   int64            `json:"status,omitempty"`
				}{
					SignedTransactionInfo: testSignedTransactionInfo,
				},
			},
			setup: func() {
				gomock.InOrder(
					mockJwtParser.EXPECT().
						ParseAppleSignedBody(testSignedTransactionInfo).
						Return(testJWTToken, nil),

					mockProduct.EXPECT().
						GetProduct(gomock.Any(), int64(123)).
						Return(
							&ent.Product{
								ID:    123,
								AppID: testAppID,
							}, nil,
						),

					mockSubscriptions.EXPECT().
						GetSubscriptionByOriginalAppleTransactionID(gomock.Any(), testTransactionID, false).
						Return(
							&ent.Subscriptions{
								ID:       1,
								UserID:   testUserID,
								TenantID: testTenantID,
								AppID:    testAppID,
							}, nil,
						),

					mockInvoices.EXPECT().
						CreateInvoice(gomock.Any(), gomock.Any()).
						Return(nil, errors.New("failed to create invoice")),
				)
			},
			wantErr: true,
		},
		{
			name: "error_revoking_active_subscription",
			payload: data.Payload{
				NotificationType: data.TypeRevoke,
				Data: struct {
					AppAppleID               int64            `json:"appAppleID,omitempty"`
					BundleID                 string           `json:"bundleID,omitempty"`
					BundleVersion            string           `json:"bundleVersion,omitempty"`
					ConsumptionRequestReason string           `json:"consumptionRequestReason,omitempty"`
					Environment              data.Environment `json:"environment,omitempty"`
					SignedRenewalInfo        string           `json:"signedRenewalInfo,omitempty"`
					SignedTransactionInfo    string           `json:"signedTransactionInfo,omitempty"`
					Status                   int64            `json:"status,omitempty"`
				}{
					SignedTransactionInfo: testSignedTransactionInfo,
				},
			},
			setup: func() {
				gomock.InOrder(
					mockJwtParser.EXPECT().
						ParseAppleSignedBody(testSignedTransactionInfo).
						Return(testJWTToken, nil),

					mockSubscriptions.EXPECT().
						GetSubscriptionByOriginalAppleTransactionID(gomock.Any(), testTransactionID, false).
						Return(
							&ent.Subscriptions{
								ID:       1,
								UserID:   testUserID,
								TenantID: testTenantID,
								AppID:    testAppID,
							}, nil,
						),

					mockSubscriptions.EXPECT().
						RevokeActiveSubscription(gomock.Any(), int64(1), gomock.Any()).
						Return(errors.New("failed to revoke subscription")),
				)
			},
			wantErr: true,
		},
		{
			name: "error_marshaling_json_claims",
			payload: data.Payload{
				NotificationType: data.TypeSubscribed,
				Data: struct {
					AppAppleID               int64            `json:"appAppleID,omitempty"`
					BundleID                 string           `json:"bundleID,omitempty"`
					BundleVersion            string           `json:"bundleVersion,omitempty"`
					ConsumptionRequestReason string           `json:"consumptionRequestReason,omitempty"`
					Environment              data.Environment `json:"environment,omitempty"`
					SignedRenewalInfo        string           `json:"signedRenewalInfo,omitempty"`
					SignedTransactionInfo    string           `json:"signedTransactionInfo,omitempty"`
					Status                   int64            `json:"status,omitempty"`
				}{
					SignedTransactionInfo: testSignedTransactionInfo,
				},
			},
			setup: func() {
				mockJwtParser.EXPECT().
					ParseAppleSignedBody(testSignedTransactionInfo).
					Return(
						&jwt.Token{
							Claims: jwt.MapClaims{
								"appAccountToken": func() {},
							},
						}, nil,
					)
			},
			wantErr: true,
		},
		{
			name: "error_unmarshaling_json_claims",
			payload: data.Payload{
				NotificationType: data.TypeSubscribed,
				Data: struct {
					AppAppleID               int64            `json:"appAppleID,omitempty"`
					BundleID                 string           `json:"bundleID,omitempty"`
					BundleVersion            string           `json:"bundleVersion,omitempty"`
					ConsumptionRequestReason string           `json:"consumptionRequestReason,omitempty"`
					Environment              data.Environment `json:"environment,omitempty"`
					SignedRenewalInfo        string           `json:"signedRenewalInfo,omitempty"`
					SignedTransactionInfo    string           `json:"signedTransactionInfo,omitempty"`
					Status                   int64            `json:"status,omitempty"`
				}{
					SignedTransactionInfo: testSignedTransactionInfo,
				},
			},
			setup: func() {
				// Создаем некорректный JSON
				invalidClaims := jwt.MapClaims{
					"appAccountToken":       123, // Неверный тип (должна быть строка)
					"productID":             "123",
					"purchaseDate":          int64(1640995200000),
					"expiresDate":           int64(1643673600000),
					"originalTransactionID": testTransactionID,
				}

				mockJwtParser.EXPECT().
					ParseAppleSignedBody(testSignedTransactionInfo).
					Return(&jwt.Token{Claims: invalidClaims}, nil)
			},
			wantErr: true,
		},
		{
			name: "error_getting_subscription_in_process_expired",
			payload: data.Payload{
				NotificationType: data.TypeExpired,
				Data: struct {
					AppAppleID               int64            `json:"appAppleID,omitempty"`
					BundleID                 string           `json:"bundleID,omitempty"`
					BundleVersion            string           `json:"bundleVersion,omitempty"`
					ConsumptionRequestReason string           `json:"consumptionRequestReason,omitempty"`
					Environment              data.Environment `json:"environment,omitempty"`
					SignedRenewalInfo        string           `json:"signedRenewalInfo,omitempty"`
					SignedTransactionInfo    string           `json:"signedTransactionInfo,omitempty"`
					Status                   int64            `json:"status,omitempty"`
				}{
					SignedTransactionInfo: testSignedTransactionInfo,
				},
			},
			setup: func() {
				gomock.InOrder(
					mockJwtParser.EXPECT().
						ParseAppleSignedBody(testSignedTransactionInfo).
						Return(testJWTToken, nil),

					mockSubscriptions.EXPECT().
						GetSubscriptionByOriginalAppleTransactionID(gomock.Any(), testTransactionID, false).
						Return(nil, errors.New("database error in processExpired")),
				)
			},
			wantErr: true,
		},
		{
			name: "error_marshaling_json_claims_in_process_expired",
			payload: data.Payload{
				NotificationType: data.TypeExpired,
				Data: struct {
					AppAppleID               int64            `json:"appAppleID,omitempty"`
					BundleID                 string           `json:"bundleID,omitempty"`
					BundleVersion            string           `json:"bundleVersion,omitempty"`
					ConsumptionRequestReason string           `json:"consumptionRequestReason,omitempty"`
					Environment              data.Environment `json:"environment,omitempty"`
					SignedRenewalInfo        string           `json:"signedRenewalInfo,omitempty"`
					SignedTransactionInfo    string           `json:"signedTransactionInfo,omitempty"`
					Status                   int64            `json:"status,omitempty"`
				}{
					SignedTransactionInfo: testSignedTransactionInfo,
				},
			},
			setup: func() {
				mockJwtParser.EXPECT().
					ParseAppleSignedBody(testSignedTransactionInfo).
					Return(
						&jwt.Token{
							Claims: jwt.MapClaims{
								"appAccountToken": func() {},
							},
						}, nil,
					)
			},
			wantErr: true,
		},
		{
			name: "error_unmarshaling_json_claims_in_process_expired",
			payload: data.Payload{
				NotificationType: data.TypeExpired,
				Data: struct {
					AppAppleID               int64            `json:"appAppleID,omitempty"`
					BundleID                 string           `json:"bundleID,omitempty"`
					BundleVersion            string           `json:"bundleVersion,omitempty"`
					ConsumptionRequestReason string           `json:"consumptionRequestReason,omitempty"`
					Environment              data.Environment `json:"environment,omitempty"`
					SignedRenewalInfo        string           `json:"signedRenewalInfo,omitempty"`
					SignedTransactionInfo    string           `json:"signedTransactionInfo,omitempty"`
					Status                   int64            `json:"status,omitempty"`
				}{
					SignedTransactionInfo: testSignedTransactionInfo,
				},
			},
			setup: func() {
				// Создаем некорректный JSON
				invalidClaims := jwt.MapClaims{
					"appAccountToken":       123, // Неверный тип (должна быть строка)
					"productID":             "123",
					"purchaseDate":          int64(1640995200000),
					"expiresDate":           int64(1643673600000),
					"originalTransactionID": testTransactionID,
				}

				mockJwtParser.EXPECT().
					ParseAppleSignedBody(testSignedTransactionInfo).
					Return(&jwt.Token{Claims: invalidClaims}, nil)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				tt.setup()
				err := uc.ProcessPayload(ctx, tt.payload)
				if tt.wantErr {
					assert.Error(t, err)
					return
				}
				assert.NoError(t, err)
			},
		)
	}
}

func TestDefaultJWTParser_ParseAppleSignedBody(t *testing.T) {
	parser := data.NewDefaultJWTParser()

	// Тест на невалидный токен
	_, err := parser.ParseAppleSignedBody("invalid.token")
	assert.Error(t, err)
}
