package middleware

import (
	"github.com/vicanso/beginner/cache"
	"github.com/vicanso/beginner/config"
	"github.com/vicanso/beginner/util"
	"github.com/vicanso/elton"
	session "github.com/vicanso/elton-session"
)

var scf = config.MustGetSessionConfig()

// NewSession new session middleware
func NewSession() elton.Handler {
	store := cache.GetRedisSession()
	return session.NewByCookie(session.CookieConfig{
		// 数据存储
		Store: store,
		// cookie是否签名认证
		Signed: true,
		// session有效期
		Expired: scf.TTL,
		// 生成session id
		GenID: util.GenXID,
		// cookie名称
		Name: scf.Key,
		// cookie目录
		Path: scf.CookiePath,
		// cookie的有效期
		MaxAge: int(scf.TTL.Seconds()),
		// 是否设置http only
		HttpOnly: true,
	})
}
