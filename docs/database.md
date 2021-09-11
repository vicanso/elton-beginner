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
- 创建时间：该记录的创建时间，在保存时自动生成
- 更新时间：该记录的更新时间，在数据变化时生成

ent的schema支持Mixin形式，因此将创建时间与更新时间定义为公共的schema，定义如下：

```go
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

用户schema的定义则如下：

```go
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
```

## 代码编译

定义好schema之后则可以根据schema编译生成对应的程序代码，首先安装`entc`，执行如下命令`go get entgo.io/ent/cmd/entc@v0.8.0`，需要注意安装版本最好与项目依赖的版本号一致。

安装成功后执行`entc generate ./schema --target ./ent`指定编译代码存放目录。


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

注意：entc编译生成的代码并未添加至代码库中，因此每次执行`make generate`或`entc generate ./schema --target ./ent`生成。