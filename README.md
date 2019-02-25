# cod-concurrent-limiter

[![Build Status](https://img.shields.io/travis/vicanso/cod-concurrent-limiter.svg?label=linux+build)](https://travis-ci.org/vicanso/cod-concurrent-limiter)


Concurrent limiter for cod. It support to get lock value from five way. `Client IP`, `QueryString`, `Request Header`, `Route Params` and `Post Body`.

- `IP` The key's name is `:ip`
- `QueryString` The key's name has prefix `q:`
- `Request Header` The key's name has prefix `h:`
- `Route Params` The key's name has prefix `p:`
- `Post Body` The other's key

```go
package main

import (
	"bytes"
	"sync"
	"time"

	"github.com/vicanso/cod"

	concurrentLimiter "github.com/vicanso/cod-concurrent-limiter"
)

func main() {

	d := cod.New()
	d.Keys = []string{
		"cuttlefish",
	}
	m := new(sync.Map)
	d.Use(concurrentLimiter.New(concurrentLimiter.Config{
		Keys: []string{
			":ip",
			"h:X-Token",
			"q:type",
			"p:id",
			"account",
		},
		Lock: func(key string, c *cod.Context) (success bool, unlock func(), err error) {
			_, loaded := m.LoadOrStore(key, true)
			// the key not exists
			if !loaded {
				success = true
				unlock = func() {
					m.Delete(key)
				}
			}
			return
		},
	}))

	d.GET("/", func(c *cod.Context) (err error) {
		time.Sleep(3 * time.Second)
		c.BodyBuffer = bytes.NewBufferString("hello world")
		return
	})

	d.ListenAndServe(":7001")
}
```