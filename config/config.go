package config

import "os"

var (
	env = os.Getenv("GO_ENV")
)

const (
	// Dev 开发模式下的环境变量
	Dev = "dev"
	// Test 测试环境下的环境变量
	Test = "test"
	// Production 生产环境下的环境变量
	Production = "production"
)

// GetENV 获取当前运行环境
func GetENV() string {
	if env == "" {
		return Dev
	}
	return env
}
