package helper

import (
	"context"
	"database/sql"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/stdlib"
	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/rs/zerolog"
	"github.com/vicanso/beginner/config"
	"github.com/vicanso/beginner/cs"
	"github.com/vicanso/beginner/ent"
	"github.com/vicanso/beginner/ent/hook"
	"github.com/vicanso/beginner/log"
	"github.com/vicanso/beginner/util"
	"github.com/vicanso/hes"
	"go.uber.org/atomic"
)

var (
	defaultEntDriver, defaultEntClient = mustNewEntClient()
)
var databaseConfig = config.MustGetDatabaseConfig()
var (
	initSchemaOnce sync.Once
)

func getMaskURI(uri string) string {
	reg := regexp.MustCompile(`://\S+?:(\S+?)@`)
	result := reg.FindAllStringSubmatch(uri, 1)
	if len(result) != 1 && len(result[0]) != 2 {
		return uri
	}
	return strings.Replace(uri, result[0][1], "***", 1)
}
func pgOnBeforeConnect(ctx context.Context, config *pgx.ConnConfig) error {
	log.Info(ctx).
		Str("category", "pgEvent").
		Str("host", config.Host).
		Msg("pg connecting")
	return nil
}

func pgOnAfterConnect(ctx context.Context, conn *pgx.Conn) error {
	log.Info(ctx).
		Str("category", "pgEvent").
		Str("host", conn.Config().Host).
		Msg("pg connected")
	return nil
}

func newClientDB(uri string) (*sql.DB, string, error) {
	if strings.HasPrefix(uri, "postgres://") {
		config, err := pgx.ParseConfig(uri)
		if err != nil {
			return nil, "", err
		}
		db := stdlib.OpenDB(
			*config,
			stdlib.OptionBeforeConnect(pgOnBeforeConnect),
			stdlib.OptionAfterConnect(pgOnAfterConnect),
		)
		return db, dialect.Postgres, err
	}
	mysqlPrefix := "mysql://"
	if strings.HasPrefix(uri, mysqlPrefix) {
		db, err := sql.Open("mysql", strings.Replace(uri, mysqlPrefix, "", 1))
		return db, dialect.MySQL, err
	}
	return nil, "", hes.New("not support the database")
}

// mustNewEntClient 初始化客户端与driver
func mustNewEntClient() (*entsql.Driver, *ent.Client) {

	maskURI := getMaskURI(databaseConfig.URI)
	log.Info(context.Background()).
		Str("uri", maskURI).
		Msg("connect database")
	// 根据连接串初始化mysql或postgres
	db, driverType, err := newClientDB(databaseConfig.URI)
	if err != nil {
		panic(err)
	}
	if databaseConfig.MaxIdleConns != 0 {
		db.SetMaxIdleConns(databaseConfig.MaxIdleConns)
	}
	if databaseConfig.MaxOpenConns != 0 {
		db.SetMaxOpenConns(databaseConfig.MaxOpenConns)
	}
	if databaseConfig.MaxIdleTime != 0 {
		db.SetConnMaxIdleTime(databaseConfig.MaxIdleTime)
	}

	// Create an ent.Driver from `db`.
	driver := entsql.OpenDB(driverType, db)
	entLogger := log.NewEntLogger()
	c := ent.NewClient(ent.Driver(driver), ent.Log(entLogger.Log))

	initSchemaHooks(c)
	return driver, c
}

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

// EntPing ent driver ping
func EntPing() error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return defaultEntDriver.DB().PingContext(ctx)
}

// EntInitSchema 初始化schema
func EntInitSchema() (err error) {
	initSchemaOnce.Do(func() {
		err = defaultEntClient.Schema.Create(context.Background())
	})
	return
}

// EntGetStats get ent stats
func EntGetStats() map[string]interface{} {
	info := defaultEntDriver.DB().Stats()
	stats := map[string]interface{}{
		"maxOpenConns":      info.MaxOpenConnections,
		"openConns":         info.OpenConnections,
		"inUse":             info.InUse,
		"idle":              info.Idle,
		"waitCount":         info.WaitCount,
		"waitDuration":      info.WaitDuration,
		"maxIdleClosed":     info.MaxIdleClosed,
		"maxIdleTimeClosed": info.MaxIdleTimeClosed,
		"maxLifetimeClosed": info.MaxLifetimeClosed,
	}
	return stats
}

// EntGetClient get ent client
func EntGetClient() *ent.Client {
	return defaultEntClient
}
