package main

import (
	"github.com/vicanso/elton"
)

func main() {
	e := elton.New()

	// 监听端口
	err := e.ListenAndServe(":7001")
	// 如果失败则直接panic，因为程序无法提供服务
	if err != nil {
		panic(err)
	}
}
