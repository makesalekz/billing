package onevisionpay

import (
	"errors"
)

type SuccessResponse struct {
	Success   bool   `json:"success"`
	PaymentID int64  `json:"payment_id,omitempty"`
	Data      string `json:"data,omitempty"`
	Sign      string `json:"sign,omitempty"`
}

type ErrorResponse struct {
	Success   bool   `json:"success"`
	ErrorCode string `json:"error_code,omitempty"`
	ErrorMsg  string `json:"error_msg,omitempty"`
}

// Общая структура для унификации обработки
type APIResponse struct {
	Success   bool   `json:"success"`
	PaymentID string `json:"payment_id,omitempty"`
	Data      string `json:"data,omitempty"`
	Sign      string `json:"sign,omitempty"`
	ErrorCode string `json:"error_code,omitempty"`
	ErrorMsg  string `json:"error_msg,omitempty"`
}

// PaymentRequest represents the request payload for initiating a payment.
// Required fields: Amount, Currency, OrderID, Description
// Optional fields: PaymentType, PaymentMethod, Items, UserID, Email, Phone, SuccessURL, FailureURL, CallbackURL, PaymentLifetime, CreateRecurrentProfile, RecurrentProfileLifetime, Lang, ExtraParams
type PaymentRequest struct {
	Amount                   int64                  `json:"amount"`                     // Required: Payment amount
	Currency                 string                 `json:"currency"`                   // Required: Currency code (e.g., USD, EUR)
	OrderID                  string                 `json:"order_id"`                   // Required: Unique order identifier
	Description              string                 `json:"description"`                // Required: Payment description
	PaymentType              PaymentType            `json:"payment_type"`               // Optional: Type of payment
	PaymentMethod            PaymentMethod          `json:"payment_method"`             // Optional: Method of payment
	Items                    []PaymentItem          `json:"items"`                      // Optional: List of payment items
	UserID                   string                 `json:"user_id"`                    // Optional: User identifier
	Email                    string                 `json:"email"`                      // Optional: User email address
	Phone                    string                 `json:"phone"`                      // Optional: User phone number
	SuccessURL               string                 `json:"success_url"`                // Optional: URL for successful payment redirection
	FailureURL               string                 `json:"failure_url"`                // Optional: URL for failed payment redirection
	CallbackURL              string                 `json:"callback_url"`               // Optional: URL for payment status callback
	PaymentLifetime          int                    `json:"payment_lifetime"`           // Optional: Payment lifetime in seconds
	CreateRecurrentProfile   bool                   `json:"create_recurrent_profile"`   // Optional: Flag to create a recurrent profile
	RecurrentProfileLifetime int                    `json:"recurrent_profile_lifetime"` // Optional: Lifetime of the recurrent profile
	Lang                     string                 `json:"lang"`                       // Optional: Language code (e.g., en, ru)
	ExtraParams              map[string]interface{} `json:"extra_params"`               // Optional: Additional parameters
}

// Validate checks if the required fields of PaymentRequest are set and have valid values.
func (pr *PaymentRequest) Validate() error {
	if pr.Amount <= 0 {
		return errors.New("amount must be greater than zero")
	}
	if pr.Currency == "" {
		return errors.New("currency is required")
	}
	if len(pr.Currency) != 3 {
		return errors.New("currency must be a valid 3-letter ISO code")
	}
	if pr.OrderID == "" {
		return errors.New("order_id is required")
	}
	if pr.Description == "" {
		return errors.New("description is required")
	}
	return nil
}

// PaymentItem represents an individual item in a payment.
// Required fields: MerchantID, ServiceID, Name, Quantity
// Optional fields: MerchantName, AmountOnePcs, AmountSum
type PaymentItem struct {
	MerchantID   string `json:"merchant_id"`    // Required: Merchant identifier
	ServiceID    string `json:"service_id"`     // Required: Service identifier
	MerchantName string `json:"merchant_name"`  // Optional: Merchant name
	Name         string `json:"name"`           // Required: Item name
	Quantity     int    `json:"quantity"`       // Required: Quantity of the item
	AmountOnePcs int64  `json:"amount_one_pcs"` // Optional: Amount per one piece
	AmountSum    int64  `json:"amount_sum"`     // Optional: Total amount for the item
}

// Validate checks if the required fields of PaymentItem are set and have valid values.
func (pi *PaymentItem) Validate() error {
	if pi.MerchantID == "" {
		return errors.New("merchant_id is required")
	}
	if pi.ServiceID == "" {
		return errors.New("service_id is required")
	}
	if pi.Name == "" {
		return errors.New("name is required")
	}
	if pi.Quantity <= 0 {
		return errors.New("quantity must be greater than zero")
	}
	if pi.AmountOnePcs < 0 {
		return errors.New("amount_one_pcs must be non-negative")
	}
	if pi.AmountSum < 0 {
		return errors.New("amount_sum must be non-negative")
	}
	return nil
}

// RefundRequest represents the request payload for refunding a payment.
// Required fields: PaymentID, Amount
// Optional fields: Description
// TestMode values: 1 (success), 2 (error)
type RefundRequest struct {
	PaymentID   int64  `json:"payment_id"`  // Required: Payment identifier
	Amount      int64  `json:"amount"`      // Required: Amount to refund
	Description string `json:"description"` // Optional: Description of the refund
	TestMode    *int32 `json:"test_mode"`   // Optional: Flag to enable test mode
}

// Validate checks if the required fields of RefundRequest are set and have valid values.
func (rr *RefundRequest) Validate() error {
	if rr.PaymentID == 0 {
		return errors.New("payment_id is required")
	}
	if rr.Amount <= 0 {
		return errors.New("amount must be greater than zero")
	}
	return nil
}

// CancelRequest represents the request payload for cancelling a payment.
// Required fields: PaymentID
// Optional fields: Description
type CancelRequest struct {
	PaymentID   int64  `json:"payment_id"`  // Required: Payment identifier
	Description string `json:"description"` // Optional: Description of the cancellation
}

// Validate checks if the required fields of CancelRequest are set.
func (cr *CancelRequest) Validate() error {
	if cr.PaymentID == 0 {
		return errors.New("payment_id is required")
	}
	return nil
}

// ClearingRequest represents the request payload for clearing a payment.
// Required fields: PaymentID, Amount
type ClearingRequest struct {
	PaymentID int64 `json:"payment_id"` // Required: Payment identifier
	Amount    int64 `json:"amount"`     // Required: Amount to clear
}

// Validate checks if the required fields of ClearingRequest are set and have valid values.
func (clr *ClearingRequest) Validate() error {
	if clr.PaymentID == 0 {
		return errors.New("payment_id is required")
	}
	if clr.Amount <= 0 {
		return errors.New("amount must be greater than zero")
	}
	return nil
}

// ConfirmRequest represents the request payload for confirming a payment.
// Required fields: PaymentID, ConfirmID, ConfirmRes, ConfirmCode, ConfirmOTP
type ConfirmRequest struct {
	PaymentID   int64  `json:"payment_id"`   // Required: Payment identifier
	ConfirmID   string `json:"confirm_id"`   // Required: Confirmation identifier
	ConfirmRes  string `json:"confirm_res"`  // Required: Confirmation result
	ConfirmCode string `json:"confirm_code"` // Required: Confirmation code
	ConfirmOTP  string `json:"confirm_otp"`  // Required: One-time password (OTP)
}

// Validate checks if the required fields of ConfirmRequest are set.
func (cr *ConfirmRequest) Validate() error {
	if cr.PaymentID == 0 {
		return errors.New("payment_id is required")
	}
	if cr.ConfirmID == "" {
		return errors.New("confirm_id is required")
	}
	if cr.ConfirmRes == "" {
		return errors.New("confirm_res is required")
	}
	if cr.ConfirmCode == "" {
		return errors.New("confirm_code is required")
	}
	if cr.ConfirmOTP == "" {
		return errors.New("confirm_otp is required")
	}
	return nil
}

// RecurrentRequest represents the request payload for a recurrent payment.
// Required fields: RecurrentToken, Amount, OrderID
// Optional fields: Description
// TestMode values: 1 (success), 2 (ov_server_error), 3 (ov_card_not_found), 4 (provider_common_error)
type RecurrentRequest struct {
	RecurrentToken string `json:"token"`       // Required: Recurrent token
	Amount         int64  `json:"amount"`      // Required: Payment amount
	OrderID        string `json:"order_id"`    // Required: Unique order identifier
	Description    string `json:"description"` // Optional: Payment description
	TestMode       *int32 `json:"test_mode"`   // Optional: Flag to enable test mode
}

// Validate checks if the required fields of RecurrentRequest are set and have valid values.
func (rr *RecurrentRequest) Validate() error {
	if rr.RecurrentToken == "" {
		return errors.New("recurrent_token is required")
	}
	if rr.Amount <= 0 {
		return errors.New("amount must be greater than zero")
	}
	if rr.OrderID == "" {
		return errors.New("order_id is required")
	}
	return nil
}

// StatusRequest represents the request payload for checking the status of a payment.
// Required fields: PaymentID, OrderID
type StatusRequest struct {
	PaymentID int64  `json:"payment_id"` // Required: Payment identifier
	OrderID   string `json:"order_id"`   // Required: Unique order identifier
}

// Validate checks if the required fields of StatusRequest are set and have valid values.
func (sr *StatusRequest) Validate() error {
	if sr.PaymentID <= 0 {
		return errors.New("payment_id must be greater than zero")
	}
	if sr.OrderID == "" {
		return errors.New("order_id is required")
	}
	return nil
}

// BalanceRequest represents the request payload for retrieving balance information.
type BalanceRequest struct {
	GetPayoutBalance bool `json:"get_payout_balance"` // Optional: Flag to get payout balance
}

// ReceiptRequest represents the request payload for retrieving a receipt.
// Required fields: PaymentID, OrderID
type ReceiptRequest struct {
	PaymentID int64  `json:"payment_id"` // Required: Payment identifier
	OrderID   string `json:"order_id"`   // Required: Unique order identifier
}

// Validate checks if the required fields of ReceiptRequest are set.
func (rr *ReceiptRequest) Validate() error {
	if rr.PaymentID == 0 {
		return errors.New("payment_id is required")
	}
	if rr.OrderID == "" {
		return errors.New("order_id is required")
	}
	return nil
}

// StatusResponse represents the response payload for checking the status of a payment.
type StatusResponse struct {
	PaymentID       int64         `json:"payment_id"`
	OrderID         string        `json:"order_id"`
	PaymentType     PaymentType   `json:"payment_type"`
	PaymentMethod   PaymentMethod `json:"payment_method"`
	PaymentStatus   PaymentStatus `json:"payment_status"`
	ErrorCode       PaymentErrors `json:"error_code"`
	RecurrentToken  string        `json:"recurrent_token"`
	Amount          int64         `json:"amount"`
	AmountInitial   int64         `json:"amount_initial"`
	AuthAmount      int64         `json:"auth_amount"`
	CapturedAmount  int64         `json:"captured_amount"`
	CapturedDetails interface{}   `json:"captured_details"`
	RefundDetails   interface{}   `json:"refund_details"`
	CanceledDetails interface{}   `json:"canceled_details"`
	RefundedAmount  int64         `json:"refunded_amount"`
	CanceledAmount  int64         `json:"canceled_amount"`
	CreatedDate     string        `json:"created_date"`
	PayerInfo       struct {
		PanMasked string `json:"pan_masked"`
		Holder    string `json:"holder"`
		Email     string `json:"email"`
		Phone     string `json:"phone"`
		UserToken string `json:"user_token"`
	} `json:"payer_info"`
	ConfirmURL     string `json:"confirm_url"`
	PaymentPageURL string `json:"payment_page_url"`
}

// Balance represents balance information.
type Balance struct {
	Currency      string `json:"currency"`
	BalanceAmount int64  `json:"balance_amount"`
	HoldAmount    int64  `json:"hold_amount"`
}

// BalanceResponse represents the response payload for retrieving balance information.
type BalanceResponse struct {
	Balance Balance `json:"balance"`
}

// ReceiptResponse represents the response payload for retrieving a receipt.
type ReceiptResponse struct {
	PaymentID   int64  `json:"payment_id"`
	ReceiptData string `json:"receipt_data"`
}
