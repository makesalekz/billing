package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

// Dummy holds the schema definition for the Dummy entity.
type Dummy struct {
	ent.Schema
}

// Fields of the Dummy.
func (Dummy) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("user_id"),
		field.String("name").Default(""),
		field.Time("created_at").Default(time.Now),
		field.Time("updated_at").Default(time.Now),
	}
}

// Edges of the Dummy.
func (Dummy) Edges() []ent.Edge {
	return nil
}
