# cod-concurrent-limiter

[![Build Status](https://img.shields.io/travis/vicanso/cod-concurrent-limiter.svg?label=linux+build)](https://travis-ci.org/vicanso/cod-concurrent-limiter)


Concurrent limiter for cod.

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
		Keys: []string{":ip"},
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