package schema

import (
	"github.com/makesalekz/billing/ent/enum"
	"github.com/makesalekz/billing/ent/mixins"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/shopspring/decimal"
)

// Product holds the schema definition for the Product entity.
type Product struct {
	ent.Schema
}

// Fields of the Product.
func (Product) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id"),
		field.String("app_id"),
		field.String("name"),
		field.String("description"),
		field.Float("price").
			GoType(decimal.Decimal{}).
			SchemaType(
				map[string]string{
					dialect.Postgres: "numeric",
				},
			),
		field.String("currency").Default("KZT").MaxLen(3),
		field.Bool("is_active").Default(true).Comment("Indicates that this product is active."),
		field.Bool("is_limited").Default(false).Comment("Indicates that this product is limited."),
		field.Time("limited_till").Optional().Nillable().Comment("End of limited period."),
		field.Int64("left").Default(0).Comment("Number of items left in stock."),
		field.Bool("is_unique").Optional().Default(false).Comment("Indicates that this product can only be limited amount of times."),
		field.Int64("unique_limit").Default(0).Max(100).Comment("Number of times this product can be purchased."),
		field.Bool("is_expiring").Optional().Default(false).Comment("Indicates that this product requires renewal."),
		field.Time("expiring_time").Optional().Nillable().Comment("Time when this product expires."),
		field.Enum("payment_model").GoType(enum.PaymentModel("")).Default(enum.Recurrent.Value()).
			Comment("Payment model for this product."),
		field.Enum("product_period").GoType(enum.ProductPeriod("")).Default(enum.ProductPeriodMonth.Value()).
			Comment("Period of product."),
	}
}

// Edges of the Product.
func (Product) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("invoices", Invoice.Type),
		edge.To("subscriptions", Subscriptions.Type),
		edge.To("bundles", Bundle.Type),
		edge.To("reservations", ProductReservation.Type),
	}
}

func (Product) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.CreateUpdateMixin{},
		mixins.SoftDeleteMixin{},
	}
}
