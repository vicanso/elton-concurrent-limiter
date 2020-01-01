# elton-concurrent-limiter

[![Build Status](https://img.shields.io/travis/vicanso/elton-concurrent-limiter.svg?label=linux+build)](https://travis-ci.org/vicanso/elton-concurrent-limiter)


Concurrent limiter for elton. It support to get lock value from five ways. `Client IP`, `QueryString`, `Request Header`, `Route Params` and `Post Body`.

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

	"github.com/vicanso/elton"

	concurrentLimiter "github.com/vicanso/elton-concurrent-limiter"
)

func main() {

	e := elton.New()
	m := new(sync.Map)
	limit := concurrentLimiter.New(concurrentLimiter.Config{
		Keys: []string{
			":ip",
			"h:X-Token",
			"q:type",
			"p:id",
			"account",
		},
		Lock: func(key string, c *elton.Context) (success bool, unlock func(), err error) {
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
	})

	e.POST("/login", limit, func(c *elton.Context) (err error) {
		time.Sleep(3 * time.Second)
		c.BodyBuffer = bytes.NewBufferString("hello world")
		return
	})

	err := e.ListenAndServe(":3000")
	if err != nil {
		panic(err)
	})
}
```

```bash
curl -XPOST 'http://127.0.0.1:7001/login'
```