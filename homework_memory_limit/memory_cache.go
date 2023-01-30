package cache

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

var (
	errKeyNotFound = errors.New("cache: key not found")
)

type MemoryMapCacheOption func(cache *MemoryMapCache)

type MemoryMapCache struct {
	data      map[string]*item
	mutex     sync.RWMutex
	close     chan struct{}
	onEvicted func(key string, val []byte)
}

func NewMemoryMapCache(interval time.Duration, opts ...MemoryMapCacheOption) *MemoryMapCache {
	res := &MemoryMapCache{
		data:  make(map[string]*item, 100),
		close: make(chan struct{}),
	}

	for _, opt := range opts {
		opt(res)
	}

	go func() {
		ticker := time.NewTicker(interval)
		for {
			select {
			case t := <-ticker.C:
				res.mutex.Lock()
				i := 0
				for key, val := range res.data {
					if i > 10000 {
						break
					}
					if val.deadlineBefore(t) {
						res.delete(key)
					}
					i++
				}
				res.mutex.Unlock()
			case <-res.close:
				return
			}
		}
	}()

	return res
}

func MemoryMapCacheWithEvictedCallback(fn func(key string, val []byte)) MemoryMapCacheOption {
	return func(cache *MemoryMapCache) {
		cache.onEvicted = fn
	}
}

func (c *MemoryMapCache) Set(ctx context.Context, key string, val []byte, expiration time.Duration) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.set(key, val, expiration)
}

func (c *MemoryMapCache) set(key string, val []byte, expiration time.Duration) error {
	var dl time.Time
	if expiration > 0 {
		dl = time.Now().Add(expiration)
	}
	c.data[key] = &item{
		val:      val,
		deadline: dl,
	}
	return nil
}

func (c *MemoryMapCache) Get(ctx context.Context, key string) ([]byte, error) {
	c.mutex.RLock()
	res, ok := c.data[key]
	c.mutex.RUnlock()
	if !ok {
		return nil, fmt.Errorf("%w, key: %s", errKeyNotFound, key)
	}
	now := time.Now()
	if res.deadlineBefore(now) {
		c.mutex.Lock()
		defer c.mutex.Unlock()
		res, ok = c.data[key]
		if !ok {
			//return nil, errKeyNotFound
			return nil, fmt.Errorf("%w, key: %s", errKeyNotFound, key)
		}
		if res.deadlineBefore(now) {
			c.delete(key)
			return nil, fmt.Errorf("%w, key: %s", errKeyNotFound, key)
		}
	}
	return res.val, nil
}

func (c *MemoryMapCache) Delete(ctx context.Context, key string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.delete(key)
	return nil
}

func (c *MemoryMapCache) LoadAndDelete(ctx context.Context, key string) ([]byte, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	val, ok := c.data[key]
	if !ok {
		return nil, errKeyNotFound
	}
	c.delete(key)
	return val.val, nil
}

func (c *MemoryMapCache) delete(key string) {
	itm, ok := c.data[key]
	if !ok {
		return
	}
	delete(c.data, key)
	c.onEvicted(key, itm.val)
}

// 我要是调用两次 close?
func (c *MemoryMapCache) Close() error {
	select {
	case c.close <- struct{}{}:
	default:
		return errors.New("重复关闭")
	}
	return nil
}

type item struct {
	val      []byte
	deadline time.Time
}

func (i *item) deadlineBefore(t time.Time) bool {
	return !i.deadline.IsZero() && i.deadline.Before(t)
}

func (c *MemoryMapCache) OnEvicted(fn func(key string, val []byte)) {
	c.onEvicted = fn
}
