package enum

type ProductPeriod string

const (
	ProductPeriodDay       ProductPeriod = "day"
	ProductPeriodWeek      ProductPeriod = "week"
	ProductPeriodMonth     ProductPeriod = "month"
	ProductPeriodYear      ProductPeriod = "year"
	ProductPeriodUnlimited ProductPeriod = "unlimited"
)

func productPeriodValues() []ProductPeriod {
	return []ProductPeriod{
		ProductPeriodDay,
		ProductPeriodWeek,
		ProductPeriodMonth,
		ProductPeriodYear,
		ProductPeriodUnlimited,
	}
}

func (ProductPeriod) Values() (kinds []string) {
	for _, value := range productPeriodValues() {
		kinds = append(kinds, string(value))
	}
	return
}

func (e ProductPeriod) Value() string {
	return string(e)
}

func (e ProductPeriod) IsValid() bool {
	switch e {
	case ProductPeriodDay, ProductPeriodWeek, ProductPeriodMonth, ProductPeriodYear, ProductPeriodUnlimited:
		return true
	}
	return false
}
