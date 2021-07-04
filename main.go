package main

import (
	"bytes"
	"fmt"
	"time"

	"github.com/vicanso/elton"
)

func main() {
	e := elton.New()
	responseTimeKey := "responseTime"
	// 日志
	e.Use(func(c *elton.Context) error {
		// 转至下一个处理函数(或中间件)
		err := c.Next()
		fmt.Println(fmt.Sprintf("%s %s - %s", c.Request.Method, c.Request.RequestURI, c.GetDuration(responseTimeKey)))
		return err
	})
	// 响应时间
	e.Use(func(c *elton.Context) error {
		// 记录开始时间
		start := time.Now()
		err := c.Next()
		// 根据开始时间计算响应时长
		c.Set(responseTimeKey, time.Since(start))
		return err
	})
	// /ping url的具体处理
	e.GET("/ping", func(c *elton.Context) error {
		c.Body = bytes.NewBufferString("pong")
		return nil
	})
	// 监听端口
	err := e.ListenAndServe(":7001")
	if err != nil {
		panic(err)
	}
}
