package enum

type PaymentProvider string

const (
	AppStore         PaymentProvider = "APP_STORE"
	OneVisionPayment PaymentProvider = "ONE_VISION_PAYMENT"
	TipTopPayment    PaymentProvider = "TIP_TOP_PAYMENT"
)

func paymentProviderValues() []PaymentProvider {
	return []PaymentProvider{AppStore, OneVisionPayment, TipTopPayment}
}

func (PaymentProvider) Values() (kinds []string) {
	for _, value := range paymentProviderValues() {
		kinds = append(kinds, string(value))
	}
	return
}

func (m PaymentProvider) Value() string {
	return string(m)
}

func (m PaymentProvider) IsValid() bool {
	for _, value := range paymentProviderValues() {
		if m == value {
			return true
		}
	}
	return false
}
