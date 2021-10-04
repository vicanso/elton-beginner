---
description: 系统配置
---

# 系统配置

对于应用系统，一般获取配置的方式有以下几种：

- 从环境变量中获取
- 启动中指定启动参数
- 从配置文件中获取
- 从数据库（mysql、etcd等）中获取

一般而已，对于密码等有保密性要求的参数，适合使用环境变量的形式设置。其它系统配置则通过配置文件获取，业务相关配置则以数据库配置为主，下面主要讲解如何从配置文件中加载配置。

由于在不同的环境运行时，使用的配置有可能不太一致，文件配置需要支持不同的配置可以加载不同的配置文件。在实际使用中，不同的环境的配置仅存在部分差异，因此又需要支持共用默认配置的形式。

配置文件选择使用yaml的格式，运行环境分为：`dev`, `test`, `production`，默认配置为`default`。使用[viperx](https://github.com/vicanso/viperx)来加载配置，此模块仅简单的增强了viper。

```go
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

// MustGetBasicConfig 获取基本配置信息
func MustGetBasicConfig() *BasicConfig {
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
```

通过go1.16新增支持的embed，将当前目录中的yml文件打包，默认先加载default.yml文件，之后再加载GO_ENV对应的yml文件，通过此方式实现共用配置与当前环境配置的合并。应用配置在各模块均有可能使用，因此直接初始化，所有引入它的模块均可直接使用。`mustLoadConfig`在加载配置失败时，会触发panic，各配置获取的时候会调用`mustValidate`，也会触发panic，因此获取配置应该直接一开始就初始化而非在函数中再获取。

需要注意，viper的处理只是当前配置获取不到时再去读取默认配置而并非真正的将两组配置合并，因此要获取时尽可能一个属性一个属性的获取。


```yml
# default.yml
# 系统基本配置
basic:
  name: forest
  # 系统并发限制，如果调整此限制，需要确认tracer中的大小也需要调整
  requestLimit: 100
  listen: :7001
  timeout: 30ss
```

```yml
# production.yml
basic:
  requestLimit: 1000
```

下面是main.go中获取应用配置的处理：

```go
package main

import (
	"regexp"

	humanize "github.com/dustin/go-humanize"
	"github.com/vicanso/beginner/config"
	_ "github.com/vicanso/beginner/controller"
	"github.com/vicanso/beginner/log"
	"github.com/vicanso/beginner/router"
	"github.com/vicanso/elton"
	compress "github.com/vicanso/elton-compress"
	"github.com/vicanso/elton/middleware"
	"github.com/vicanso/hes"
)

var basicConfig = config.MustGetBasicConfig()

func main() {
	// -- 略 --

	addr := basicConfig.Listen
	logger.Info().
		Str("addr", addr).
		Msg("server is running")
	// 监听端口
	err := e.ListenAndServe(addr)
	// 如果失败则直接panic，因为程序无法提供服务
	if err != nil {
		logger.Error().
			Err(err).
			Msg("server listen fail")
		panic(err)
	}
}
```