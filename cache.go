package geecache

import "container/list"

// 使用LRU算法实现队列中最近使用最少的节点的淘汰
type Cache struct {
	maxBytes  int64                         //表示缓存能存储的最大字节数上限
	nBytes    int64                         //已经存储的字节数
	ll        *list.List                    //缓存队列
	cache     map[string]*list.Element      //存储队列中节点的地址
	OnEvicted func(key string, value Value) //删除节点时的回调函数
}

type Value interface {
	Len() int64
}

type Entry struct {
	key   string
	value Value
}

func NewCache(maxBytes int64, OnEvicted func(key string, value Value)) *Cache {
	return &Cache{
		maxBytes:  maxBytes,
		ll:        list.New(),
		cache:     make(map[string]*list.Element),
		OnEvicted: OnEvicted,
	}
}

func (c *Cache) allocate(node *Entry) {
	c.nBytes += int64(node.value.Len()) + int64(len(node.key))
}
func (c *Cache) deallocate(node *Entry) {
	c.nBytes -= int64(node.value.Len()) + int64(len(node.key))
}

func (c *Cache) Get(key string) (Value, bool) {
	if p, ok := c.cache[key]; ok {
		c.ll.MoveToFront(p)
		kv := p.Value.(*Entry)
		return kv.value, true

	}
	return nil, false
}

func (c *Cache) RemoveOldest() {

	oldest := c.ll.Back()
	if oldest != nil {
		kv := oldest.Value.(*Entry)
		c.ll.Remove(oldest)
		c.deallocate(kv)
		// c.nBytes -= int64(len(kv.key)) + int64(kv.value.Len())
		delete(c.cache, kv.key)

		if c.OnEvicted != nil {
			c.OnEvicted(kv.key, kv.value)
		}
	}
}

func (c *Cache) Add(key string, value Value) {
	if p, ok := c.cache[key]; ok {
		kv := p.Value.(*Entry)
		c.deallocate(kv)
		kv.value = value
		c.allocate(kv)
		c.ll.MoveToFront(p)

	} else {
		ele := &Entry{
			key:   key,
			value: value,
		}
		listEle := c.ll.PushBack(ele)
		c.allocate(ele)
		c.cache[ele.key] = listEle
		if c.nBytes > c.maxBytes {
			c.RemoveOldest()

		}
	}

}

func (c *Cache) Len() int {
	return c.ll.Len()
}
