---
description: 使用elton创建HTTP服务
---

# 启动HTTP服务

Elton提供简单的方式监听端口提供http(s)服务，`ListenAndServe`提供http服务，`ListenAndServeTLS`则提供https服务，下面的示例是监听7001端口并提供http服务。

```go
package main

import (
	"time"

	"github.com/vicanso/elton"
)

func main() {
	e := elton.New()

	// 可根据应用场景调整http server的配置
	// 一般保持默认不调整即可
	e.Server.MaxHeaderBytes = 50 * 1024
	e.Server.IdleTimeout = 30 * time.Second

	// 监听端口
	err := e.ListenAndServe(":7001")
	// 如果失败则直接panic，因为程序无法提供服务
	if err != nil {
		panic(err)
	}
}
```
