package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"entgo.io/ent/schema/mixin"
)

// TimeMixin 公共的时间schema
type TimeMixin struct {
	mixin.Schema
}

// Fields 公共时间schema的字段，包括创建于与更新于
func (TimeMixin) Fields() []ent.Field {
	return []ent.Field{
		field.Time("created_at").
			// 对于多个单词组成的，如果需要使用select，则需要添加sql tag
			StructTag(`json:"createdAt" sql:"created_at"`).
			Immutable().
			Default(time.Now).
			Comment("创建时间，添加记录时由程序自动生成"),
		field.Time("updated_at").
			StructTag(`json:"updatedAt" sql:"updated_at"`).
			Default(time.Now).
			Immutable().
			UpdateDefault(time.Now).
			Comment("更新时间，更新记录时由程序自动生成"),
	}
}

// Indexes 公共时间字段索引
func (TimeMixin) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("created_at"),
		index.Fields("updated_at"),
	}
}
