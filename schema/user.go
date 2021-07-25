package schema

import (
	"regexp"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// 用户角色
const (
	// UserRoleNormal normal user
	UserRoleNormal = "normal"
	// UserRoleSu super user
	UserRoleSu = "su"
	// UserRoleAdmin admin user
	UserRoleAdmin = "admin"
)

// User holds the schema definition for the User entity.
type User struct {
	ent.Schema
}

// Mixin 用户表的mixin
func (User) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}

// Fields 用户表的字段配置
func (User) Fields() []ent.Field {
	return []ent.Field{
		field.String("account").
			Match(regexp.MustCompile("[a-zA-Z_0-9]+$")).
			NotEmpty().
			Immutable().
			Unique().
			Comment("用户账户"),
		field.String("password").
			Sensitive().
			NotEmpty().
			Comment("用户密码，保存hash之后的值"),
		field.String("name").
			Optional().
			Comment("用户名称"),
		field.Strings("roles").
			Optional().
			Comment("用户角色，由管理员分配"),
		field.Strings("groups").
			Optional().
			Comment("用户分组，按用户职能分配至不同的分组"),
		field.String("email").
			Optional().
			Comment("用户邮箱"),
	}
}

// Edges of the User.
func (User) Edges() []ent.Edge {
	return nil
}

// Indexes 用户表索引
func (User) Indexes() []ent.Index {
	return []ent.Index{
		// 用户账户唯一索引
		index.Fields("account").Unique(),
	}
}
