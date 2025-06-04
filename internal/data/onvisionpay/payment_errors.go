package onevisionpay

type PaymentErrors string

const (
	ProviderServerError             PaymentErrors = "provider_server_error"
	ProviderCommonError             PaymentErrors = "provider_common_error"
	ProviderTimeOut                 PaymentErrors = "provider_time_out"
	ProviderIncorrectResponseFormat PaymentErrors = "provider_incorrect_response_format"
	OvServerError                   PaymentErrors = "ov_server_error"
	OvRoutesUnavailable             PaymentErrors = "ov_routes_unavailable"
	OvSendOtpError                  PaymentErrors = "ov_send_otp_error"
	OvIncorrectOtp                  PaymentErrors = "ov_incorrect_otp"
	OvNotNeedApproveStatus          PaymentErrors = "ov_not_need_approve_status"
	OvPaymentMethodIncorrect        PaymentErrors = "ov_payment_method_incorrect"
	OvPaymentTypeIncorrect          PaymentErrors = "ov_payment_type_incorrect"
	OvCreateOperationError          PaymentErrors = "ov_create_operation_error"
	OvConfirmPaymentError           PaymentErrors = "ov_confirm_payment_error"
	OvCancelPaymentError            PaymentErrors = "ov_cancel_payment_error"
	OvEntityNotFound                PaymentErrors = "ov_entity_not_found"
	OvEntityDuplicate               PaymentErrors = "ov_entity_duplicate"
	OvPaymentNotFound               PaymentErrors = "ov_payment_not_found"
	OvPaymentExpired                PaymentErrors = "ov_payment_expired"
	OvPaymentAlreadyProcessed       PaymentErrors = "ov_payment_already_processed"
	OvOperationNotFound             PaymentErrors = "ov_operation_not_found"
	OvCardNotFound                  PaymentErrors = "ov_card_not_found"
	OvCardIncorrectData             PaymentErrors = "ov_card_incorrect_data"
	OvLimitPaymentAmountMin         PaymentErrors = "ov_limit_payment_amount_min"
	OvLimitPaymentAmount            PaymentErrors = "ov_limit_payment_amount"
	OvLimitPaymentsCount            PaymentErrors = "ov_limit_payments_count"
	OvLimitPaymentsDailyAmount      PaymentErrors = "ov_limit_payments_daily_amount"
	OvLimitPaymentsMonthlyAmount    PaymentErrors = "ov_limit_payments_monthly_amount"
	OvLimitPaymentsDailyCount       PaymentErrors = "ov_limit_payments_daily_count"
	OvLimitPaymentsMonthlyCount     PaymentErrors = "ov_limit_payments_monthly_count"
	OvCommissionIncorrect           PaymentErrors = "ov_commission_incorrect"
	OvRefundAmountExceeded          PaymentErrors = "ov_refund_amount_exceeded"
	OvRefundNotPermitted            PaymentErrors = "ov_refund_not_permitted"
	OvMerchantBalanceInsufficient   PaymentErrors = "ov_merchant_balance_insufficient"
	OvMerchantBalanceNotFound       PaymentErrors = "ov_merchant_balance_not_found"
	OvMerchantNotFound              PaymentErrors = "ov_merchant_not_found"
	OvNotTwoStagePayment            PaymentErrors = "ov_not_two_stage_payment"
	OvClearAmountExceeded           PaymentErrors = "ov_clear_amount_exceeded"
	OvAntiFraudRejected             PaymentErrors = "ov_anti_fraud_rejected"
	OvAntiFraudInternalError        PaymentErrors = "ov_anti_fraud_internal_error"
	OvAntiFraudTimeOut              PaymentErrors = "ov_anti_fraud_time_out"
	OvPaymentLock                   PaymentErrors = "ov_payment_lock"
	OvBalanceLocked                 PaymentErrors = "ov_balance_locked"
	ProviderCardIncorrect           PaymentErrors = "provider_card_incorrect"
	ProviderLogicError              PaymentErrors = "provider_logic_error"
	ProviderCardExpired             PaymentErrors = "provider_card_expired"
	ProviderAntiFraudError          PaymentErrors = "provider_anti_fraud_error"
	ProviderNotPermittedOperation   PaymentErrors = "provider_not_permitted_operation"
	ProviderCardNotFound            PaymentErrors = "provider_card_not_found"
	ProviderInsufficientBalance     PaymentErrors = "provider_insufficient_balance"
	ProviderSendOtpError            PaymentErrors = "provider_send_otp_error"
	ProviderIncorrectOtpError       PaymentErrors = "provider_incorrect_otp_error"
	ProviderAbonentNotFound         PaymentErrors = "provider_abonent_not_found"
	ProviderLimitError              PaymentErrors = "provider_limit_error"
	ProviderAbonentInactive         PaymentErrors = "provider_abonent_inactive"
	ProviderConfirmPaymentError     PaymentErrors = "provider_confirm_payment_error"
	ProviderCancelPaymentError      PaymentErrors = "provider_cancel_payment_error"
	ProviderRefundPaymentError      PaymentErrors = "provider_refund_payment_error"
	ProviderMpiDefaultError         PaymentErrors = "provider_mpi_default_error"
	ProviderMpi3dsError             PaymentErrors = "provider_mpi_3ds_error"
	ProviderCommonIncorrect         PaymentErrors = "provider_common_incorrect"
	ProviderRecurrentError          PaymentErrors = "provider_recurrent_error"
	MpiDefaultError                 PaymentErrors = "mpi_default_error"
	Mpi3dsError                     PaymentErrors = "mpi_3ds_error"
	OvEmailRequired                 PaymentErrors = "ov_email_required"
	None                            PaymentErrors = ""
)

func (pe *PaymentErrors) IsValid() bool {
	switch *pe {
	case ProviderServerError, ProviderCommonError, ProviderTimeOut, ProviderIncorrectResponseFormat, OvServerError, OvRoutesUnavailable, OvSendOtpError, OvIncorrectOtp, OvNotNeedApproveStatus, OvPaymentMethodIncorrect, OvPaymentTypeIncorrect, OvCreateOperationError, OvConfirmPaymentError, OvCancelPaymentError, OvEntityNotFound, OvEntityDuplicate, OvPaymentNotFound, OvPaymentExpired, OvPaymentAlreadyProcessed, OvOperationNotFound, OvCardNotFound, OvCardIncorrectData, OvLimitPaymentAmountMin, OvLimitPaymentAmount, OvLimitPaymentsCount, OvLimitPaymentsDailyAmount, OvLimitPaymentsMonthlyAmount, OvLimitPaymentsDailyCount, OvLimitPaymentsMonthlyCount, OvCommissionIncorrect, OvRefundAmountExceeded, OvRefundNotPermitted, OvMerchantBalanceInsufficient, OvMerchantBalanceNotFound, OvMerchantNotFound, OvNotTwoStagePayment, OvClearAmountExceeded, OvAntiFraudRejected, OvAntiFraudInternalError, OvAntiFraudTimeOut, OvPaymentLock, OvBalanceLocked, ProviderCardIncorrect, ProviderLogicError, ProviderCardExpired, ProviderAntiFraudError, ProviderNotPermittedOperation, ProviderCardNotFound, ProviderInsufficientBalance, ProviderSendOtpError, ProviderIncorrectOtpError, ProviderAbonentNotFound, ProviderLimitError, ProviderAbonentInactive, ProviderConfirmPaymentError, ProviderCancelPaymentError, ProviderRefundPaymentError, ProviderMpiDefaultError, ProviderMpi3dsError, ProviderCommonIncorrect, ProviderRecurrentError, MpiDefaultError, Mpi3dsError, OvEmailRequired:
		return true
	}
	return false
}

func (pe *PaymentErrors) DefaultIfInvalid() PaymentErrors {
	if pe.IsValid() {
		return *pe
	}
	return None
}

func (pe *PaymentErrors) Value() string {
	return string(*pe)
}

func (pe *PaymentErrors) Values() []string {
	return []string{
		string(ProviderServerError),
		string(ProviderCommonError),
		string(ProviderTimeOut),
		string(ProviderIncorrectResponseFormat),
		string(OvServerError),
		string(OvRoutesUnavailable),
		string(OvSendOtpError),
		string(OvIncorrectOtp),
		string(OvNotNeedApproveStatus),
		string(OvPaymentMethodIncorrect),
		string(OvPaymentTypeIncorrect),
		string(OvCreateOperationError),
		string(OvConfirmPaymentError),
		string(OvCancelPaymentError),
		string(OvEntityNotFound),
		string(OvEntityDuplicate),
		string(OvPaymentNotFound),
		string(OvPaymentExpired),
		string(OvPaymentAlreadyProcessed),
		string(OvOperationNotFound),
		string(OvCardNotFound),
		string(OvCardIncorrectData),
		string(OvLimitPaymentAmountMin),
		string(OvLimitPaymentAmount),
		string(OvLimitPaymentsCount),
		string(OvLimitPaymentsDailyAmount),
		string(OvLimitPaymentsMonthlyAmount),
		string(OvLimitPaymentsDailyCount),
		string(OvLimitPaymentsMonthlyCount),
		string(OvCommissionIncorrect),
		string(OvRefundAmountExceeded),
		string(OvRefundNotPermitted),
		string(OvMerchantBalanceInsufficient),
		string(OvMerchantBalanceNotFound),
		string(OvMerchantNotFound),
		string(OvNotTwoStagePayment),
		string(OvClearAmountExceeded),
		string(OvAntiFraudRejected),
		string(OvAntiFraudInternalError),
		string(OvAntiFraudTimeOut),
		string(OvPaymentLock),
		string(OvBalanceLocked),
		string(ProviderCardIncorrect),
		string(ProviderLogicError),
		string(ProviderCardExpired),
		string(ProviderAntiFraudError),
		string(ProviderNotPermittedOperation),
		string(ProviderCardNotFound),
		string(ProviderInsufficientBalance),
		string(ProviderSendOtpError),
		string(ProviderIncorrectOtpError),
		string(ProviderAbonentNotFound),
		string(ProviderLimitError),
		string(ProviderAbonentInactive),
		string(ProviderConfirmPaymentError),
		string(ProviderCancelPaymentError),
		string(ProviderRefundPaymentError),
		string(ProviderMpiDefaultError),
		string(ProviderMpi3dsError),
		string(ProviderCommonIncorrect),
		string(ProviderRecurrentError),
		string(MpiDefaultError),
		string(Mpi3dsError),
		string(OvEmailRequired),
		string(None),
	}
}

func (pe *PaymentErrors) String() string {
	return string(*pe)
}
