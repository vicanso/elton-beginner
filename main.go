package main

import (
	_ "github.com/vicanso/beginner/controller"
	"github.com/vicanso/beginner/log"
	"github.com/vicanso/beginner/router"
	"github.com/vicanso/elton"
	"github.com/vicanso/elton/middleware"
)

func main() {
	e := elton.New()

	e.Use(middleware.NewDefaultResponder())

	// 将初始化的分组路由添加到当前实例中
	for _, g := range router.GetGroups() {
		e.AddGroup(g)
	}

	addr := ":7001"
	logger := log.Default()
	logger.Info().
		Str("addr", addr).
		Msg("server is running")
	// 监听端口
	err := e.ListenAndServe(addr)
	// 如果失败则直接panic，因为程序无法提供服务
	if err != nil {
		logger.Error().
			Err(err).
			Msg("server listen fail")
		panic(err)
	}
}
