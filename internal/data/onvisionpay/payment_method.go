package onevisionpay

// PaymentMethod представляет метод оплаты
type PaymentMethod string

const (
	Ecom       PaymentMethod = "ecom"
	MC         PaymentMethod = "mc"
	Wallet     PaymentMethod = "wallet"
	EcomToMC   PaymentMethod = "ecom_to_mc"
	EcomToEcom PaymentMethod = "ecom_to_ecom"
	MCToEcom   PaymentMethod = "mc_to_ecom"
	MCToMC     PaymentMethod = "mc_to_mc"
)

func (pm *PaymentMethod) IsValid() bool {
	switch *pm {
	case Ecom, MC, Wallet, EcomToMC, EcomToEcom, MCToEcom, MCToMC:
		return true
	}
	return false
}

func (pm *PaymentMethod) DefaultIfInvalid() PaymentMethod {
	if pm.IsValid() {
		return *pm
	}
	return Ecom
}

func (pm *PaymentMethod) Value() string {
	return string(*pm)
}

func (pm *PaymentMethod) Values() []string {
	return []string{
		string(Ecom),
		string(MC),
		string(Wallet),
		string(EcomToMC),
		string(EcomToEcom),
		string(MCToEcom),
		string(MCToMC),
	}
}

func (pm *PaymentMethod) String() string {
	return string(*pm)
}
