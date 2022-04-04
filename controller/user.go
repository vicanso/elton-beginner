package controller

import (
	"crypto/sha256"
	"errors"
	"fmt"

	"github.com/vicanso/beginner/ent/user"
	"github.com/vicanso/beginner/helper"
	M "github.com/vicanso/beginner/middleware"
	"github.com/vicanso/beginner/router"
	"github.com/vicanso/beginner/util"
	"github.com/vicanso/beginner/validate"
	"github.com/vicanso/elton"
	session "github.com/vicanso/elton-session"
	"github.com/vicanso/hes"
)

// 对应的所有函数均实现在此struct中
type userCtrl struct{}

// 注册参数
type userRegisterParams struct {
	// 账号
	Account string `json:"account" validate:"required,xUserAccount"`
	// 密码
	Password string `json:"password" validate:"required,xUserPassword"`
}

const (
	sessionTokenKey   = "token"
	sessionAccountKey = "account"
)

// 登录参数
type userLoginParams struct {
	// 账号
	Account string `json:"account" validate:"required,xUserAccount"`
	// 密码
	Password string `json:"password" validate:"required,xUserPassword"`
}

func init() {
	ctrl := userCtrl{}
	g := router.NewGroup(
		"/users",
		// 添加当前组共用中间件
		M.NewSession(),
	)

	// 当前登录信息查询
	g.GET("/v1/me", ctrl.me)
	// 注册用户
	g.POST("/v1/me", ctrl.register)

	// 获取登录token
	g.GET("/v1/login", ctrl.getLoginToken)
	// 登录用户
	g.POST("/v1/login", ctrl.login)
}

func (*userCtrl) me(c *elton.Context) error {
	se := session.MustGet(c)
	account := se.GetString(sessionAccountKey)
	c.Body = &struct {
		Name string `json:"name"`
	}{
		Name: account,
	}
	return nil
}

func (*userCtrl) getLoginToken(c *elton.Context) error {
	se := session.MustGet(c)
	// 生成随机token
	// 用于登录时用密码生成hash值
	token := util.GenXID()
	// 设置token至session中
	err := se.Set(c.Context(), sessionTokenKey, token)
	if err != nil {
		return err
	}

	c.Body = &struct {
		Token string `json:"token"`
	}{
		token,
	}
	return nil
}

func (*userCtrl) login(c *elton.Context) error {
	params := userLoginParams{}
	err := validate.Do(&params, c.RequestBody)
	if err != nil {
		return err
	}
	se := session.MustGet(c)
	user, err := helper.EntGetClient().User.Query().
		Where(user.AccountEQ(params.Account)).
		First(c.Context())
	if err != nil {
		return err
	}
	// 数据库中保存的密码已经是sha256
	token := se.GetString(sessionTokenKey)
	pwd := fmt.Sprintf("%x", sha256.Sum256([]byte(user.Password+token)))
	if params.Password != pwd {
		// 不直接提示密码错
		return hes.New("用户名或密码错误")
	}
	// 设置账号至session
	err = se.Set(c.Context(), sessionAccountKey, params.Account)
	if err != nil {
		return err
	}

	// 成功返回用户信息
	c.Body = user
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

	user, err := helper.EntGetClient().User.Create().
		SetAccount(params.Account).
		// 密码前端使用sha256(password)处理
		SetPassword(params.Password).
		Save(c.Context())

	if err != nil {
		// 可转换出错再转换为更友好的出错提示
		return err
	}
	c.Created(user)
	return nil
}
