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
