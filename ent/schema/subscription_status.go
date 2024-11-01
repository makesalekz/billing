package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// SubscriptionStatus holds the schema definition for the SubscriptionStatus entity.
type SubscriptionStatus struct {
	ent.Schema
}

// Fields of the SubscriptionStatus.
func (SubscriptionStatus) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id"),
		field.Int64("invoice_id").Immutable(),
		field.Int64("item_id").Immutable(),
		field.Time("active_till").Optional(),
		field.Time("start_from").Optional(),
	}
}

// Edges of the SubscriptionStatus.
func (SubscriptionStatus) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("invoice", Invoice.Type).
			Ref("subscription_statuses").
			Unique().
			Required().
			Immutable().
			Field("invoice_id"),
		edge.From("item", Item.Type).
			Ref("subscription_statuses").
			Unique().
			Required().
			Immutable().
			Field("item_id"),
	}
}
