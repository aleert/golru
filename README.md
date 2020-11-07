Golru [WIP]
===========

Golru is a simple LRU cache implementation suitable for parallel
access.

It's based (loosely) on some of the [ristretto](https://github.com/dgraph-io/ristretto) ideas, however much less sophisticated.

It uses map sharding and batching to improve availability and
reduce number of locks.

Let's get to examples:
```go
	c := NewCache(cacheSize, nil)
	for i := 0; i < 100; i++ {
		k := strconv.Itoa(i)
		ok := c.Add(k, k)
	}
	for i := 0; i < 50; i++ {
		k := strconv.Itoa(i)
		val, ok := c.Get(k)
	}
	for i := 50; i < 100; i++ {
		k := strconv.Itoa(i)
		ok := c.Remove(k)
	}
```

for more sophisticated examples with concurrent access see test files.

You can also run tests with
```sh
go test . -race
```

TODO
----
Add some benchmarks