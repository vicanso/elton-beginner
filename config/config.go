package config

import (
	"bytes"
	"embed"
	"io"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/spf13/cast"
	"github.com/vicanso/viperx"
)

//go:embed *.yml
var configFS embed.FS

var (
	// 当前运行环境
	env              = os.Getenv("GO_ENV")
	defaultValidator = validator.New()
	defaultViperX    = mustLoadConfig()
)

const (
	// Dev 开发模式下的环境变量
	Dev = "dev"
	// Test 测试环境下的环境变量
	Test = "test"
	// Production 生产环境下的环境变量
	Production = "production"
)

type (
	// BasicConfig 应用基本配置信息
	BasicConfig struct {
		// 监听地址
		Listen string `validate:"required,ascii"`
		// 最大处理请求数
		RequestLimit uint `validate:"required"`
		// 应用名称
		Name string `validate:"required,ascii"`
		// 应用前缀
		Prefixes []string `validate:"omitempty"`
		// 超时（用于设置所有请求)
		Timeout time.Duration
	}

	// RedisConfig redis配置
	RedisConfig struct {
		// 连接地址
		Addrs []string `validate:"required,dive,hostname_port"`
		// 用户名
		Username string
		// 密码
		Password string
		// 慢请求时长
		Slow time.Duration `validate:"required"`
		// 最大的正在处理请求量
		MaxProcessing uint32 `validate:"required"`
		// 连接池大小
		PoolSize int
		// sentinel模式下使用的master name
		Master string
	}
	// DatabaseConfig 数据库配置
	DatabaseConfig struct {
		// 连接串
		URI string `validate:"required"`
		// 最大连接数
		MaxOpenConns int `default:"100"`
		// 最大空闲连接数
		MaxIdleConns int `default:"10"`
		// 最大空闲时长
		MaxIdleTime time.Duration `default:"5m"`
	}

	// SessionConfig session相关配置信息
	SessionConfig struct {
		// cookie的保存路径
		CookiePath string `validate:"required,ascii"`
		// cookie的key
		Key string `validate:"required,ascii"`
		// cookie的有效期
		TTL time.Duration `validate:"required"`
		// 用于加密cookie的key
		Keys []string `validate:"required"`
	}
)

// GetENV 获取当前运行环境
func GetENV() string {
	if env == "" {
		return Dev
	}
	return env
}

// 对数据校验，如果出错则panic，仅用于初始化时的配置检查
func mustValidate(v interface{}) {
	err := defaultValidator.Struct(v)
	if err != nil {
		panic(err)
	}
}

// 加载配置，出错是则抛出panic
func mustLoadConfig() *viperx.ViperX {
	configType := "yml"
	defaultViperX := viperx.New(configType)

	readers := make([]io.Reader, 0)
	for _, name := range []string{
		// 配置的顺序需要固定
		// 后面的配置相同属性覆盖前一个配置
		"default",
		GetENV(),
	} {
		data, err := configFS.ReadFile(name + "." + configType)
		if err != nil {
			panic(err)
		}
		readers = append(readers, bytes.NewReader(data))
	}

	// 加载配置
	err := defaultViperX.ReadConfig(readers...)
	if err != nil {
		panic(err)
	}
	return defaultViperX
}

// MustGetBasicConfig 获取基本配置信息
func MustGetBasicConfig() *BasicConfig {
	prefix := "basic."
	basicConfig := &BasicConfig{
		Name:         defaultViperX.GetString(prefix + "name"),
		RequestLimit: defaultViperX.GetUint(prefix + "requestLimit"),
		// 端口优先读取env，若未指定则读取配置文件
		Listen:   defaultViperX.GetStringFromENV(prefix + "listen"),
		Prefixes: defaultViperX.GetStringSlice(prefix + "prefixes"),
		// 超时优先读取env，若未指定则读取配置文件
		Timeout: defaultViperX.GetDurationFromENV(prefix + "timeout"),
	}
	mustValidate(basicConfig)
	return basicConfig
}

// MustGetRedisConfig 获取redis的配置
func MustGetRedisConfig() *RedisConfig {
	prefix := "redis."
	// redis配置优先读取env
	// 建议数据库类配置则都使用env的形式配置
	uri := defaultViperX.GetStringFromENV(prefix + "uri")
	uriInfo, err := url.Parse(uri)
	if err != nil {
		panic(err)
	}
	// 获取密码
	password, _ := uriInfo.User.Password()
	username := uriInfo.User.Username()

	query := uriInfo.Query()
	// 获取slow设置的时间间隔
	slowValue := query.Get("slow")
	slow := 100 * time.Millisecond
	if slowValue != "" {
		slow, err = time.ParseDuration(slowValue)
		if err != nil {
			panic(err)
		}
	}

	// 获取最大处理数的配置
	maxProcessing := 1000
	maxValue := query.Get("maxProcessing")
	if maxValue != "" {
		maxProcessing, err = strconv.Atoi(maxValue)
		if err != nil {
			panic(err)
		}
	}

	// 转换失败则为0
	// 连接池大小
	poolSize, _ := strconv.Atoi(query.Get("poolSize"))

	redisConfig := &RedisConfig{
		Addrs:         strings.Split(uriInfo.Host, ","),
		Username:      username,
		Password:      password,
		Slow:          slow,
		MaxProcessing: uint32(maxProcessing),
		PoolSize:      poolSize,
		Master:        query.Get("master"),
	}

	mustValidate(redisConfig)
	return redisConfig
}

// MustGetPostgresConfig 获取数据库配置
func MustGetDatabaseConfig() *DatabaseConfig {
	prefix := "database."
	// 优先读取env
	uri := defaultViperX.GetStringFromENV(prefix + "uri")
	maxIdleConns := 0
	maxOpenConns := 0
	var maxIdleTime time.Duration
	arr := strings.Split(uri, "?")
	if len(arr) == 2 {
		query, _ := url.ParseQuery(arr[1])
		maxIdleConns = cast.ToInt(query.Get("maxIdleConns"))
		maxOpenConns = cast.ToInt(query.Get("maxOpenConns"))
		maxIdleTime = cast.ToDuration(query.Get("maxIdleTime"))
		query.Del("maxIdleConns")
		query.Del("maxOpenConns")
		query.Del("maxIdleTime")
		uri = arr[0]
		s := query.Encode()
		if s != "" {
			uri += ("?" + s)
		}
	}

	databaseConfig := &DatabaseConfig{
		URI:          uri,
		MaxIdleConns: maxIdleConns,
		MaxOpenConns: maxOpenConns,
		MaxIdleTime:  maxIdleTime,
	}
	mustValidate(databaseConfig)
	return databaseConfig
}

// MustGetSessionConfig 获取session的配置
func MustGetSessionConfig() *SessionConfig {
	prefix := "session."
	sessConfig := &SessionConfig{
		TTL:        defaultViperX.GetDurationFromENV(prefix + "ttl"),
		Key:        defaultViperX.GetStringFromENV(prefix + "key"),
		CookiePath: defaultViperX.GetStringFromENV(prefix + "path"),
		Keys:       defaultViperX.GetStringSliceFromENV(prefix + "keys"),
	}
	mustValidate(sessConfig)
	return sessConfig
}
