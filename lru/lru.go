package lru

import (
    "container/list"
    "fmt"
)

// Cache 是一个采用 LRU (最近最少使用) 策略的缓存结构体。
// 它不是并发安全的。
type Cache struct {
    maxBytes  int64                         // 表示缓存能存储的最大字节数上限
    nBytes    int64                         // 已经存储的字节数
    ll        *list.List                    // 使用标准库的双向链表作为缓存队列
    cache     map[string]*list.Element      // 哈希表，用于存储键到链表节点的映射
    OnEvicted func(key string, value Value) // 某个条目被移除时的回调函数，可以为 nil
}

// Value 是一个接口，用于计算一个值所占用的内存大小。
// 任何希望被存储在 Cache 中的值类型都必须实现此接口。
type Value interface {
    Len() int
}

// Entry 是双向链表中存储的数据类型。
// 它包含键和值，方便在淘汰队尾节点时，能通过键从哈希表中删除映射。
type Entry struct {
    key   string
    value Value
}

// New 创建并返回一个新的 Cache 实例。
//
// 此函数用于初始化一个 LRU 缓存。可以指定缓存的最大容量（字节）和一个可选的回调函数，
// 该函数在条目被淘汰时调用。
//
// 参数:
//   maxBytes: 缓存的最大容量（以字节为单位）。如果为 0，表示不限制容量。
//   OnEvicted: 当一个条目被淘汰时调用的回调函数。可以为 nil。
//
// 返回值:
//   *Cache: 一个指向新创建的 Cache 实例的指针。
func New(maxBytes int64, OnEvicted func(key string, value Value)) *Cache {
    return &Cache{
        maxBytes:  maxBytes,
        ll:        list.New(),
        cache:     make(map[string]*list.Element),
        OnEvicted: OnEvicted,
    }
}

// allocate 增加缓存已用字节数。
//
// 这是一个内部辅助函数，用于在添加新条目或更新现有条目时，
// 将该条目占用的字节数（键和值的长度之和）加到 c.nBytes 上。
//
// 参数:
//   node: 指向要计算空间的 Entry 节点的指针。
func (c *Cache) allocate(node *Entry) {
    c.nBytes += int64(node.value.Len()) + int64(len(node.key))
}

// deallocate 减少缓存已用字节数。
//
// 这是一个内部辅助函数，用于在删除条目或更新现有条目时，
// 将该条目占用的字节数从 c.nBytes 中减去。
//
// 参数:
//   node: 指向要计算空间的 Entry 节点的指针。
func (c *Cache) deallocate(node *Entry) {
    c.nBytes -= int64(node.value.Len()) + int64(len(node.key))
}

// Get 方法根据键从缓存中查找对应的值。
//
// 如果键存在于缓存中，此方法会将对应的条目移动到双向链表的头部（表示最近使用），并返回其值。
//
// 参数:
//   key: 要查找的键。
//
// 返回值:
//   Value: 查找到的值。如果未找到，则为 nil。
//   bool: 如果找到了键，则为 true；否则为 false。
func (c *Cache) Get(key string) (Value, bool) {
    if p, ok := c.cache[key]; ok {
        c.ll.MoveToFront(p)
        kv := p.Value.(*Entry)
        return kv.value, true

    }
    return nil, false
}

// RemoveOldest 淘汰并移除缓存中最久未使用的条目。
//
// 此方法会找到双向链表的尾部元素（即最久未使用的条目），将其从链表和哈希表中删除，
// 并更新已用字节数 c.nBytes。如果设置了 OnEvicted 回调函数，则会调用它。
func (c *Cache) RemoveOldest() {

    oldest := c.ll.Back()
    if oldest != nil {
        kv := oldest.Value.(*Entry)
        c.ll.Remove(oldest)
        c.deallocate(kv)
        delete(c.cache, kv.key)

        if c.OnEvicted != nil {
            c.OnEvicted(kv.key, kv.value)
        }
    }
    fmt.Println(c.ll.Len())
}

// Add 方法向缓存中添加或更新一个键值对。
//
// 如果键已存在，则更新其值，并将该条目移动到链表头部。
// 如果键不存在，则创建一个新条目并将其添加到链表头部。
// 添加或更新后，会检查当前已用字节数是否超过最大限制，如果超过，
// 则会循环调用 RemoveOldest 来淘汰旧条目，直到满足容量要求。
//
// 参数:
//   key: 要添加或更新的键。
//   value: 与键关联的值，该值必须实现 Value 接口。
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
        listEle := c.ll.PushFront(ele)
        c.allocate(ele)
        c.cache[ele.key] = listEle

    }

    for c.maxBytes != 0 && c.nBytes > c.maxBytes {
        c.RemoveOldest()
    }
}

// Len 方法返回缓存中当前的条目数量。
//
// 它返回的是缓存中存储的键值对的数量，而不是已用字节数。
//
// 返回值:
//   int: 缓存中的条目总数。
func (c *Cache) Len() int {
    return c.ll.Len()
}