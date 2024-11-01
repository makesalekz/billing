package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// ConsumedStatus holds the schema definition for the ConsumedStatus entity.
type ConsumedStatus struct {
	ent.Schema
}

// Fields of the ConsumedStatus.
func (ConsumedStatus) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id"),
		field.Int64("user_id").Immutable(),
		field.Int64("tenant_id").Immutable(),
		field.Int64("app_id").Immutable(),
		field.Int64("item_id").Immutable(),
		field.Float("consumed").Optional().Default(0),
		field.Float("left").Optional().Default(0),
		field.Time("active_till").Optional(),
		field.Time("start_from").Optional(),
	}
}

// Edges of the ConsumedStatus.
func (ConsumedStatus) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("item", Item.Type).
			Ref("consumed_statuses").
			Unique().
			Required().
			Immutable().
			Field("item_id"),
	}
}

func (ConsumedStatus) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id", "app_id", "item_id"),
	}
}
