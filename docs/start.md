---
description: 使用elton创建HTTP服务
---

# 启动HTTP服务

Elton提供简单的方式监听端口提供http(s)服务，`ListenAndServe`提供http服务，`ListenAndServeTLS`则提供https服务，下面的示例是监听7001端口并提供http服务。

```go
package main

import (
	"github.com/vicanso/elton"
)

func main() {
	e := elton.New()

	// 监听端口
	err := e.ListenAndServe(":7001")
	// 如果失败则直接panic，因为程序无法提供服务
	if err != nil {
		panic(err)
	}
}
```
