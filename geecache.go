package geecache

import (
	"sync"
)

type Group struct {
	name      string
	maincache cache
	getter    Getter
}

type Getter interface {
	Get(key string) ([]byte, error)
}

type GetterFunc func(key string) ([]byte, error)

func (f GetterFunc) Get(key string) ([]byte, error) {
	return f(key)
}

var (
	mu     sync.RWMutex
	groups = make(map[string]*Group)
)

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

func GetGroup(name string) *Group {
	mu.RLock()
	g := groups[name]
	mu.RUnlock()
	return g
}

func (g *Group) Get(key string) (value ByteView, err error) {

	if v, ok := g.maincache.get(key); ok {
		return v, nil
	}
	return g.load(key)

}

func (g *Group) load(key string) (value ByteView, err error) {
	return g.getLocally(key)
}

func (g *Group) getLocally(key string) (value ByteView, err error) {

	v, err := g.getter.Get(key)
	if err != nil {
		return ByteView{}, err
	}

	value = ByteView{b: cloneBytes(v)}
	g.populateCache(key, value)

	return
}
func (g *Group) populateCache(key string, value ByteView) {
	g.maincache.add(key, value)
}
