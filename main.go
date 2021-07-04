package main

import (
	"github.com/vicanso/beginner/log"
	"github.com/vicanso/elton"
)

func main() {
	e := elton.New()

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
