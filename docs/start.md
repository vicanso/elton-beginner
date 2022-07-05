---
description: 使用elton创建HTTP服务
---

# 启动HTTP服务

Elton提供简单的方式监听端口提供http(s)服务，`ListenAndServe`提供http服务，`ListenAndServeTLS`则提供https服务，下面的示例是监听7001端口并提供http服务。

```go
package main

import (
	"time"

	"github.com/vicanso/elton"
)

func main() {
	e := elton.New()

	// 可根据应用场景调整http server的配置
	// 一般保持默认不调整即可
	e.Server.MaxHeaderBytes = 50 * 1024
	e.Server.IdleTimeout = 30 * time.Second

	// 监听端口
	err := e.ListenAndServe(":7001")
	// 如果失败则直接panic，因为程序无法提供服务
	if err != nil {
		panic(err)
	}
}
```

## HTTP2

golang自带的HTTP已经支持http2，因此服务仅需要支持https的形式则可：

```go
package main

import (
	"bytes"

	"github.com/vicanso/elton"
)

func main() {
	e := elton.New()
	e.GET("/", func(c *elton.Context) error {
		c.BodyBuffer = bytes.NewBufferString("Hello, World!")
		return nil
	})

	certFile := "/tmp/me.dev+3.pem"
	keyFile := "/tmp/me.dev+3-key.pem"
	err := e.ListenAndServeTLS(":3000", certFile, keyFile)
	if err != nil {
		panic(err)
	}
}
```

上面例子中证书是以文件的形式保存，使用时证书可能统一存储，加密访问（如保存在数据库等），下面的例子讲解如果使用[]byte来初始化TLS：

```go
package main

import (
	"bytes"
	"crypto/tls"
	"io/ioutil"

	"github.com/vicanso/elton"
)

// 获取证书内容
func getCert() (cert []byte, key []byte, err error) {
	// 此处仅简单从文件中读取，在实际使用，是从数据库中读取
	cert, err = ioutil.ReadFile("/tmp/me.dev+3.pem")
	if err != nil {
		return
	}
	key, err = ioutil.ReadFile("/tmp/me.dev+3-key.pem")
	if err != nil {
		return
	}
	return
}

func main() {
	e := elton.New()
	cert, key, err := getCert()
	if err != nil {
		panic(err)
	}
	// 先初始化tls配置，生成证书
	tlsConfig := &tls.Config{}
	tlsConfig.Certificates = make([]tls.Certificate, 1)
	tlsConfig.Certificates[0], err = tls.X509KeyPair(cert, key)
	if err != nil {
		panic(err)
	}
	e.Server.TLSConfig = tlsConfig

	e.GET("/", func(c *elton.Context) error {
		c.BodyBuffer = bytes.NewBufferString("hello world!")
		return nil
	})

	err = e.ListenAndServeTLS(":3000", "", "")
	if err != nil {
		panic(err)
	}
}
```

## H2C

默认的https需要在以https的方式提供，对于内部系统间的调用，如果希望以http的方式使用http2，那么可以考虑h2c的处理。下面的代码示例包括了服务端与客户端怎么使用以http的方式使用http2。

```go
package main

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/vicanso/elton"
	"github.com/vicanso/elton/middleware"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

var http2Client = &http.Client{
	// 强制使用http2
	Transport: &http2.Transport{
		// 允许使用http的方式
		AllowHTTP: true,
		// tls的dial覆盖
		DialTLS: func(network, addr string, cfg *tls.Config) (net.Conn, error) {
			return net.Dial(network, addr)
		},
	},
}

func main() {
	go func() {
		time.Sleep(time.Second)
		resp, err := http2Client.Get("http://127.0.0.1:3000/")
		if err != nil {
			panic(err)
		}
		fmt.Println(resp.Proto)
	}()

	e := elton.New()

	e.Use(middleware.NewDefaultResponder())

	e.GET("/", func(c *elton.Context) error {
		c.Body = "Hello, World!"
		return nil
	})
	// http1与http2均支持
	e.Server = &http.Server{
		Handler: h2c.NewHandler(e, &http2.Server{}),
	}

	err := e.ListenAndServe(":3000")
	if err != nil {
		panic(err)
	}
}
```

## HTTP3

http3现在支持的浏览器只有chrome canary以及firefox最新版本，虽然http3的标准方案已确定，但是需要注意http3模块的使用范围并不广泛，建议不要在正式环境中大规模使用。下面是使用[quic-go](https://github.com/lucas-clemente/quic-go)使用http3的示例：

```go
package main

import (
	"bytes"
	"crypto/tls"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/lucas-clemente/quic-go/http3"
	"github.com/vicanso/elton"
)

const listenAddr = ":4000"

// 获取证书内容
func getCert() (cert []byte, key []byte, err error) {
	// 此处仅简单从文件中读取，在实际使用，是从数据库中读取
	cert, err = ioutil.ReadFile("/tmp/me.dev+3.pem")
	if err != nil {
		return
	}
	key, err = ioutil.ReadFile("/tmp/me.dev+3-key.pem")
	if err != nil {
		return
	}
	return
}

func http3Get() {
	client := &http.Client{
		Transport: &http3.RoundTripper{},
	}
	resp, err := client.Get("https://127.0.0.1" + listenAddr + "/")
	if err != nil {
		log.Fatalln("http3 get fail ", err)
		return
	}
	log.Println("http3 get success", resp.Proto, resp.Status, resp.Header)
}

func main() {
	// 延时一秒后以http3的访问访问
	go func() {
		time.Sleep(time.Second)
		http3Get()
	}()

	e := elton.New()

	cert, key, err := getCert()
	if err != nil {
		panic(err)
	}
	// 先初始化tls配置，生成证书
	tlsConfig := &tls.Config{}
	tlsConfig.Certificates = make([]tls.Certificate, 1)
	tlsConfig.Certificates[0], err = tls.X509KeyPair(cert, key)
	if err != nil {
		panic(err)
	}
	e.Server.TLSConfig = tlsConfig.Clone()

	// 初始化http3服务
	http3Server := http3.Server{
		Server: &http.Server{
			Handler: e,
			Addr:    listenAddr,
		},
	}
	http3Server.TLSConfig = tlsConfig.Clone()

	e.Use(func(c *elton.Context) error {
		http3Server.SetQuicHeaders(c.Header())
		return c.Next()
	})

	e.GET("/", func(c *elton.Context) error {
		c.BodyBuffer = bytes.NewBufferString("hello " + c.Request.Proto + "!")
		return nil
	})

	go func() {
		// http3
		err := http3Server.ListenAndServe()
		if err != nil {
			panic(err)
		}
	}()

	// https
	err = e.ListenAndServeTLS(listenAddr, "", "")
	if err != nil {
		panic(err)
	}
}
```

## 小结

本章通过简单的示例介绍了elton启用HTTP, HTTPS, HTTP2 以及 HTTP3的方法，实际使用时一般仅使用HTTP的方式，而HTTPS或HTTP2/3等处理由前置的nginx处理。
