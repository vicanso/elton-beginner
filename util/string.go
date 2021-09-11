package util

import (
	"strings"

	"github.com/rs/xid"
)

// CutRune 按rune截断字符串
func CutRune(str string, max int) string {
	result := []rune(str)
	if len(result) < max {
		return str
	}
	return string(result[:max]) + "..."
}

// GenXID 生成随机id
func GenXID() string {
	return strings.ToUpper(xid.New().String())
}
