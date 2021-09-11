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
	mask.RegExpOption(regexp.MustCompile(`password`)),
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
	account := util.GetAccount(ctx)
	// deviceID := util.GetDeviceID(ctx)
	if account == "" {
		return e
	}
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
