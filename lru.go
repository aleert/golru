package golru

import (
	"container/list"
	"sync"
)

// Cache is a thread-safe LRU cache that utilizes
// map sharding to provide parallel acess to elements
// and batching to reduce cache lock time
// it means that it has eventual element priority update and removal
// rather than immediate ones
type Cache struct {
	shards    [128]*shard
	ll        *list.List
	getch     chan *Value
	pushch    chan struct{}
	mu        sync.Mutex
	cap       uint
	getThresh uint
	delThresh uint
}

// CacheConfig controls when cache metadata updates
// values update their priorities when more than GetThresh values were accessed
// and remove when size of cache exceeds its capacity + DelThresh
// to implement immediate updates set both thresholds to 0
type CacheConfig struct {
	GetThresh uint
	DelThresh uint
}

// New cache return Cache with specified config
// default thresholds are cap*0.02 + 2 for getThresh and cap*0.01 + 2 fof delThresh
func NewCache(cp uint, conf *CacheConfig) *Cache {
	var shards [128]*shard
	for i := 0; i < 128; i++ {
		shards[i] = &shard{m: make(map[string]*Value)}
	}

	if conf == nil {
		conf = &CacheConfig{}
	}
	if conf.GetThresh == 0 {
		conf.GetThresh = uint(float64(cp)*0.02) + 2
	}
	if conf.DelThresh == 0 {
		conf.DelThresh = uint(float64(cp)*0.01) + 2
	}
	c := &Cache{
		shards:    shards,
		getch:     make(chan *Value),
		pushch:    make(chan struct{}),
		cap:       cp,
		ll:        list.New(),
		getThresh: conf.GetThresh,
		delThresh: conf.DelThresh,
	}
	c.shedGet()
	c.trim()
	return c
}

func (c *Cache) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.ll.Len()
}

// shedGet updates c.ll when more than getThresh was accessed
func (c *Cache) shedGet() {
	go func(c *Cache) {
		var a []*Value
		for v := range c.getch {
			a = append(a, v)
			if uint(len(a)) >= c.getThresh {
				c.mu.Lock()
				for _, i := range a {
					if i.listVal == nil {
						continue
					}
					c.ll.MoveToFront(i.listVal)
				}
				c.mu.Unlock()
				a = a[:0]
			}
		}
	}(c)
}

// trim removes oldest accessed elements when cache size exceeds capacity by newThresh
func (c *Cache) trim() {
	var listLen int
	go func(c *Cache) {
		for range c.pushch {
			listLen++
			if d := listLen - int(c.cap); d >= int(c.delThresh)-1 {
				c.mu.Lock()
				for i := 0; i < d; i++ {
					val := c.ll.Remove(c.ll.Back()).(*Value)
					val.sh.del(val.k)
				}
				listLen -= d
				c.mu.Unlock()
			}
		}
	}(c)
}

// Get returns cache value, setting ok to false if no value present in cache
func (c *Cache) Get(k string) (val string, ok bool) {
	sh := c.shards[MemHashString(k)%128]
	v, ok := sh.get(k)
	if !ok {
		return "", false
	}
	c.getch <- v
	return v.val, true
}

// Add returns true on success, false if such key already in cache
func (c *Cache) Add(k, v string) bool {
	shard := c.shards[MemHashString(k)%128]
	_, ok := shard.get(k)
	if ok {
		return false
	}
	val := &Value{k: k, val: v, listVal: nil, sh: shard}
	shard.set(k, val)
	c.mu.Lock()
	elem := c.ll.PushFront(val)
	c.mu.Unlock()
	val.setList(elem)
	c.pushch <- struct{}{}
	return true
}

// Remove deletes element with key k from cache and returns ok, false if k not in cache
// Remove blocks cache until value is deleted
func (c *Cache) Remove(k string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	shard := c.shards[MemHashString(k)%128]
	v, ok := shard.get(k)
	if !ok {
		return false
	}
	c.ll.Remove(v.listVal)
	shard.del(v.k)
	return true
}

// shard represends map shard with its own lock
type shard struct {
	m  map[string]*Value
	mu sync.RWMutex
}

func (sh *shard) get(key string) (*Value, bool) {
	sh.mu.Lock()
	defer sh.mu.Unlock()
	v, ok := sh.m[key]
	return v, ok
}

func (sh *shard) set(key string, val *Value) {
	sh.mu.Lock()
	defer sh.mu.Unlock()
	sh.m[key] = val
}

func (sh *shard) del(key string) {
	sh.mu.Lock()
	defer sh.mu.Unlock()
	delete(sh.m, key)
}

// Value is a shard value with additional metadata
type Value struct {
	k       string
	val     string
	listVal *list.Element
	mu      sync.Mutex
	sh      *shard
}

func (v *Value) setList(el *list.Element) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.listVal = el
}
