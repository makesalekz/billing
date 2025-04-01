package messages

import "gitlab.calendaria.team/services/finance/billing/ent"

type RefreshItems struct {
	Item     *ent.Item `json:"item"`
	Amount   float64   `json:"amount"`
	UserID   int64     `json:"user_id"`
	TenantID int64     `json:"tenant_id"`
	AppID    string    `json:"app_id"`
	IsTrial  bool      `json:"is_trial"`
}
