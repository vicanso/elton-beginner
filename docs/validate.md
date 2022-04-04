---
description: 参数校验是系统最容易被忽略的一点，如何在参数校验严谨性与易用性上取得平衡则是需要考虑的一大重点
---

# 参数校验

参数校验使用了[validator](https://github.com/go-playground/validator)，其提供了各类常用的校验tag，可以使用`RegisterAlias`将此类tag组合使用，定义更符合业务场景的校验规则。当现有校验规则都不合适时，也可以使用`RegisterValidation`自定义完整的校验函数。

## 校验规则

对于接口的传参均需要校验，尽可能使用严格的规则以及自定义的形式来校验，严格的参数校验能尽可能避免一些潜在错误或被攻击的风险，使用自定义的校验规则则可通过配置形式忽略或重新定义规则。

下面主要介绍`doValidate`的处理，它包含了`Unmarshal`、`SetDefaults`以及`Validate`。validate的定义是通过struct tag定义的，需要先将数据赋值至struct中，WEB程序基本以json的形式传输数据，因此使用的是`json.Unmarshal`，并设置默认值，之后进行参数校验。代码如下：

```go
// doValidate 校验struct
func doValidate(s interface{}, data interface{}) error {
	if data != nil {
		buf, ok := data.([]byte)
		if !ok {
			tmp, err := json.Marshal(data)
			if err != nil {
				return err
			}
			buf = tmp
		}
		if len(buf) == 0 {
			return hes.New("data is empty", errJSONParseCategory)
		}
		err := json.Unmarshal(buf, s)
		if err != nil {
			he := hes.Wrap(err)
			he.Category = errJSONParseCategory
			return he
		}
	}
	// 设置默认值
	defaults.SetDefaults(s)
	return defaultValidator.Struct(s)
}
```

对参数增加校验之后，能提升程序的安全性，但也有可能因为对于参数规则的误解等原因导致设置了错误的校验规则，导致正常的用户也受影响，因此要添加规则时需要多认真了解所有参数的数据定义。

针对参数添加校验规则之后，后续需要关注的就是参数校验的出错有哪些，出错比例等等，因此需要在出错转换的时候，关注`json-parse`与`validate`这两类错误，增加监控以及定期整理确认完善校验规则即可。

如果真的出现规则错误，导致大批量用户不可用，可以通过ENV的形式动态调整规则，如`xAccount`的校验规则覆盖，可以在环境变量中设置`VALIDATE_xAccount=*`后，重启应用则可调整为`xAccount`的规则为允许任何参数，也可定义为自定义的规则，方便在原有定义规则有问题时覆盖。

## 用户参数校验

通过对每个模板中使用到的参数增加自定义校验，实现对参数校验的统一规范化的校验。

```go
package validate

func init() {
	// 用户账号
	AddAlias("xUserAccount", "ascii,min=2,max=10")
	// 用户密码
	AddAlias("xUserPassword", "ascii,min=6,max=50")
}
```