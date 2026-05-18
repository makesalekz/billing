package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"

	"github.com/makesalekz/billing/ent/mixins"
)

// PaymentProfile holds the schema definition for the Item entity.
type PaymentProfile struct {
	ent.Schema
}

// Fields of the PaymentProfile.
func (PaymentProfile) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id"),
		field.Int64("user_id").Immutable(),
		field.String("pan_masked").Immutable(),
		field.String("holder").Immutable(),
		field.String("email").Optional().Default(""),
		field.String("phone").Optional().Default(""),
		field.String("user_token").Optional().Default(""),
		field.String("recurrent_token").Optional().Nillable().Unique(),
	}
}

// Edges of the PaymentProfile.
func (PaymentProfile) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("invoices", Invoice.Type),
	}
}

// Mixin of the PaymentProfile.
func (PaymentProfile) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.CreateUpdateMixin{},
		mixins.SoftDeleteMixin{},
	}
}
