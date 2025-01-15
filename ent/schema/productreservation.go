package schema

import (
	"time"

	"gitlab.calendaria.team/services/finance/billing/ent/enum"
	"gitlab.calendaria.team/services/finance/billing/ent/mixins"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// ProductReservation holds the schema definition for the ProductReservation entity.
type ProductReservation struct {
	ent.Schema
}

// Fields of the ProductReservation.
func (ProductReservation) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id"),
		field.Int64("product_id").Immutable().
			Comment("References the product being reserved."),
		field.Int64("invoice_id").Immutable().
			Comment("References the invoice associated with the reservation."),
		field.Int64("user_id").Immutable().
			Comment("References the user who created the reservation."),
		field.Int64("reserved_quantity").Positive().Default(1).
			Comment("The quantity of the product reserved."),
		field.Enum("status").
			GoType(enum.ReservationStatus("")).
			Default(enum.Pending.Value()).
			Comment("The status of the reservation."),
		field.Time("expiration_time").
			Default(
				func() time.Time {
					return time.Now().Add(15 * time.Minute)
				},
			).
			Comment("The time until the reservation expires."),
	}
}

// Edges of the ProductReservation.
func (ProductReservation) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("product", Product.Type).
			Ref("reservations").
			Unique().
			Required(),
		edge.From("invoice", Invoice.Type).
			Ref("reservations").
			Unique().
			Required(),
	}
}

func (ProductReservation) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.CreateUpdateMixin{},
	}
}
