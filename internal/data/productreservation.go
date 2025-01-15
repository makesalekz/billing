package data

import (
	"context"
	"time"

	"github.com/pkg/errors"

	"gitlab.calendaria.team/services/finance/billing/ent"
	"gitlab.calendaria.team/services/finance/billing/ent/enum"
	"gitlab.calendaria.team/services/finance/billing/ent/productreservation"
)

type ProductReservationRepo struct {
	db *ent.Client
}

// NewProductReservationRepo creates a new repository instance.
func NewProductReservationRepo(d *Data) *ProductReservationRepo {
	return &ProductReservationRepo{
		db: d.db,
	}
}

// CreateReservation creates a new product reservation within a transaction and updates the product stock.
func (r *ProductReservationRepo) CreateReservation(
	ctx context.Context, reservationDto ProductReservationDto,
) (*ent.ProductReservation, error) {
	var reservation *ent.ProductReservation

	tx, err := r.db.Tx(ctx)
	if err != nil {
		return nil, err
	}

	defer func() {
		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				err = rollbackErr
			}
		}
	}()

	product, err := tx.Product.Get(ctx, reservationDto.ProductID)
	if err != nil {
		return nil, err
	}

	if product.Left < reservationDto.ReservationQuantity {
		return nil, errors.New("insufficient stock")
	}

	if _, err := tx.Product.
		UpdateOneID(reservationDto.ProductID).
		AddLeft(-reservationDto.ReservationQuantity).
		Save(ctx); err != nil {
		return nil, err
	}

	// Create reservation
	reservation, err = tx.ProductReservation.
		Create().
		SetProductID(reservationDto.ProductID).
		SetInvoiceID(reservationDto.InvoiceID).
		SetUserID(reservationDto.UserID).
		SetReservedQuantity(reservationDto.ReservationQuantity).
		Save(ctx)

	if err != nil {
		return nil, err
	}

	if commitErr := tx.Commit(); commitErr != nil {
		return nil, errors.Wrap(commitErr, "failed to commit transaction")
	}

	return reservation, nil
}

// GetReservation retrieves a reservation by its ID.
func (r *ProductReservationRepo) GetReservation(ctx context.Context, id int64) (*ent.ProductReservation, error) {
	return r.db.ProductReservation.Get(ctx, id)
}

// UpdateReservationStatusByInvoiceID updates the status of a reservation.
func (r *ProductReservationRepo) UpdateReservationStatusByInvoiceID(
	ctx context.Context, invoiceID int64, status enum.ReservationStatus,
) error {
	_, err := r.db.ProductReservation.
		Update().
		Where(productreservation.InvoiceIDEQ(invoiceID)).
		SetStatus(status).
		Save(ctx)
	if err != nil && ent.IsNotFound(err) {
		return nil
	}
	return err
}

// CancelReservationByInvoiceID cancels a reservation by invoice ID and restores the product stock.
func (r *ProductReservationRepo) CancelReservationByInvoiceID(ctx context.Context, invoiceID int64) error {
	tx, err := r.db.Tx(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				err = rollbackErr
			}
		}
	}()

	reservations, err := tx.ProductReservation.
		Query().
		Where(
			productreservation.InvoiceIDEQ(invoiceID),
			productreservation.StatusEQ(enum.Pending),
		).
		All(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil
		}
		return err
	}
	if len(reservations) == 0 {
		err = tx.Commit()
		if err != nil {
			return err
		}
		return nil
	}

	for _, reservation := range reservations {
		if _, err := tx.Product.
			UpdateOneID(reservation.ProductID).
			AddLeft(reservation.ReservedQuantity).
			Save(ctx); err != nil {
			return err
		}

		if _, err := tx.ProductReservation.
			UpdateOneID(reservation.ID).
			SetStatus(enum.Cancelled).
			Save(ctx); err != nil {
			return err
		}
	}

	if commitErr := tx.Commit(); commitErr != nil {
		return commitErr
	}

	return nil
}

// ProcessExpiredReservations processes all reservations that have expired by restoring product stock and updating reservation statuses.
func (r *ProductReservationRepo) ProcessExpiredReservations(ctx context.Context) error {
	tx, err := r.db.Tx(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				err = rollbackErr
			}
		}
	}()

	reservations, err := tx.ProductReservation.
		Query().
		Where(
			productreservation.StatusEQ(enum.Pending),
			productreservation.ExpirationTimeLT(time.Now()),
		).
		All(ctx)
	if err != nil {
		return err
	}

	for _, reservation := range reservations {
		if _, err := tx.Product.
			UpdateOneID(reservation.ProductID).
			AddLeft(reservation.ReservedQuantity).
			Save(ctx); err != nil {
			return err
		}

		if _, err := tx.ProductReservation.
			UpdateOneID(reservation.ID).
			SetStatus(enum.Expired).
			Save(ctx); err != nil {
			return err
		}
	}

	if commitErr := tx.Commit(); commitErr != nil {
		return commitErr
	}

	return nil
}
