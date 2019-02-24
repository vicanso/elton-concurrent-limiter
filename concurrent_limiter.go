// Copyright 2018 tree xie
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package concurrentlimiter

import (
	"net/http"
	"strings"

	jsoniter "github.com/json-iterator/go"
	"github.com/vicanso/cod"
	"github.com/vicanso/hes"
)

var (
	// errSubmitTooFrequently submit too frequently
	errSubmitTooFrequently = &hes.Error{
		StatusCode: http.StatusBadRequest,
		Message:    "submit too frequently",
		Category:   ErrCategoryConcurrentLimiter,
	}
	json = jsoniter.ConfigCompatibleWithStandardLibrary
)

const (
	ipKey     = ":ip"
	headerKey = "h:"
	queryKey  = "q:"
	paramKey  = "p:"
	// ErrCategoryConcurrentLimiter concurrent limiter error category
	ErrCategoryConcurrentLimiter = "cod-concurrent-limiter"
)

type (
	// Lock lock the key
	Lock func(string, *cod.Context) (bool, func(), error)
	// Config concurrent limiter config
	Config struct {
		// 生成limit key的相关参数
		Keys    []string
		Lock    Lock
		Skipper cod.Skipper
	}
	// KeyInfo the concurrent key's info
	KeyInfo struct {
		Name   string
		Params bool
		Query  bool
		Header bool
		Body   bool
		IP     bool
	}
)

// New create a concurrent limiter middleware
func New(config Config) cod.Handler {
	if config.Lock == nil {
		panic("require lock function")
	}
	keys := make([]*KeyInfo, 0)
	// 根据配置生成key的处理
	for _, key := range config.Keys {
		if key == ipKey {
			keys = append(keys, &KeyInfo{
				IP: true,
			})
			continue
		}
		if strings.HasPrefix(key, headerKey) {
			keys = append(keys, &KeyInfo{
				Name:   key[2:],
				Header: true,
			})
			continue
		}
		if strings.HasPrefix(key, queryKey) {
			keys = append(keys, &KeyInfo{
				Name:  key[2:],
				Query: true,
			})
			continue
		}
		if strings.HasPrefix(key, paramKey) {
			keys = append(keys, &KeyInfo{
				Name:   key[2:],
				Params: true,
			})
			continue
		}
		keys = append(keys, &KeyInfo{
			Name: key,
			Body: true,
		})
	}
	skipper := config.Skipper
	if skipper == nil {
		skipper = cod.DefaultSkipper
	}
	return func(c *cod.Context) (err error) {
		if skipper(c) {
			return c.Next()
		}
		values := make([]string, len(keys))
		// 获取 lock 的key
		for i, key := range keys {
			v := ""
			name := key.Name
			if key.IP {
				v = c.RealIP()
			} else if key.Header {
				v = c.GetRequestHeader(name)
			} else if key.Query {
				query := c.Query()
				v = query[name]
			} else if key.Params {
				v = c.Param(name)
			} else {
				body := c.RequestBody
				v = json.Get(body, name).ToString()
			}
			values[i] = v
		}
		lockKey := strings.Join(values, ",")

		success, unlock, err := config.Lock(lockKey, c)
		if err != nil {
			return
		}
		if !success {
			err = errSubmitTooFrequently
			return
		}

		if unlock != nil {
			defer unlock()
		}

		return c.Next()
	}
}
