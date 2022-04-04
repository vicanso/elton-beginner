---
description: elton提供一些常用的事件监听，可根据此类事件添加一些全局的处理
---

## 当前处理中的请求数

elton提供了两组事件用于监控路由请求处理前、以及处理完成的回调监听函数，可用于记录处理时长、响应状态等信息(一般建议使用中间件处理)。下面的处理主要是用于记录当前处理请求数，记录此数据可以实时计算当前的请求量，根据其判断是否需要增加系统资源或是否代码有问题出现死锁逻辑导致请求一直未完成（如某个资源一直无法获取，也无超时处理时则并不会触发处理完成的回调）。

需要注意`OnBefore`只在有对应路由的时候才会触发，`404`与`405`的并不会触发此回调。

```go
// 省略部分代码
// ...
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
// ...
// 省略部分代码
```

## 出错回调

elton的使用一般通过中间件将error转化为对应的输出响应，但也存在部分处理未被中间件处理，此类出错会由elton本身的出错处理来转化输出响应。对于此类出错可以通过监听出错的回调，代码如下：

```go
// 省略部分代码
// ...
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
// ...
// 省略部分代码
```

## 404与405的定义处理

由于404与405并不会触发相关的回调，因此只能提供自定义的函数替换默认函数来实现自定义的逻辑，代码如下：

```go
// 省略部分代码
// ...
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
// ...
// 省略部分代码
```