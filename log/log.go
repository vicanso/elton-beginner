package log

import (
	"context"
	"fmt"
	"net/url"
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
	// 指定长度截断(如果不希望截断的，则可添加自定义处理)
	mask.MaxLengthOption(logFieldValueMaxSize),
	// 手机号码中间4位不展示
	mask.CustomMaskOption(regexp.MustCompile(`mobile`), func(key, value string) string {
		size := len(value)
		if size < 8 {
			return value
		}
		return value[0:size-8] + "****" + value[size-4:]
	}),
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

// URLValues create a url.Values log event
func URLValues(query url.Values) *zerolog.Event {
	if len(query) == 0 {
		return zerolog.Dict()
	}
	return zerolog.Dict().Fields(logMask.URLValues(query))
}

// Struct create a struct log event
func Struct(data interface{}) *zerolog.Event {
	if data == nil {
		return zerolog.Dict()
	}

	m, _ := logMask.Struct(data)

	return zerolog.Dict().Fields(m)
}

// NewEntLogger create a ent logger
func NewEntLogger() *entLogger {
	return &entLogger{}
}
