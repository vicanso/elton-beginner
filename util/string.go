package util

// CutRune 按rune截断字符串
func CutRune(str string, max int) string {
	result := []rune(str)
	if len(result) < max {
		return str
	}
	return string(result[:max]) + "..."
}
