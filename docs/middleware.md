---
description: 中间件的机制能让代码的复用率提高，合理的利用全局中间件、分组中间件快速便捷的实现各业务流程
---

# 中间件

elton的中间件模块提供了各类常用的中间件，下面来选择一些添加至当前应用服务中。

## 日志

logger中间件可以方便输出访问日志，其日志输出仅需配置格式化模板则可，如`{method} {url} {status} {size-human} - {latency-ms} ms`。

当前项目的日志输出使用的是json的形式，希望能更细化的输出访问日志，每个属性单独输出便于后续数据分析，因此选择使用`stats`中间件来获取数据生成访问日志。

```go
// ... 省略部分代码
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
// ... 省略部分代码
```

## 异常恢复

当程序触发panic时，合理的操作是捕获此panic后，记录相关日志或发送告警，之后程序以优雅的方式退出(重新启动依赖于守护进程，如docker)。recover中间件简单的获取panic的出错信息，根据客户端的accept属性返回json或text，并触发一个类型为`ErrRecoverCategory`的error事件，可监听出错事件，并判断其类型，对于此类出错则程序退出。


```go
// ... 省略部分代码
	// 只有未被处理的error才会触发此回调
	// 一般的出错均由error中间件处理，不会触发此回调
	e.OnError(func(c *elton.Context, err error) {
		he := hes.Wrap(err)
		ip := c.RealIP()
		uri := c.Request.RequestURI
		// 可以针对实际场景输出更多的日志信息
		log.Default().Error().
			Str("category", "exception").
			Str("ip", ip).
			Str("route", c.Route).
			Str("uri", uri).
			Msg(he.Error())

		if he.Category == middleware.ErrRecoverCategory {
			// 设置不再处理接收到的请求
			// 等待10秒后退出程序
			// 因为会调用sleep，因此启用新的goroutine
			// 如果有数据库等，可关闭相应的连接
			go e.GracefulClose(10 * time.Second)
		}
	})
	// panic的恢复处理，放在最前
	e.Use(middleware.NewRecover())
// ... 省略部分代码
```

## 请求数据解析

elton中`RequestBody`保存请求提交的字节数据，默认时并没有对请求数据读取解析，根据实际应用时不同数据类型实现自定义的body parser中间件，elton实现了一个json的body parser，可以基于它来扩展更多数据类型的parser。

```go
// ... 省略部分代码
	// json(application/json)+gzip(提交数据是经过gzip压缩）的body parser
	// 限制数据长度为50KB，如果想自定义配置查看中间件的说明
	e.Use(middleware.NewDefaultBodyParser())
// ... 省略部分代码
```

## 响应数据压缩

http服务中数据响应绝大部分均为文本类数据，可通过压缩的方式减少数据传输。基本所有的浏览器均支持`gzip`压缩，而绝大部分浏览器也支持`br`压缩。如果仅使用`gzip`压缩，可以直接使用默认的压缩中间件。选择压缩算法的可以基于以下的思路：

1、如果客户端是浏览器的，基本只能选择`br`与`gzip`，尽可能选择高压缩率（因为CPU一般比网络资源较为便宜，而现在大多数的应用均非计算密集型）
2、对于内部应用，现代的网络设备能满足大量的数据传输需要，一般不需要考虑数据压缩。不过对于跨IDC的传输，专线费用较为昂贵，因此可以考虑对于这些跨IDC的访问使用snappy或者zstd压缩后再传输，减少带宽的占用并提升传输性能
3、数据压缩可基于应用场景、数据长度等选择不同的压缩率，实现更动态化的处理。例如在CPU资源较多时，选择高压缩率，CPU不足时选择低压缩率或不压缩


```go
// ... 省略部分代码
	// 数据压缩（需要放在responder中间件之后，它在responder转换响应数据后再压缩）
	config := middleware.NewCompressConfig(
		// 优先br
		new(middleware.BrCompressor),
		// 如果不指定最小压缩长度，则为1KB
		new(middleware.GzipCompressor),
	)
	// 配置针对哪此数据类型压缩
	config.Checker = regexp.MustCompile("text|javascript|json|wasm|font")
	e.Use(middleware.NewCompress(config)))
// ... 省略部分代码
```

如果需要支持其它的压缩方式，则可使用[elton-compress](https://github.com/vicanso/elton-compress)来添加更多的压缩方式，它支持`snappy`以及`zstd`等压缩，snappy等可用于内部服务之间调用。

```go
// ... 省略部分代码
	// 数据压缩（需要放在responder中间件之后，它在responder转换响应数据后再压缩）
	config := middleware.NewCompressConfig(
		// 优先snappy，压缩速度快，压缩率较低
		&compress.SnappyCompressor{
			MinLength: 1024,
		},
		// 如果不指定最小压缩长度，则为1KB
		new(middleware.GzipCompressor),
	)
	// 配置针对哪此数据类型压缩
	config.Checker = regexp.MustCompile("text|javascript|json|wasm|font")
	e.Use(middleware.NewCompress(config))
// ... 省略部分代码
```

## 304

如果HTTP服务提供的数据不经常变化，但又不希望客户端直接使用缓存时，可以增加etag与fresh的减少数据传输。

```go
// ... 省略部分代码
	// eTag与fresh的处理（需配合使用并放在responder之前）
	e.Use(middleware.NewDefaultFresh()).
		Use(middleware.NewDefaultETag())
// ... 省略部分代码
```

## 更多的中间件

elton提供了10多个中间件的实现，具体可参考[常用中间件](https://treexie.gitbook.io/elton/middlewares)。
