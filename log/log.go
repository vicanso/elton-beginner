package log

import (
	"fmt"
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

type entLogger struct{}

func (el *entLogger) Log(args ...interface{}) {
	Default().Info().
		Msg(fmt.Sprint(args...))
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

// NewEntLogger create a ent logger
func NewEntLogger() *entLogger {
	return &entLogger{}
}
