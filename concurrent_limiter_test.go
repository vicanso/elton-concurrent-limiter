package concurrentlimiter

import (
	"errors"
	"fmt"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/vicanso/cod"
)

func TestNoLockFunction(t *testing.T) {
	assert := assert.New(t)
	defer func() {
		r := recover()
		assert.Equal(r.(error), errRequireLockFunction)
	}()

	New(Config{})
}

func TestConcurrentLimiter(t *testing.T) {
	m := new(sync.Map)
	fn := New(Config{
		Keys: []string{
			":ip",
			"h:X-Token",
			"q:type",
			"p:id",
			"account",
		},
		Lock: func(key string, c *cod.Context) (success bool, unlock func(), err error) {
			if key != "192.0.2.1,xyz,1,123,tree.xie" {
				err = errors.New("key is invalid")
				return
			}
			_, loaded := m.LoadOrStore(key, 1)
			// 如果已存在，则获取销失败
			if loaded {
				return
			}
			success = true
			// 删除锁
			unlock = func() {
				m.Delete(key)
			}
			return
		},
	})

	req := httptest.NewRequest("POST", "/users/login?type=1", nil)
	resp := httptest.NewRecorder()
	c := cod.NewContext(resp, req)
	req.Header.Set("X-Token", "xyz")
	c.RequestBody = []byte(`{
		"account": "tree.xie"
	}`)
	c.Params = map[string]string{
		"id": "123",
	}

	t.Run("first", func(t *testing.T) {
		assert := assert.New(t)
		done := false
		c.Next = func() error {
			done = true
			return nil
		}
		err := fn(c)
		assert.Nil(err)
		assert.True(done)
	})

	t.Run("too frequently", func(t *testing.T) {
		assert := assert.New(t)
		done := false
		c.Next = func() error {
			time.Sleep(100 * time.Millisecond)
			done = true
			return nil
		}
		go func() {
			time.Sleep(10 * time.Millisecond)
			e := fn(c)
			assert.Equal(e.Error(), "category=cod-concurrent-limiter, message=submit too frequently")
		}()
		err := fn(c)
		// 登录限制,192.0.2.1,xyz,1,123,tree.xie
		assert.Nil(err)
		assert.True(done)
	})

	t.Run("lock function return error", func(t *testing.T) {
		assert := assert.New(t)
		c.Params = map[string]string{}
		err := fn(c)
		assert.Equal(err.Error(), "message=key is invalid")
	})
}

// https://stackoverflow.com/questions/50120427/fail-unit-tests-if-coverage-is-below-certain-percentage
func TestMain(m *testing.M) {
	// call flag.Parse() here if TestMain uses flags
	rc := m.Run()

	// rc 0 means we've passed,
	// and CoverMode will be non empty if run with -cover
	if rc == 0 && testing.CoverMode() != "" {
		c := testing.Coverage()
		if c < 0.9 {
			fmt.Println("Tests passed but coverage failed at", c)
			rc = -1
		}
	}
	os.Exit(rc)
}
