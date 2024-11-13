package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/shopspring/decimal"
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
		field.Float("overusage_price").
			GoType(decimal.Decimal{}).
			SchemaType(map[string]string{
				dialect.Postgres: "numeric",
			}),
	}
}

// Edges of the Bundle.
func (Bundle) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("product", Product.Type).Ref("bundles").Unique().Required().Immutable().Field("product_id"),
		edge.From("item", Item.Type).Ref("bundles").Unique().Required().Immutable().Field("item_id"),
	}
}

func (Bundle) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("product_id", "item_id").Unique(),
	}
}

func (Bundle) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.SoftDeleteMixin{},
		mixins.CreateUpdateMixin{},
	}
}
