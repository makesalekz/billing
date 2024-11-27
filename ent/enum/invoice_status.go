package enum

type InvoiceStatus string

const (
	Created          InvoiceStatus = "CREATED"
	Paid             InvoiceStatus = "PAID"
	CanceledByUser   InvoiceStatus = "CANCELED_BY_USER"
	CanceledByVendor InvoiceStatus = "CANCELED_BY_VENDOR"
	Failed           InvoiceStatus = "FAILED"
	Rejected         InvoiceStatus = "REJECTED"
	Revoked          InvoiceStatus = "REVOKED"
)

func invoiceStatusValues() []InvoiceStatus {
	return []InvoiceStatus{Created, Paid, CanceledByUser, CanceledByVendor, Failed, Rejected, Revoked}
}

func (InvoiceStatus) Values() (kinds []string) {
	for _, value := range invoiceStatusValues() {
		kinds = append(kinds, string(value))
	}
	return
}

func (m InvoiceStatus) Value() string {
	return string(m)
}

func (m InvoiceStatus) IsValid() bool {
	for _, value := range invoiceStatusValues() {
		if m == value {
			return true
		}
	}
	return false
}
