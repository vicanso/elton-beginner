package cs

import "regexp"

const (
	// ResultSuccess result success
	ResultSuccess = iota
	// ResultFail result fail
	ResultFail
)

// ***处理
var MaskRegExp = regexp.MustCompile(`(?i)password`)
