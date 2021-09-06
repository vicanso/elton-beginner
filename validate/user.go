package validate

func init() {
	// 用户账号
	AddAlias("xUserAccount", "ascii,min=2,max=10")
	// 用户密码
	AddAlias("xUserPassword", "ascii,min=6,max=50")
}
