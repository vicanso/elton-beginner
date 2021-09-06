package controller

import (
	"crypto/sha256"
	"errors"
	"fmt"

	"github.com/vicanso/beginner/helper"
	"github.com/vicanso/beginner/router"
	"github.com/vicanso/beginner/validate"
	"github.com/vicanso/elton"
)

type userCtrl struct{}

// 注册参数
type userRegisterParams struct {
	// 账号
	Account string `json:"account" validate:"required,xUserAccount"`
	// 密码
	Password string `json:"password" validate:"required,xUserPassword"`
}

func init() {
	ctrl := userCtrl{}
	g := router.NewGroup("/users")

	// 当前登录信息查询
	g.GET("/v1/me", ctrl.me)
	// 注册用户
	g.POST("/v1/me", ctrl.register)

	// 客户列表查询
	// TODO 添加仅能管理员调用
	g.GET("/v1", ctrl.list)
}

func (*userCtrl) me(c *elton.Context) error {
	// mock用户信息
	c.Body = &struct {
		Name string `json:"name"`
	}{
		Name: "test",
	}
	return nil
}

func (*userCtrl) list(c *elton.Context) error {
	return errors.New("仅允许管理员访问")
}

func (*userCtrl) register(c *elton.Context) error {
	params := userRegisterParams{}
	err := validate.Do(&params, c.RequestBody)
	if err != nil {
		return err
	}
	pwd := fmt.Sprintf("%x", sha256.Sum256([]byte(params.Password)))

	user, err := helper.EntGetClient().User.Create().
		SetAccount(params.Account).
		SetPassword(pwd).
		Save(c.Context())

	if err != nil {
		return err
	}
	c.Created(user)
	return nil
}
