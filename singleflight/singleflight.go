package singleflight

import "sync"

// call is an in-flight or completed Do call
// call表示正在进行或已结束的请求
// call中定义的sync.WaitGroup，用于使得同一时刻最多只有第一个的fn()被执行，后续的请求使用wg.Wait()等候第一个的执行wg.Done()
// 起到了防止重复执行fn()的作用
type call struct {
	wg  sync.WaitGroup
	val any   //请求返回的值
	err error //请求返回的错误
}

// Group represents a class of work and forms a namespace in which
// units of work can be executed with duplicate suppression.
type Group struct {
	mu sync.Mutex
	m  map[string]*call //一个key只对应一个请求
}

func (g *Group) Do(key string, fn func() (any, error)) (any, error) {

	g.mu.Lock()
	if g.m == nil {
		g.m = make(map[string]*call)
	}

	if c, ok := g.m[key]; ok {
		g.mu.Unlock()
		c.wg.Wait()
		return c.val, c.err
	}

	c := new(call)
	g.m[key] = c
	g.mu.Unlock()

	c.wg.Add(1)
	c.val, c.err = fn()
	c.wg.Done()

	g.mu.Lock()
	delete(g.m, key)
	g.mu.Unlock()

	return c.val, c.err

}
