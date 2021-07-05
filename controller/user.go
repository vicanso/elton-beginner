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
