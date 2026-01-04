package singleflightx

import (
	"sync"
	"sync/atomic"
)

type Group[K comparable, V any] struct {
	mu   sync.Mutex
	m    map[K]*call[V]
	pool sync.Pool
}

type call[V any] struct {
	wg   sync.WaitGroup
	val  V
	err  error
	refs int32 // сколько горутин ждут результат
}

func (g *Group[K, V]) getCall() *call[V] {
	if v := g.pool.Get(); v != nil {
		return v.(*call[V])
	}
	return new(call[V])
}

func (g *Group[K, V]) putCall(c *call[V]) {
	var zeroV V
	c.val = zeroV
	c.err = nil
	c.refs = 0
	c.wg = sync.WaitGroup{}
	g.pool.Put(c)
}

// Do выполняет функцию fn только один раз для заданного ключа.
// Остальные параллельные вызовы вернут тот же результат.
func (g *Group[K, V]) Do(key K, fn func() (V, error)) (val V, err error, shared bool) {
	g.mu.Lock()
	if g.m == nil {
		g.m = make(map[K]*call[V])
	}

	if c, ok := g.m[key]; ok {
		atomic.AddInt32(&c.refs, 1)
		g.mu.Unlock()

		c.wg.Wait()
		val, err = c.val, c.err
		shared = true

		if atomic.AddInt32(&c.refs, -1) == 0 {
			g.putCall(c)
		}
		return
	}

	c := g.getCall()
	c.wg.Add(1)
	c.refs = 1
	g.m[key] = c
	g.mu.Unlock()

	val, err = fn()
	c.val, c.err = val, err
	shared = atomic.LoadInt32(&c.refs) > 1
	c.wg.Done()

	g.mu.Lock()
	delete(g.m, key)
	g.mu.Unlock()

	if atomic.AddInt32(&c.refs, -1) == 0 {
		g.putCall(c)
	}
	return
}
