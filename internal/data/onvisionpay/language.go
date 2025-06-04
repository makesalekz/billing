package onevisionpay

type Language string

const (
	Russian Language = "ru"
	English Language = "en"
	Kazakh  Language = "kz"
)

func (l *Language) IsValid() bool {
	switch *l {
	case Russian, English, Kazakh:
		return true
	}
	return false
}

func (l *Language) DefaultIfInvalid() Language {
	if l.IsValid() {
		return *l
	}
	return Russian
}

func (l *Language) Value() string {
	return string(*l)
}

func (l *Language) Values() []string {
	return []string{
		string(Russian),
		string(English),
		string(Kazakh),
	}
}

func (l *Language) String() string {
	return string(*l)
}
