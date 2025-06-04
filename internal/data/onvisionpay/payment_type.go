package onevisionpay

type PaymentType string

const (
	Pay      PaymentType = "pay"
	Payout   PaymentType = "payout"
	Transfer PaymentType = "transfer"
)

func (pt *PaymentType) IsValid() bool {
	switch *pt {
	case Pay, Payout, Transfer:
		return true
	}
	return false
}

func (pt *PaymentType) DefaultIfInvalid() PaymentType {
	if pt.IsValid() {
		return *pt
	}
	return Pay
}

func (pt *PaymentType) Value() string {
	return string(*pt)
}

func (pt *PaymentType) Values() []string {
	return []string{
		string(Pay),
		string(Payout),
		string(Transfer),
	}
}

func (pt *PaymentType) String() string {
	return string(*pt)
}
