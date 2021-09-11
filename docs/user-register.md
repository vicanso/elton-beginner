---
description: 注册用户是整个系统的根本，登录系统的用户比匿名用户有着更大的粘性，也更大可能为系统的深度使用者
---

# 易用性与安全性

注册用户的流程必须足够的简单快捷，过多的流程会极大影响用户体验，一般而言会使用短信方式来注册，简单而又能多获取一个可推送消息给客户的渠道。
对于非使用短信注册的形式，则尽可能只让客户输入账号与密码即可，对于其它的信息不要在每一次注册时强制客户填写，应该在后续使用中再提示客户完善。

## 短信注册

短信方式注册流程中，主要考虑如果避免注册接口被攻击导致过多发送短信骚扰用户，一般都是基于IP或者手机号码来限制注册次数，措施一般如下：

- 对于多次调用短信注册的IP，在一定次数之后增加图形验证码之类限制机器人行为
- 对于多次调用短信注册，但并未完成注册的IP，此类调用大概率为攻击，将此IP加入限制列表（如一小时内不可再调用注册服务）
- 短信注册可限制调用发送短信与校验短信为同一IP，避免攻击者碰撞
- 短信校验限制次数（如5次），避免攻击暴力尝试

## 账号密码注册

账号密码注册的方式，安全性没有短信注册方式安全，因此使用此方式时需要更多的考虑安全性，主要措施如下：

- 注册时账号需要唯一，注册时需要判断当前账号是否已注册，为了避免通过此方法去攻击获取已注册账号，对于注册服务需要对IP限制
- 密码不能以明文保存，最简单的方式是sha1(密码)此种形式，但是由于此类方式生成的hash值有可能与网上可获取的[彩虹表](https://zh.wikipedia.org/wiki/%E5%BD%A9%E8%99%B9%E8%A1%A8)一样，因此建议使用sha1(应用名称+密码)的形式来生成hash值，或者更严格的可使用sha1(账号+密码)的形式生成hash值，或者使用更复杂的加密函数，如：bcrypt。

## 账号注册实现

### 参数定义与校验

```bash
type userRegisterParams struct {
	// 账号
	Account string `json:"account" validate:"required,xUserAccount"`
	// 密码
	Password string `json:"password" validate:"required,xUserPassword"`
}
```

参数定义仅简单账号与密码，增加基本的类型校验，线上系统建议增加图形验证码来、IP频率调用限制等增强校验，避免机器人调用。

```go
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
		return err
	}
	c.Created(user)
	return nil
}
```

调用注册接口，成功注册账号：

```bash
curl -XPOST -d '{"account":"vicanso", "password":"96cae35ce8a9b0244178bf28e4966c2ce1b8385723a96a6b838858cdd6ca0a1e"}' \
  -H 'Content-Type:application/json;charset=utf-8' \
  'http://127.0.0.1:7001/users/v1/me'

{"id":1,"createdAt":"2021-09-03T15:38:58.405002+08:00","updatedAt":"2021-09-03T15:38:58.405003+08:00","account":"vicanso"}
```