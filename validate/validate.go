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

func wrapError(err error) error {

	he := hes.Wrap(err)
	if he.Category == "" {
		he.Category = errCategory
	}
	he.StatusCode = http.StatusBadRequest
	return he
}

// Do 执行校验
func Do(s interface{}, data interface{}) error {
	err := doValidate(s, data)
	if err != nil {
		return wrapError(err)
	}
	return nil
}

// 对struct校验
func Struct(s interface{}) error {
	defaults.SetDefaults(s)
	err := defaultValidator.Struct(s)
	if err != nil {
		return wrapError(err)
	}
	return nil
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
