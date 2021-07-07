---
description: 日志是应用系统的灵魂所在，在系统设计初期则应该考虑日志的相关规范与处理
---

# 日志-应用系统的灵魂

为什么是日志是应用系统的灵魂呢？一个应用系统稳定的运行，业务逻辑正常时，它的外在表现就那么的光鲜靓丽，而当日志设计不合理，系统出现问题排查时才发现，原来它的灵魂是那么的肮脏，突然间你开始明白了什么是『可远观而不可亵玩焉』。

## 日志关键要素

- 时间：记录该日志的创建时间
- 日志类型：记录该日志的类型，用于区分error，warn等类别的不同日志
- 日志级别控制：可指定输出的日志级别，方便本地开发时可以输出debug类日志
- 账户等关键信息：日志中需指定客户信息等关键信息


## 代码实现

对比[zap](https://github.com/uber-go/zap)与[zerolog](https://github.com/rs/zerolog)的实现与使用方式之后，我选择使用zerolog作为日志处理模块。

### 初始化日志实例

```go
package log

import (
	"os"
	"strconv"

	"github.com/rs/zerolog"
	"github.com/vicanso/beginner/util"
)

type tracerHook struct{}

func (h tracerHook) Run(e *zerolog.Event, level zerolog.Level, msg string) {
	if level == zerolog.NoLevel {
		return
	}
	// TODO 实现相关重现信息的添加
}

var defaultLogger = newLogger()

// newLogger 初始化logger
func newLogger() *zerolog.Logger {
	// 如果要节约日志空间，可以配置
	zerolog.TimestampFieldName = "t"
	zerolog.LevelFieldName = "l"
	// 时间格式化
	zerolog.TimeFieldFormat = "2006-01-02T15:04:05.999Z07:00"

	var l zerolog.Logger
	// 开发环境以console writer的形式输出日志
	if util.IsDevelopment() {
		// 可根据需要精简日志输出
		// 如不添加tracer hook timestamp等
		l = zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout}).
			Hook(&tracerHook{}).
			With().
			Timestamp().
			Logger()
	} else {
		l = zerolog.New(os.Stdout).
			Level(zerolog.InfoLevel).
			Hook(&tracerHook{}).
			With().
			Timestamp().
			Logger()
	}

	// 如果指定了log level的级别，则指定日志级别
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel != "" {
		lv, _ := strconv.Atoi(logLevel)
		l = l.Level(zerolog.Level(lv))
	}

	return &l
}

// Default 获取默认的logger
func Default() *zerolog.Logger {
	return defaultLogger
}
```

初始化日志实例的处理逻辑比较简单，根据不同的运行环境使用不同的配置以及日志输出级别等。
注：日志模块中并没有讲述如果添加账户等关键信息（由tracer hook实现)，在后面章节中会再讲述。

### HTTP服务监听前输出监听日志

在最开始的http服务启动之后，程序并没有输出任何日志，这样并不方便确认程序是否成功运行（当前置依赖的处理增多之后），因此调整代码增加日志输出。

```go
package main

import (
	"github.com/vicanso/beginner/log"
	"github.com/vicanso/elton"
)

func main() {
	e := elton.New()

	addr := ":7001"
	logger := log.Default()
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
