package main

import (
	"bytes"
	"sync"
	"time"

	"github.com/vicanso/elton"

	concurrentLimiter "github.com/vicanso/elton-concurrent-limiter"
)

func main() {

	d := elton.New()
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

	d.POST("/login", limit, func(c *elton.Context) (err error) {
		time.Sleep(3 * time.Second)
		c.BodyBuffer = bytes.NewBufferString("hello world")
		return
	})

	d.ListenAndServe(":7001")
}
