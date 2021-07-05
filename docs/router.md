---
description: 路由是HTTP服务的重要组成部分，如果合理的设计路由分组，路由组公共中间件能让项目更清晰简洁
---

# 路由

路由主要实现两个逻辑，接口分组以及分组的公共中间件。elton已提供了分组的路由处理，直接使用则可。

```go
package router

import (
	"github.com/vicanso/elton"
)

var (
	// groupList 路由组列表
	groupList = make([]*elton.Group, 0)
)

// 创建新的路由分组
func NewGroup(path string, handlerList ...elton.Handler) *elton.Group {
    // path为分组路由的前缀
	g := elton.NewGroup(path, handlerList...)
	groupList = append(groupList, g)
	return g
}

// 获取所有初始化的分组路由 
func GetGroups() []*elton.Group {
	return groupList
}
```

只此只提供了分组路由的创建，具体在各controller中使用。下面在实例elton的代码中添加分组路由。

```go
// ... 其它部分代码
	// 将初始化的分组路由添加到当前实例中
	for _, g := range router.GetGroups() {
		e.AddGroup(g)
	}
// ... 其它部分代码
```