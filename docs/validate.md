---
description: 参数校验是系统最容易被忽略的一点，如何在参数校验严谨性与易用性上取得平衡则是需要考虑的一大重点
---

# 参数校验

参数校验使用了[validator](https://github.com/go-playground/validator)，其提供了各类常用的校验tag，可以使用`RegisterAlias`将此类tag组合使用，定义更符合业务场景的校验规则。当现有校验规则都不合适时，也可以使用`RegisterValidation`自定义完整的校验函数。

## 校验规则

一般而言对于接口的传参均需要使用校验，尽可能使用严格的规则以及自定义的形式来校验，严格的参数校验能尽可能避免一些潜在错误或被攻击的风险，使用自定义的校验规则则可通过配置形式忽略或重新定义规则。

```go
package validate

import (
	"encoding/json"
	"net/http"
	"os"

	"github.com/go-playground/validator/v10"
	"github.com/mcuadros/go-defaults"
	"github.com/vicanso/hes"
)

var (
	defaultValidator = validator.New()
	// validate默认的出错类别
	errCategory = "validate"
	// json parse失败时的出错类别
	errJSONParseCategory = "json-parse"
)

// doValidate 校验struct
func doValidate(s interface{}, data interface{}) (err error) {
	// statusCode := http.StatusBadRequest
	if data != nil {
		switch data := data.(type) {
		case []byte:
			if len(data) == 0 {
				he := hes.New("data is empty")
				he.Category = errJSONParseCategory
				err = he
				return
			}
			err = json.Unmarshal(data, s)
			if err != nil {
				he := hes.Wrap(err)
				he.Category = errJSONParseCategory
				err = he
				return
			}
		default:
			buf, err := json.Marshal(data)
			if err != nil {
				return err
			}
			err = json.Unmarshal(buf, s)
			if err != nil {
				return err
			}
		}
	}
	// 设置默认值
	defaults.SetDefaults(s)
	err = defaultValidator.Struct(s)
	return
}

func wrapError(err error) error {

	he := hes.Wrap(err)
	if he.Category == "" {
		he.Category = errCategory
	}
	he.StatusCode = http.StatusBadRequest
	return he
}

// Do 执行校验
func Do(s interface{}, data interface{}) (err error) {
	err = doValidate(s, data)
	if err != nil {
		return wrapError(err)
	}
	return
}

// 对struct校验
func Struct(s interface{}) (err error) {
	defaults.SetDefaults(s)
	err = defaultValidator.Struct(s)
	if err != nil {
		return wrapError(err)
	}
	return
}

// 任何参数均返回true，不校验。用于临时将某个校验禁用
func notValidate(fl validator.FieldLevel) bool {
	return true
}

func getCustomDefine(tag string) string {
	return os.Getenv("VALIDATE_" + tag)
}

// Add 添加一个校验函数
func Add(tag string, fn validator.Func, args ...bool) {
	custom := getCustomDefine(tag)
	if custom == "*" {
		_ = defaultValidator.RegisterValidation(tag, notValidate)
		return
	}
	if custom != "" {
		defaultValidator.RegisterAlias(tag, custom)
		return
	}
	err := defaultValidator.RegisterValidation(tag, fn, args...)
	if err != nil {
		panic(err)
	}
}

// AddAlias add alias
func AddAlias(alias, tags string) {
	custom := getCustomDefine(alias)
	if custom == "*" {
		_ = defaultValidator.RegisterValidation(alias, notValidate)
		return
	}
	if custom != "" {
		tags = custom
	}
	defaultValidator.RegisterAlias(alias, tags)
}
```

代码中主要实现了将`[]byte`或者`query`执行`Unmarshal`之后，设置相关的默认值，并执行参数校验。如果对于某个自定义的校验tag，如`xAccount`的校验规则覆盖，可以在环境变量中设置`VALIDATE_xAccount=*`后，重启应用则可调整为`xAccount`的规则为允许任何参数，也可定义为自定义的规则，方便在原有定义规则有问题时覆盖。

默认值设置使用[go-defaults](https://github.com/mcuadros/go-defaults)

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