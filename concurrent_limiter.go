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
	"errors"
	"net/http"
	"strings"

	jsoniter "github.com/json-iterator/go"
	"github.com/vicanso/elton"
	"github.com/vicanso/hes"
)

var (
	// errSubmitTooFrequently submit too frequently
	errSubmitTooFrequently = &hes.Error{
		StatusCode: http.StatusBadRequest,
		Message:    "submit too frequently",
		Category:   ErrCategory,
	}
	errRequireLockFunction = errors.New("require lock function")
	json                   = jsoniter.ConfigCompatibleWithStandardLibrary
)

const (
	ipKey     = ":ip"
	headerKey = "h:"
	queryKey  = "q:"
	paramKey  = "p:"
	// ErrCategory concurrent limiter error category
	ErrCategory = "elton-concurrent-limiter"
)

type (
	// Lock lock the key
	Lock func(string, *elton.Context) (bool, func(), error)
	// Config concurrent limiter config
	Config struct {
		// Keys keys for generate lock id
		Keys []string
		// Lock lock function
		Lock    Lock
		Skipper elton.Skipper
	}
	// keyInfo the concurrent key's info
	keyInfo struct {
		Name   string
		Params bool
		Query  bool
		Header bool
		Body   bool
		IP     bool
	}
)

// New create a concurrent limiter middleware
func New(config Config) elton.Handler {
	if config.Lock == nil {
		panic(errRequireLockFunction)
	}
	keys := make([]*keyInfo, 0)
	// 根据配置生成key的处理
	for _, key := range config.Keys {
		if key == ipKey {
			keys = append(keys, &keyInfo{
				IP: true,
			})
			continue
		}
		if strings.HasPrefix(key, headerKey) {
			keys = append(keys, &keyInfo{
				Name:   key[2:],
				Header: true,
			})
			continue
		}
		if strings.HasPrefix(key, queryKey) {
			keys = append(keys, &keyInfo{
				Name:  key[2:],
				Query: true,
			})
			continue
		}
		if strings.HasPrefix(key, paramKey) {
			keys = append(keys, &keyInfo{
				Name:   key[2:],
				Params: true,
			})
			continue
		}
		keys = append(keys, &keyInfo{
			Name: key,
			Body: true,
		})
	}
	skipper := config.Skipper
	if skipper == nil {
		skipper = elton.DefaultSkipper
	}
	keyLength := len(keys)
	return func(c *elton.Context) (err error) {
		if skipper(c) {
			return c.Next()
		}
		sb := new(strings.Builder)
		// 先申请假定每个value的长度
		sb.Grow(8 * keyLength)
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
			sb.WriteString(v)
			if i < keyLength-1 {
				sb.WriteRune(',')
			}
		}
		lockKey := sb.String()

		success, unlock, err := config.Lock(lockKey, c)
		if err != nil {
			err = hes.Wrap(err)
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
