package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/shopspring/decimal"
	"gitlab.calendaria.team/services/finance/invoices/ent/enum"
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
		field.Time("paid_at").Optional(),
	}
}

// Edges of the Invoice.
func (Invoice) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("product", Product.Type).Ref("invoices").Required().Immutable().Unique().Field("product_id"),
		edge.To("consumed_statuses", ConsumedStatus.Type),
		edge.To("subscription_statuses", SubscriptionStatus.Type),
	}
}
