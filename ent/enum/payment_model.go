package enum

type PaymentModel string

const (
	OneTime   PaymentModel = "ONE_TIME"
	Recurrent PaymentModel = "RECURRENT"
)

func paymentModelValues() []PaymentModel {
	return []PaymentModel{
		OneTime,
		Recurrent,
	}
}

func (PaymentModel) Values() (kinds []string) {
	for _, value := range paymentModelValues() {
		kinds = append(kinds, string(value))
	}
	return
}

func (e PaymentModel) Value() string {
	return string(e)
}

func (e PaymentModel) IsValid() bool {
	switch e {
	case OneTime, Recurrent:
		return true
	}
	return false
}
