---
description: 用户登录比注册使用频率更高，更需要对安全性上增加更多的考虑
---

# 安全性

用户登录的流程中必须尽可能保护客户密码的安全性，一般而言会直接直接选用https对传输数据做加密处理，而应用层面则需要考虑以下方面：

- 密码参数不要直接使用原始密码，避免中间人攻击等就方式获取到传输数据或应用中输出了登录参数等形为导致原始密码泄露
- 每次登录时通过加盐方式生成密码hash值，避免数据重放登录
- 增加频率限制，如：限制IP每天登录次数，多次登录失败后则锁定账号等
- 增加图形验证码等校验方式，避免自动化攻击

## 账号登录实现

### session中间件

session的中间件可以直接使用elton-session，可以通过自定义store实例session的存储，一般常用redis。

```go
package middleware

import (
	"github.com/vicanso/beginner/cache"
	"github.com/vicanso/beginner/config"
	"github.com/vicanso/beginner/util"
	"github.com/vicanso/elton"
	session "github.com/vicanso/elton-session"
)

var scf = config.MustGetSessionConfig()

// NewSession new session middleware
func NewSession() elton.Handler {
	store := cache.GetRedisSession()
	return session.NewByCookie(session.CookieConfig{
		// 数据存储
		Store: store,
		// cookie是否签名认证
		Signed: true,
		// session有效期
		Expired: scf.TTL,
		// 生成session id
		GenID: util.GenXID,
		// cookie名称
		Name: scf.Key,
		// cookie目录
		Path: scf.CookiePath,
		// cookie的有效期
		MaxAge: int(scf.TTL.Seconds()),
		// 是否设置http only
		HttpOnly: true,
	})
}
```

调整`NewGroup`添加session中间件，因此创建组的代码调整如下：

```go
	g := router.NewGroup(
		"/users",
		M.NewSession(),
	)
```


### 生成每次登录时使用的token

每次登录先先获取生成随机的token，用于密码hash时加盐处理，下面的代码则是生成token并保存至session。

路由定义：

```go
	g.GET("/v1/login", ctrl.getLoginToken)
```

controller的处理：

```go
func (*userCtrl) getLoginToken(c *elton.Context) error {
	se := session.MustGet(c)
	// 生成随机token
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
```

### 用户登录

登录时客户端需要将密码做hash处理：sha256(sha256(用户密码) + token)，下面的代码为校验用户账号与密码，校验成功后则将用户账号写入session。

路由定义：

```go
	g.POST("/v1/login", ctrl.login)
```

controller的处理：

```go
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
```

### 获取用户信息

当成功登录后则可通过从session中获取当前登录的账号，代码逻辑如下。

路由定义：

```go
	g.GET("/v1/me", ctrl.me)
```

controller的处理：

```go
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
```
