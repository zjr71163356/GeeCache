package geecache

import (
	"GeeCache/lru"
	"sync"
)

// cache 是一个并发安全的缓存结构体，封装了 LRU 缓存策略。
type cache struct {
	mu         sync.Mutex
	cache      *lru.Cache
	cacheBytes int64
}

// add 方法向缓存中添加一个键值对。
//
// 此方法是并发安全的。如果内部的 lru.Cache 尚未初始化，
// 它会在此次调用中进行延迟初始化。
//
// 参数:
//
//	key: 要添加的键。
//	value: 与键关联的值。
func (c *cache) add(key string, value ByteView) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cache == nil {
		c.cache = lru.New(c.cacheBytes, nil)
	}
	c.cache.Add(key, value)

}

// get 方法根据键从缓存中查找对应的值。
//
// 此方法是并发安全的。如果缓存尚未初始化，它将直接返回零值。
//
// 参数:
//
//	key: 要查找的键。
//
// 返回值:
//
//	value: 查找到的值。如果未找到，则为空的 ByteView。
//	ok: 如果找到了键，则为 true；否则为 false。
func (c *cache) get(key string) (value ByteView, ok bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cache == nil {
		return
	}
	v, ok := c.cache.Get(key)
	if ok {
		return v.(ByteView), ok
	}
	return
}
