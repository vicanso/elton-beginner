---
description: 控制器是业务功能的入口，通过控制器指定路由对应的各中间件及处理函数，并调用各服务实现完整的业务功能
---

# 控制器

下面先简单的实现查询当前登录用户信息的controller，暂时未实现具体的查询逻辑，仅是一个controller的示例。

```go
package controller

import (
	"errors"

	"github.com/vicanso/beginner/router"
	"github.com/vicanso/elton"
)

type userCtrl struct{}

func init() {
	ctrl := userCtrl{}
	g := router.NewGroup("/users")

	// 当前登录信息查询
	g.GET("/v1/me", ctrl.me)

	// 客户列表查询
	// TODO 添加仅能管理员调用
	g.GET("/v1", ctrl.list)
}

func (*userCtrl) me(c *elton.Context) (err error) {
	// mock用户信息
	c.Body = &struct {
		Name string `json:"name"`
	}{
		Name: "test",
	}
	return
}

func (*userCtrl) list(c *elton.Context) (err error) {
	return errors.New("仅允许管理员访问")
}

```

如上面的代码所示，每个controller会实现其对应的一个struct，如`userCtrl`用于添加各路由的处理函数，一般命名时将功能名称作为前缀，避免多个功能的变量命名冲突。

路由的初始化均在`init`函数中处理，步骤一般如下：

- 创建ctrl，`ctrl := userCtrl{}`
- 初始化路由分组，`g := router.NewGroup("/users")`
- 对具体路由实现添加对应处理函数，`g.GET("/v1/me", ctrl.me)`


## 响应数据

示例中响应客户信息时，仅将数据赋值至`c.Body`中则可，之后访问`http://127.0.0.1:7001/users/v1/me`接口并没有返回任何数据，非预期的返回对应的json。

elton默认的并没有对`Body`的数据转换为输出数据，此响应的转换应该由开发者自定义中间件来实现，对于json的转换可以使用已实现好的中间件[]()，代码逻辑也简单，仅需要要elton实例中添加使用中间件即可。

```go
// ... 部分代码省略
	e.Use(middleware.NewDefaultResponder())
// ... 部分代码省略
```

增加此中间件之后，响应数据以期望的json返回。

```bash
curl 'http://127.0.0.1:7001/users/v1/me' -v
*   Trying 127.0.0.1...
* TCP_NODELAY set
* Connected to 127.0.0.1 (127.0.0.1) port 7001 (#0)
> GET /users/v1/me HTTP/1.1
> Host: 127.0.0.1:7001
> User-Agent: curl/7.64.1
> Accept: */*
>
< HTTP/1.1 200 OK
< Content-Type: application/json; charset=utf-8
< Date: Mon, 05 Jul 2021 10:33:42 GMT
< Content-Length: 15
<
* Connection #0 to host 127.0.0.1 left intact
{"name":"test"}* Closing connection 0
```