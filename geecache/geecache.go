package geecache

import (
	"log"
	"sync"
)

// Getter 接口定义了从数据源获取数据的回调。
// 当缓存未命中时，会调用此接口的方法来获取源数据。
type Getter interface {
	Get(key string) ([]byte, error)
}

// GetterFunc 类型是一个函数类型，它实现了 Getter 接口。
// 这是一种适配器模式，使得普通函数可以作为 Getter 使用，而无需定义一个新结构体。
type GetterFunc func(key string) ([]byte, error)

// Get 实现了 Getter 接口的 Get 方法。
//
// 它允许 GetterFunc 类型的函数满足 Getter 接口，
// 内部只是简单地调用函数自身。
//
// 参数:
//
//	key: 要获取数据的键。
//
// 返回值:
//
//	[]byte: 获取到的数据。
//	error: 如果获取过程中发生错误，则返回错误信息。
func (f GetterFunc) Get(key string) ([]byte, error) {
	return f(key)
}

// Group 是 GeeCache 的核心数据结构，负责与用户的交互，并且控制缓存值存储和获取的流程。
// 一个 Group 可以被看作一个独立的缓存命名空间。
type Group struct {
	name      string
	maincache cache
	getter    Getter
	peers     PeerPicker
}

var (
	mu     sync.RWMutex
	groups = make(map[string]*Group)
)

// NewGroup 创建并注册一个新的 Group 实例。
//
// 此函数会检查提供的 getter 是否为 nil，如果是则会引发 panic。
// 它会以并发安全的方式将新创建的 group 注册到全局的 groups 映射中。
// 如果已存在同名 group，则会覆盖。
//
// 参数:
//
//	name: group 的唯一名称。
//	cacheBytes: 分配给该 group 的缓存最大容量（字节）。
//	getter: 当缓存未命中时，用于加载源数据的回调函数。
//
// 返回值:
//
//	*Group: 一个指向新创建的 Group 实例的指针。
func NewGroup(name string, cacheBytes int64, getter Getter) *Group {

	if getter == nil {
		panic(`geecache: nil Getter`)
	}
	mu.Lock()
	defer mu.Unlock()

	newGroup := &Group{
		name:   name,
		getter: getter,
		maincache: cache{
			cacheBytes: cacheBytes,
		},
	}

	groups[name] = newGroup

	return newGroup
}

// GetGroup 根据名称从全局 `groups` 映射中获取一个 Group。
//
// 这是一个并发安全的只读操作。
//
// 参数:
//
//	name: 要获取的 group 的名称。
//
// 返回值:
//
//	*Group: 查找到的 Group 指针。如果未找到，则返回 nil。
func GetGroup(name string) *Group {
	mu.RLock()
	defer mu.RUnlock()
	g := groups[name]
	return g
}

// Get 是 Group 的主要方法，用于根据 key 获取值。
//
// 它首先会尝试从主缓存 (maincache) 中获取值。如果缓存中不存在，
// 它将调用 load 方法来从数据源加载数据。
//
// 参数:
//
//	key: 要获取值的键。
//
// 返回值:
//
//	value: 查找到的值，类型为 ByteView。
//	err: 如果在获取过程中发生错误，则返回错误信息。
func (g *Group) Get(key string) (value ByteView, err error) {

	if v, ok := g.maincache.get(key); ok {
		log.Println("[GeeCache] hit")
		return v, nil
	}
	return g.load(key)

}

// load 在缓存未命中时加载数据。
//
// 目前它只调用 getLocally 从本地获取数据。
// （在后续步骤中，这里将被扩展为可以从远程节点获取数据）。
//
// 参数:
//
//	key: 要加载数据的键。
//
// 返回值:
//
//	value: 加载到的值。
//	err: 如果加载过程中发生错误，则返回错误信息。
func (g *Group) load(key string) (value ByteView, err error) {
	if g.peers != nil {
		if peerGetter, ok := g.peers.PickPeer(key); ok {
			if v, err := g.getFromPeer(peerGetter, key); err == nil {
				return v, nil

			}
			log.Println("[GeeCache] Failed to get from peer", err)
		}
		log.Println("[GeeCache] Failed to get from peer, will try locally")
	}

	return g.getLocally(key)
}

func (g *Group) getFromPeer(peer PeerGetter, key string) (ByteView, error) {
	bytes, err := peer.Get(g.name, key)
	if err != nil {
		return ByteView{}, err
	}
	return ByteView{b: cloneBytes(bytes)}, err

}

// getLocally 调用用户提供的 getter 来获取源数据，并将其添加到缓存中。
//
// 它会调用 group 初始化时注册的 getter 函数来获取源数据。
// 获取成功后，会将数据封装成 ByteView 并调用 populateCache 添加到缓存中。
//
// 参数:
//
//	key: 要获取数据的键。
//
// 返回值:
//
//	value: 从数据源获取到的值。
//	err: 如果 getter 返回错误，则透传该错误。
func (g *Group) getLocally(key string) (value ByteView, err error) {

	bytes, err := g.getter.Get(key)
	if err != nil {
		return ByteView{}, err
	}

	value = ByteView{b: cloneBytes(bytes)}
	g.populateCache(key, value)

	return value, nil
}

// populateCache 将一个键值对添加到 Group 的缓存中。
//
// 这是一个内部方法，用于将加载到的数据存入 maincache。
//
// 参数:
//
//	key: 要添加的键。
//	value: 要添加的值。
func (g *Group) populateCache(key string, value ByteView) {
	g.maincache.add(key, value)
}
