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
