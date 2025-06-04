package onevisionpay

type PaymentStatus string

const (
	Created         PaymentStatus = "created"
	Refunded        PaymentStatus = "refunded"
	Canceled        PaymentStatus = "canceled"
	NeedApprove     PaymentStatus = "need_approve"
	Hold            PaymentStatus = "hold"
	Clearing        PaymentStatus = "clearing"
	Withdraw        PaymentStatus = "withdraw"
	Refill          PaymentStatus = "refill"
	Processing      PaymentStatus = "processing"
	Process         PaymentStatus = "process"
	Error           PaymentStatus = "error"
	Chargeback      PaymentStatus = "chargeback"
	PartialRefund   PaymentStatus = "partial_refund"
	PartialClearing PaymentStatus = "partial_clearing"
)

func (ps *PaymentStatus) IsValid() bool {
	switch *ps {
	case Created, Refunded, Canceled, NeedApprove, Hold, Clearing, Withdraw, Refill, Processing, Process, Error, Chargeback, PartialRefund, PartialClearing:
		return true
	}
	return false
}

func (ps *PaymentStatus) DefaultIfInvalid() PaymentStatus {
	if ps.IsValid() {
		return *ps
	}
	return ""
}

func (ps *PaymentStatus) Value() string {
	return string(*ps)
}

func (ps *PaymentStatus) Values() []string {
	return []string{
		string(Created),
		string(Refunded),
		string(Canceled),
		string(NeedApprove),
		string(Hold),
		string(Clearing),
		string(Withdraw),
		string(Refill),
		string(Processing),
		string(Process),
		string(Error),
		string(Chargeback),
		string(PartialRefund),
		string(PartialClearing),
	}
}
