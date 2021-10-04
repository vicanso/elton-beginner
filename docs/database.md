---
description: 数据库要基于系统方向选择更合适的数据库，而每家公司也有各自的常用数据库，根据各自应用需要选择则可
---

# ent

ent是一个简单而又强大的orm框架，根据定义的schema自动编译出相关的代码，支持mysql、postgresql等数据库

## 用户定义

- 账户：用户账号，只允许字母与数字以及下划线，唯一索引，创建后不可修改
- 密码：用户密码，非明文存储
- 名称：用户名称，可选
- 角色：用户角色，可选，允许一个用户多个角色
- 分组：用户分组，可选，允许一个用户多个分组
- 邮箱：用户邮箱，可选
- 创建时间：该记录的创建时间，在数据创建时生成
- 更新时间：该记录的更新时间，在数据更新时生成

ent的schema支持Mixin形式，因此将创建时间、更新时间以及状态定义为公共的scheme，如下：

```go

type Status int8

const (
	// 状态启用
	StatusEnabled Status = iota + 1
	// 状态禁用
	StatusDisabled
)

// ToInt8 转换为int8
func (status Status) Int8() int8 {
	return int8(status)
}

// String 转换为string
func (status Status) String() string {
	switch status {
	case StatusEnabled:
		return "启用"
	case StatusDisabled:
		return "禁用"
	default:
		return "未知"
	}
}

// StatusMixin 状态的schema
type StatusMixin struct {
	mixin.Schema
}

// Fields 公共的status的字段
func (StatusMixin) Fields() []ent.Field {
	return []ent.Field{
		field.Int8("status").
			Range(StatusEnabled.Int8(), StatusDisabled.Int8()).
			Default(StatusEnabled.Int8()).
			GoType(Status(StatusEnabled)).
			Comment("状态，默认为启用状态"),
	}
}

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
```

用户schema的定义添加`TimeMixin`与`StatusMixin`，再添加相关的fields的定义，以及按需要添加indexes则可，代码如下：

```go
// User holds the schema definition for the User entity.
type User struct {
	ent.Schema
}

// Mixin 用户表的mixin
func (User) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
		StatusMixin{},
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
```

## 模板定义

ent提供自定义模板的形式，可以在编译出相应代码时添加各类自定义的函数。

用户的Schema中添加了Status，希望添加一个StatusDesc用来转义其对应的中文，虽然可以在定义Schema时的fields添加，但是这样会导致数据库也增加了此额外的字段，因此使用模板的形式调整编译后生成的代码。模板如下：


```tmpl
{{ define "model/fields/additional" }}
	{{/* 添加额外字段 */}}
	{{- range $i, $f := $.Fields }}
	{{- if eq $f.Name "status" }}
		// 状态描述
		StatusDesc string `json:"statusDesc,omitempty"`
	{{- end }}
	{{- end }}
{{ end }}
```

模板的处理比较简单，仅是针对fields如果有定义status则添加`StatusDesc`，具体实现的时候可根据应用场景添加更多的限制，如仅针对某schema等等，具体使用可至ent官方站点查看相关文档。

通过增加模板后，编译生成的`User`定义已添加`StatusDesc`属性，如下：

```go
type User struct {
	...
	...

	// 状态描述
	StatusDesc string `json:"statusDesc,omitempty"`
}
```

此时虽然已经添加了相应的字段，但是需要根据`Status`来生成其对应的中文描述，ent暂时未提供Query Hook（Roadmap for v1上的提及），无法在query中添加处理。考虑到`StatusDesc`用于界面上展示时使用，而接口使用json形式返回，因此调整MarshalJSON的实现，在序列化时生成此字段。

golang的`json.Marshal`序列化时，会先判断该对象是否实例了`MarshalJSON`方法，如果实现了则直接调用，因此我们只要添加自定义的`MarshalJSON`则可，代码如下：

```go
type MarshalUser User
// 转换为json时先将相应字段填充
func (u *User) MarshalJSON() ([]byte, error) {
	tmp := (*MarshalUser)(u)
	tmp.StatusDesc = tmp.Status.String()
	return json.Marshal(tmp)
}
```

由于ent数据库相关的代码是通过编译生成，因此还是需要通过模板的形式来生成代码，模板如下：

```tmpl
{{/* gotype: entgo.io/ent/entc/gen.Graph */}}

{{ define "marshal" }}

{{ $pkg := base $.Config.Package }}
{{ template "header" $ }}

import "encoding/json"

{{ range $n := $.Nodes }}

{{/* 用户 */}}
{{- if eq $n.Name "User" }}
type MarshalUser User
// 转换为json时先将相应字段填充
func (u *User) MarshalJSON() ([]byte, error) {
	tmp := (*MarshalUser)(u)
	tmp.StatusDesc = tmp.Status.String()
	return json.Marshal(tmp)
}
{{ end }}

{{ end }}

{{ end }}
```

## 代码编译

定义好schema之后则可以根据schema编译生成对应的程序代码，首先安装`entc`，执行如下命令`go get entgo.io/ent/cmd/entc@v0.9.1`，需要注意安装版本与项目依赖的版本号一致。

安装成功后执行`go run entgo.io/ent/cmd/ent generate ./schema --template ./template --target ./ent`指定编译代码存放目录。


## hooks

ent提供一列表的勾子函数可用于对于更新类操作增加勾子处理，下面代码禁止删除操作、对于数据更新类操作记录更新字段以及操作时间，并记录当前处理请求数。

```go
// initSchemaHooks 初始化相关的hooks
func initSchemaHooks(c *ent.Client) {
	ignoredNameList := []string{
		"updated_at",
		"created_at",
	}
	isIgnored := func(name string) bool {
		for _, item := range ignoredNameList {
			if item == name {
				return true
			}
		}
		return false
	}
	// 禁止删除数据
	c.Use(hook.Reject(ent.OpDelete | ent.OpDeleteOne))
	// 数据库操作统计
	c.Use(func(next ent.Mutator) ent.Mutator {
		processing := atomic.NewInt32(0)
		return ent.MutateFunc(func(ctx context.Context, m ent.Mutation) (ent.Value, error) {
			count := processing.Inc()
			defer processing.Dec()
			schemaType := m.Type()
			op := m.Op().String()

			startedAt := time.Now()
			result := cs.ResultSuccess
			message := ""

			mutateResult, err := next.Mutate(ctx, m)
			// 如果失败，则记录出错信息
			if err != nil {
				result = cs.ResultFail
				message = err.Error()
			}
			// 记录更新字段
			data := make(map[string]interface{})
			for _, name := range m.Fields() {
				if isIgnored(name) {
					continue
				}
				value, ok := m.Field(name)
				if !ok {
					continue
				}
				valueType := reflect.TypeOf(value)
				maxString := 50
				switch valueType.Kind() {
				case reflect.String:
					str, ok := value.(string)
					// 如果更新过长，则截断
					if ok {
						value = util.CutRune(str, maxString)
					}
				}
				// 对于密码等字段使用***
				if cs.MaskRegExp.MatchString(name) {
					data[name] = "***"
				} else {
					data[name] = value
				}
			}

			d := time.Since(startedAt)
			log.Info(ctx).
				Str("category", "entStats").
				Str("schema", schemaType).
				Str("op", op).
				Int("result", result).
				Int32("processing", count).
				Str("use", d.String()).
				Dict("data", zerolog.Dict().Fields(data)).
				Str("message", message).
				Msg("")
			return mutateResult, err
		})
	})
}
```

## 初始化客户端

根据配置的数据库连接初始化数据库客户端。

```go
var defaultEntDriver, defaultEntClient = mustNewEntClient()

// mustNewEntClient 初始化客户端与driver
func mustNewEntClient() (*entsql.Driver, *ent.Client) {
	postgresConfig := config.MustGetPostgresConfig()

	maskURI := postgresConfig.URI
	urlInfo, _ := url.Parse(maskURI)
	if urlInfo != nil {
		pass, ok := urlInfo.User.Password()
		if ok {
			// 连接串输出时将密码***处理
			maskURI = strings.ReplaceAll(maskURI, pass, "***")
		}
	}
	log.Info(context.Background()).
		Str("uri", maskURI).
		Msg("connect postgres")
	db, err := sql.Open("pgx", postgresConfig.URI)
	if err != nil {
		panic(err)
	}
	if postgresConfig.MaxIdleConns != 0 {
		db.SetMaxIdleConns(postgresConfig.MaxIdleConns)
	}
	if postgresConfig.MaxOpenConns != 0 {
		db.SetMaxOpenConns(postgresConfig.MaxOpenConns)
	}
	if postgresConfig.MaxIdleTime != 0 {
		db.SetConnMaxIdleTime(postgresConfig.MaxIdleTime)
	}

	// Create an ent.Driver from `db`.
	driver := entsql.OpenDB(dialect.Postgres, db)
	entLogger := log.NewEntLogger()
	c := ent.NewClient(ent.Driver(driver), ent.Log(entLogger.Log))

	initSchemaHooks(c)
	return driver, c
}
```

## 其它功能函数

- `EntPing`: 执行ping命令，用于启动程序时检测连接数据库是否成功以及定时检测告警
- `EntInitSchema`: 根据schema定义生成表结构执行migrate操作，若项目中存在大量表定义，建议不直接执行而是将相关输入至命令行，手工执行
- `EntGetStats` 获取数据库连接的相关统计指标

注意：entc编译生成的代码并未添加至代码库中，因此每次执行`make generate`或`go run entgo.io/ent/cmd/ent generate ./schema --template ./template --target ./ent`生成。