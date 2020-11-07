package golru

import "unsafe"

type stringStruct struct {
	str unsafe.Pointer
	len int
}

// idea taken from ristretto
// https://github.com/dgraph-io/ristretto/blob/master/z/rtutil.go#L42-L44
// for further details see https://dgraph.io/blog/post/introducing-ristretto-high-perf-go-cache
//go:noescape
//go:linkname memhash runtime.memhash
func memhash(p unsafe.Pointer, h, s uintptr) uintptr

// MemHashString provides fast hashing for strings
func MemHashString(str string) uint64 {
	ss := (*stringStruct)(unsafe.Pointer(&str))
	return uint64(memhash(ss.str, 0, uintptr(ss.len)))
}
