package config

import (
	"bytes"
	"embed"
	"io"
	"os"
	"time"

	"github.com/go-playground/validator/v10"
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

// GetBasicConfig 获取基本配置信息
func GetBasicConfig() *BasicConfig {
	prefix := "basic."
	basicConfig := &BasicConfig{
		Name:         defaultViperX.GetString(prefix + "name"),
		RequestLimit: defaultViperX.GetUint(prefix + "requestLimit"),
		Listen:       defaultViperX.GetStringFromENV(prefix + "listen"),
		Prefixes:     defaultViperX.GetStringSlice(prefix + "prefixes"),
		Timeout:      defaultViperX.GetDuration(prefix + "timeout"),
	}
	mustValidate(basicConfig)
	return basicConfig
}
