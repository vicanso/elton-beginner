---
description: 日志是应用系统的灵魂所在，在系统设计初期则应该考虑日志的相关规范与处理
---

# 日志-应用系统的灵魂

为什么日志是应用系统的灵魂呢？一个应用系统稳定的运行，业务逻辑正常时，它的外在表现就那么的光鲜靓丽，而当日志设计不合理，系统出现问题排查时才发现，内在的灵魂是那么的肮脏，残酷的事实让你明白『可远观而不可亵玩焉』。

## 日志关键要素

- 时间：记录该日志的创建时间
- 日志类型：记录该日志的类型，用于区分error，warn等类别的不同日志
- 日志级别控制：可指定输出的日志级别，方便本地开发时可以输出debug类日志
- 账户、请求链路等关键信息：日志中需指定客户信息等关键信息
- 支持针对隐私信息***处理


## 代码实现

对比[zap](https://github.com/uber-go/zap)与[zerolog](https://github.com/rs/zerolog)的实现与使用方式之后，我选择使用zerolog作为日志处理模块，实际应用时可根据应用需要选择不同的日志模块。

### 初始化日志实例

```go
package log

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strconv"

	"github.com/rs/zerolog"
	"github.com/vicanso/beginner/util"
	mask "github.com/vicanso/go-mask"
)

// 日志中值的最大长度
var logFieldValueMaxSize = 30

var logMask = mask.New(
	// 指定哪些日志需要处理为***
	mask.RegExpOption(regexp.MustCompile(`password`)),
	// 指定长度截断
	mask.MaxLengthOption(logFieldValueMaxSize),
)

type entLogger struct{}

func (el *entLogger) Log(args ...interface{}) {
	Info(context.Background()).
		Msg(fmt.Sprint(args...))
}

var defaultLogger = newLogger()

// newLogger 初始化logger
func newLogger() *zerolog.Logger {
	// 如果要节约日志空间，可以配置
	zerolog.TimestampFieldName = "t"
	zerolog.LevelFieldName = "l"
	zerolog.TimeFieldFormat = "2006-01-02T15:04:05.999Z07:00"

	var l zerolog.Logger
	if util.IsDevelopment() {
		l = zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout}).
			With().
			Timestamp().
			Logger()
	} else {
		l = zerolog.New(os.Stdout).
			Level(zerolog.InfoLevel).
			With().
			Timestamp().
			Logger()
	}

	// 如果有配置指定日志级别，则以配置指定的输出
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel != "" {
		lv, _ := strconv.Atoi(logLevel)
		l = l.Level(zerolog.Level(lv))
	}

	return &l
}

func fillTraceInfos(ctx context.Context, e *zerolog.Event) *zerolog.Event {
	traceID := util.GetTraceID(ctx)
	// 设置trace id，方便标记当前链路的日志
	if traceID != "" {
		e.Str("traceID", traceID)
	}
	account := util.GetAccount(ctx)
	if account == "" {
		return e
	}
	// 记录客户信息
	return e.Str("account", account)
}

func Info(ctx context.Context) *zerolog.Event {
	return fillTraceInfos(ctx, defaultLogger.Info())
}

func Error(ctx context.Context) *zerolog.Event {
	return fillTraceInfos(ctx, defaultLogger.Error())
}

func Debug(ctx context.Context) *zerolog.Event {
	return fillTraceInfos(ctx, defaultLogger.Debug())
}

func Warn(ctx context.Context) *zerolog.Event {
	return fillTraceInfos(ctx, defaultLogger.Warn())
}

// NewEntLogger create a ent logger
func NewEntLogger() *entLogger {
	return &entLogger{}
}
```

初始化日志实例的处理逻辑比较简单，根据不同的运行环境使用不同的配置以及日志输出级别等。每个日志函数均需要指定context，用于添加trace信息（如账号等）。

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
	log.Info(context.Background()).
		Str("addr", addr).
		Msg("server is running")
	// 监听端口
	err := e.ListenAndServe(addr)
	// 如果失败则直接panic，因为程序无法提供服务
	if err != nil {
		log.Error(context.Background()).
			Err(err).
			Msg("server listen fail")
		panic(err)
	}
}
```
