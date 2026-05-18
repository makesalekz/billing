package data

import (
	"context"

	"github.com/makesalekz/billing/ent"
	"github.com/makesalekz/billing/ent/paymentprofile"
)

type PaymentProfileRepo interface {
	CreateProfile(ctx context.Context, dto PaymentProfileDto) (*ent.PaymentProfile, error)
	GetProfileByUserID(ctx context.Context, userID int64) (*ent.PaymentProfile, error)
	UpdateProfile(ctx context.Context, profileID int64, dto PaymentProfileDto) error
}

type paymentProfileRepo struct {
	db *ent.Client
}

func NewPaymentProfileRepo(d *Data) PaymentProfileRepo {
	return &paymentProfileRepo{
		db: d.db,
	}
}

func (r *paymentProfileRepo) CreateProfile(ctx context.Context, dto PaymentProfileDto) (*ent.PaymentProfile, error) {
	query := r.db.PaymentProfile.Create().
		SetUserID(dto.UserID).
		SetPanMasked(dto.PanMasked).
		SetHolder(dto.Holder).
		SetUserToken(dto.UserToken)

	if dto.Email != nil {
		query.SetEmail(*dto.Email)
	}
	if dto.Phone != nil {
		query.SetPhone(*dto.Phone)
	}
	if dto.RecurrentToken != nil {
		query.SetRecurrentToken(*dto.RecurrentToken)
	}

	return query.Save(ctx)
}

func (r *paymentProfileRepo) GetProfileByUserID(ctx context.Context, userID int64) (*ent.PaymentProfile, error) {
	return r.db.PaymentProfile.Query().
		Where(paymentprofile.UserID(userID)).
		Only(ctx)
}

func (r *paymentProfileRepo) UpdateProfile(ctx context.Context, profileID int64, dto PaymentProfileDto) error {
	update := r.db.PaymentProfile.UpdateOneID(profileID)
	if dto.Email != nil {
		update = update.SetEmail(*dto.Email)
	}
	if dto.Phone != nil {
		update = update.SetPhone(*dto.Phone)
	}
	if dto.RecurrentToken != nil {
		update = update.SetRecurrentToken(*dto.RecurrentToken)
	}
	return update.Exec(ctx)
}
