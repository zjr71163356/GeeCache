package consistenthash

import (
	"hash/crc32"
	"sort"
	"strconv"
)

// Hash maps bytes to uint32
type Hash func(data []byte) uint32

// Map constains all hashed keys
type Map struct {
	hash     Hash           // hash函数
	replicas int            //每个真实节点对应的虚拟节点的个数
	keys     []int          //虚拟节点的hash值 需要排序
	hashMap  map[int]string //hashMap 其中key是虚拟节点的hash value表示真实节点
}

// New creates a Map instance
func New(replicas int, fn Hash) *Map {

	newMap := &Map{
		hash:     fn,
		replicas: replicas,
		hashMap:  make(map[int]string),
	}

	if fn == nil {
		newMap.hash = crc32.ChecksumIEEE
	}
	return newMap
}

// Add adds some keys to the hash.
func (m *Map) Add(keys ...string) {

	for _, key := range keys {
		for i := 0; i < m.replicas; i++ {
			hash := int(m.hash([]byte(strconv.Itoa(i) + key)))
			m.hashMap[hash] = key
			m.keys = append(m.keys, hash)
		}
	}

	sort.Ints(m.keys)
}

// Get gets the closest item in the hash to the provided key.
func (m *Map) Get(key string) string {

	if len(m.keys) == 0 {
		return ""
	}

	hash := int(m.hash([]byte(key)))

	idx := sort.Search(len(m.keys), func(i int) bool {
		return m.keys[i] >= hash
	})

	return m.hashMap[m.keys[idx%len(m.keys)]]
}
