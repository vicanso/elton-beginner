package main

import (
	"context"
	"net/http"
	"regexp"
	"time"

	humanize "github.com/dustin/go-humanize"
	"github.com/vicanso/beginner/config"
	_ "github.com/vicanso/beginner/controller"
	"github.com/vicanso/beginner/helper"
	"github.com/vicanso/beginner/log"
	"github.com/vicanso/beginner/router"
	"github.com/vicanso/beginner/util"
	"github.com/vicanso/elton"
	"github.com/vicanso/elton/middleware"
	"github.com/vicanso/hes"
	"go.uber.org/atomic"
)

var basicConfig = config.MustGetBasicConfig()

// 相关依赖服务的校验，主要是数据库等
func dependServiceCheck() (err error) {
	err = helper.RedisPing()
	if err != nil {
		return
	}
	err = helper.EntPing()
	if err != nil {
		return
	}
	// 程序启动后再执行init schema
	err = helper.EntInitSchema()
	if err != nil {
		return
	}
	return
}

func main() {
	e := elton.New()

	scf := config.MustGetSessionConfig()
	e.SignedKeys = &elton.RWMutexSignedKeys{}
	e.SignedKeys.SetKeys(scf.Keys)

	processingCount := atomic.NewInt32(0)
	// 所有中间件触发前调用
	e.OnBefore(func(c *elton.Context) {
		// 正在处理请求数+1
		processingCount.Inc()
		// 设置trace id
		ctx := util.SetTraceID(c.Context(), util.GenXID())
		c.WithContext(ctx)
	})
	e.OnDone(func(ctx *elton.Context) {
		// 正在处理请求数-1
		processingCount.Dec()
	})
	// 只有未被处理的error才会触发此回调
	// 一般的出错均由error中间件处理，不会触发此回调
	e.OnError(func(c *elton.Context, err error) {
		he := hes.Wrap(err)
		ip := c.RealIP()
		uri := c.Request.RequestURI
		// 可以针对实际场景输出更多的日志信息
		log.Error(c.Context()).
			Str("category", "exception").
			Str("ip", ip).
			Str("route", c.Route).
			Str("uri", uri).
			Err(he).
			Msg("")
		if he.Category == middleware.ErrRecoverCategory {
			// 设置不再处理接收到的请求
			// 等待10秒后退出程序
			// 因为会调用sleep，因此启用新的goroutine
			// 如果有数据库等，可关闭相应的连接
			go e.GracefulClose(10 * time.Second)
		}
	})

	// 自定义404与405的处理，一般404与405均是代码或被攻击时导致的
	// 因此可针对此增加相应的统计，便于及时确认问题
	e.NotFoundHandler = func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		w.Write([]byte("Not Found"))
	}
	e.MethodNotAllowedHandler = func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(405)
		w.Write([]byte("Method Not Allowed"))
	}

	// panic的恢复处理，放在最前
	e.Use(middleware.NewRecover())

	// 如果有配置应用超时设置
	if basicConfig.Timeout != 0 {
		// 仅将timeout设置给context，后续调用如果无依赖于context
		// 则不会超时
		// 后续再考虑是否增加select
		e.Use(func(c *elton.Context) error {
			ctx, cancel := context.WithTimeout(c.Context(), basicConfig.Timeout)
			defer cancel()
			c.WithContext(ctx)
			return c.Next()
		})
	}

	// 访问日志，其调用需要放在出错与响应之前，这样才能获取真实的响应数据与状态码
	e.Use(middleware.NewStats(middleware.StatsConfig{
		OnStats: func(si *middleware.StatsInfo, c *elton.Context) {
			log.Info(c.Context()).
				// 日志分类
				Str("category", "accessLog").
				Str("ip", si.IP).
				Str("method", si.Method).
				Str("route", si.Route).
				Str("uri", si.URI).
				// 响应状态码
				Int("status", si.Status).
				// 当前处理的请求数
				Uint32("connecting", si.Connecting).
				// 耗时
				Str("latency", si.Latency.String()).
				// 响应数据大小（格式化便于阅读）
				Str("size", humanize.Bytes(uint64(si.Size))).
				// 响应数据大小（字节）
				Int("bytes", si.Size).
				Msg("")
			return
		},
	}))

	// 数据压缩（需要放在responder中间件之后，它在responder转换响应数据后再压缩）
	config := middleware.NewCompressConfig(
		// 优先br
		new(middleware.BrCompressor),
		// 如果不指定最小压缩长度，则为1KB
		new(middleware.GzipCompressor),
	)
	// 配置针对哪此数据类型压缩
	config.Checker = regexp.MustCompile("text|javascript|json|wasm|font")
	e.Use(middleware.NewCompress(config))

	// eTag与fresh的处理（需配合使用并放在responder之前）
	e.Use(middleware.NewDefaultFresh()).
		Use(middleware.NewDefaultETag())

	// 出错处理
	e.Use(middleware.NewError(middleware.ErrorConfig{
		ResponseType: "json",
	}))
	// 响应数据转换处理
	e.Use(middleware.NewDefaultResponder())

	// json(application/json)+gzip(提交数据是经过gzip压缩）的body parser
	// 限制数据长度为50KB，如果想自定义配置查看中间件的说明
	e.Use(middleware.NewDefaultBodyParser())

	// 将初始化的分组路由添加到当前实例中
	e.AddGroup(router.GetGroups()...)

	err := dependServiceCheck()
	if err != nil {
		log.Error(context.Background()).
			Str("category", "depFail").
			Err(err).
			Msg("")
		return
	}

	addr := basicConfig.Listen
	log.Info(context.Background()).
		Str("addr", addr).
		Msg("server is running")
	// 监听端口
	err = e.ListenAndServe(addr)
	// 如果失败则直接panic，因为程序无法提供服务
	if err != nil {
		log.Error(context.Background()).
			Err(err).
			Msg("server listen fail")
		panic(err)
	}
}
