package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// Subscriptions holds the schema definition for the SubscriptionStatus entity.
type Subscriptions struct {
	ent.Schema
}

// Fields of the SubscriptionStatus.
func (Subscriptions) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id"),
		field.Int64("user_id").Immutable(),
		field.Int64("tenant_id").Immutable(),
		field.String("app_id").Immutable(),
		field.Int64("product_id").Immutable(),
		field.Time("renewal_rate").Optional(),
	}
}

// Edges of the SubscriptionStatus.
func (Subscriptions) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("invoices", Invoice.Type),
		edge.From("product", Product.Type).
			Ref("subscriptions").
			Unique().
			Required().
			Immutable().
			Field("product_id"),
	}
}
