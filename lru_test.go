package golru

import (
	"fmt"
	"math/rand"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/go-playground/assert"
	"github.com/stretchr/testify/require"
)

func TestStressGet(t *testing.T) {
	var cacheSize uint = 100
	c := NewCache(cacheSize, nil)
	var err error

	for i := uint(0); i < cacheSize; i++ {
		k := strconv.Itoa(int(i))
		c.Add(k, k)
	}
	time.Sleep(10 * time.Millisecond)
	wg := &sync.WaitGroup{}
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			r := rand.New(rand.NewSource(time.Now().UnixNano()))
			for a := 0; a < 1000; a++ {
				k := strconv.Itoa(r.Int() % int(cacheSize))
				if val, ok := c.Get(k); val == "" || !ok {
					err = fmt.Errorf("expected %q but got \"\"", k)
					break
				} else if val != "" && val != k {
					err = fmt.Errorf("expected %q but got %q", k, val)
					break
				}
			}
			wg.Done()
		}()
	}
	wg.Wait()
	require.NoError(t, err)
}

func TestStressSet(t *testing.T) {
	var cacheSize uint = 10
	var c = NewCache(cacheSize, &CacheConfig{GetThresh: 0, DelThresh: 0})
	var err error

	wg := &sync.WaitGroup{}
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			for a := i * 1000; a < i+1000; a++ {
				k := string(a)
				if ok := c.Add(k, k); !ok {
					err = fmt.Errorf("cant set key: %q", k)
					break
				}
			}
			wg.Done()
		}(i)
	}
	wg.Wait()
	require.NoError(t, err)
	// check everything has been evicted
	l := 0
	time.Sleep(5 * time.Millisecond)
	c.mu.Lock()
	for _, m := range c.shards {
		l += len(m.m)
	}
	c.mu.Unlock()
	assert.Equal(t, int(cacheSize), l)
	c.mu.Lock()
	assert.Equal(t, int(cacheSize), c.ll.Len())
	c.mu.Unlock()
}

func TestStressRemove(t *testing.T) {
	var cacheSize uint = 100 * 100
	c := NewCache(cacheSize, nil)
	var err error

	for i := uint(0); i < cacheSize; i++ {
		k := strconv.Itoa(int(i))
		c.Add(k, k)
	}
	assert.Equal(t, cacheSize, uint(c.Len()))
	time.Sleep(10 * time.Millisecond)
	wg := &sync.WaitGroup{}
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(c *Cache, i int) {
			for a := 0; a < 100; a++ {
				k := strconv.Itoa(100*i + a)
				if ok := c.Remove(k); !ok {
					err = fmt.Errorf("Can't remove %q", k)
					break
				}
			}
			wg.Done()
		}(c, i)
	}
	wg.Wait()
	require.NoError(t, err)
}
