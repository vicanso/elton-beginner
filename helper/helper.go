package helper

type ContextKey struct{}

var (
	// 记录命令开始时间
	startedAtKey *ContextKey = &ContextKey{}
)
