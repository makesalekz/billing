package enum

type ReservationStatus string

const (
	Pending   ReservationStatus = "PENDING"
	Completed ReservationStatus = "COMPLETED"
	Expired   ReservationStatus = "EXPIRED"
	Cancelled ReservationStatus = "CANCELLED"
)

func productReservationValues() []ReservationStatus {
	return []ReservationStatus{
		Pending,
		Completed,
		Expired,
		Cancelled,
	}
}

func (ReservationStatus) Values() (kinds []string) {
	for _, value := range productReservationValues() {
		kinds = append(kinds, string(value))
	}
	return
}

func (e ReservationStatus) Value() string {
	return string(e)
}

func (e ReservationStatus) IsValid() bool {
	switch e {
	case Pending, Completed, Expired, Cancelled:
		return true
	}
	return false
}

func (e ReservationStatus) String() string {
	return string(e)
}
