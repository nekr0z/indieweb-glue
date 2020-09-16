package main

import (
	"sync"
	"time"
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

func (c memoryCache) get(key string) ([]byte, time.Time) {
	c.mux.RLock()
	defer c.mux.RUnlock()

	i := c.items[key]
	if i.exp.Before(time.Now()) {
		delete(c.items, key)
		return nil, time.Now()
	}
	return i.content, i.exp
}

func (c memoryCache) set(key string, content []byte, exp time.Time) {
	c.mux.Lock()
	defer c.mux.Unlock()

	c.items[key] = item{
		content: content,
		exp:     exp,
	}
}
