package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"gitlab.calendaria.team/services/finance/invoices/ent/mixins"
)

// Bundle holds the schema definition for the Bundle entity.
type Bundle struct {
	ent.Schema
}

// Fields of the Bundle.
func (Bundle) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.Int64("product_id").Immutable(),
		field.Int64("item_id").Immutable(),
		field.Float("amount").Default(1),
	}
}

// Edges of the Bundle.
func (Bundle) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("product", Product.Type).Ref("bundles").Unique().Required().Immutable().Field("product_id"),
		edge.From("item", Item.Type).Ref("bundles").Unique().Required().Immutable().Field("item_id"),
	}
}

func (Bundle) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.SoftDeleteMixin{},
		mixins.CreateUpdateMixin{},
	}
}
