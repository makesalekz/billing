package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/shopspring/decimal"
	"gitlab.calendaria.team/services/finance/billing/ent/enum"
)

// Invoice holds the schema definition for the Invoice entity.
type Invoice struct {
	ent.Schema
}

// Fields of the Invoice.
func (Invoice) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id"),
		field.Int64("user_id").Immutable(),
		field.Int64("tenant_id").Immutable(),
		field.String("app_id").Immutable(),
		field.Int64("product_id").Immutable(),
		field.Int64("amount").Immutable(),
		field.Float("price").
			GoType(decimal.Decimal{}).
			SchemaType(map[string]string{
				dialect.Postgres: "numeric",
			}),
		field.String("currency").Default("KZT").MaxLen(3),
		field.Enum("status").GoType(enum.InvoiceStatus("")).Default(enum.Created.Value()),
		field.Time("paid_at").Optional().Nillable(),
		field.Time("paid_till").Optional().Nillable(),
		field.Bool("is_paid_at_processed").Default(false),
		field.Bool("is_paid_till_processed").Default(false),
		field.Int64("subscription_id").Optional().Nillable().Immutable(),
	}
}

// Edges of the Invoice.
func (Invoice) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("product", Product.Type).
			Ref("invoices").
			Required().
			Immutable().
			Unique().
			Field("product_id"),
		edge.From("subscriptions", Subscriptions.Type).
			Ref("invoices").
			Unique().
			Immutable().
			Field("subscription_id"),
	}
}
