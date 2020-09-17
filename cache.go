// Copyright (C) 2020 Evgeny Kuznetsov (evgeny@kuznetsov.md)
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along tihe this program. If not, see <https://www.gnu.org/licenses/>.

package main

import (
	"encoding/base64"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/memcachier/mc/v3"
)

type cache interface {
	get(key string) ([]byte, time.Time)
	set(key string, content []byte, exp time.Time)
}

type item struct {
	content []byte
	exp     time.Time
}

type memoryCache struct {
	items map[string]item
	mux   *sync.RWMutex
}

func newMemoryCache() *memoryCache {
	return &memoryCache{
		items: make(map[string]item),
		mux:   &sync.RWMutex{},
	}
}

func (c *memoryCache) get(key string) ([]byte, time.Time) {
	c.mux.RLock()
	defer c.mux.RUnlock()

	i := c.items[key]
	if i.exp.Before(time.Now()) {
		delete(c.items, key)
		return nil, time.Unix(0, 0)
	}
	return i.content, i.exp
}

func (c *memoryCache) set(key string, content []byte, exp time.Time) {
	c.mux.Lock()
	defer c.mux.Unlock()

	c.items[key] = item{
		content: content,
		exp:     exp,
	}
}

type mcCache struct {
	client *mc.Client
}

func newMcCache(cl *mc.Client) *mcCache {
	return &mcCache{
		client: cl,
	}
}

func (c *mcCache) get(key string) ([]byte, time.Time) {
	val, flag, _, err := c.client.Get(key)
	if err != nil {
		return nil, time.Unix(0, 0)
	}

	exp := time.Unix(int64(flag), 0)
	if exp.Before(time.Now()) {
		_ = c.client.Del(key)
		return nil, time.Unix(0, 0)
	}

	content, err := base64.StdEncoding.DecodeString(val)
	if err != nil {
		return nil, exp
	}

	return content, exp
}

func (c *mcCache) set(key string, content []byte, exp time.Time) {
	val := base64.StdEncoding.EncodeToString(content)
	unix := uint32(exp.Unix())

	_, _ = c.client.Set(key, val, unix, unix, 0)
}

func canCache(h http.Header) (bool, time.Time) {
	c := h.Values("Cache-Control")

	// if no Cache-Control is set, cache for 24 hours
	if len(c) == 0 {
		return true, time.Now().Add(time.Hour * 24)
	}

	if !containsStr(c, "public") {
		return false, time.Unix(0, 0)
	}

	for _, v := range c {
		if strings.HasPrefix(v, "max-age=") {
			seconds, err := strconv.Atoi(strings.TrimPrefix(v, "max-age="))
			if err != nil {
				return false, time.Unix(0, 0)
			}
			return true, time.Now().Add(time.Second * time.Duration(seconds))

		}
	}

	ex := h.Get("Expires")
	exp, err := time.Parse(time.RFC1123, ex)
	if err != nil {
		return true, exp
	}

	return false, time.Unix(0, 0)
}
