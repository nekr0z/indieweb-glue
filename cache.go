package main

import (
	"encoding/base64"
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
