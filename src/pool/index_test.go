package pool_test

import (
	"bytes"
	"encoding/binary"
	"io"
	"testing"

	"pool"
)

func TestIndexBase(t *testing.T) {
	tmp := NewTempDir(t)
	defer tmp.Clean()
}

type IndexInfo struct {
	offset uint32
	kind   pool.Kind
}

// Benchmark a hash that maps to offset/kind.
func BenchmarkIndexHash(b *testing.B) {
	// index := make(map[pool.OID]IndexInfo)

	// Time, the hash generation.
	for i := 0; i < b.N; i++ {
		_ = pool.IntOID(i)
		_ = pool.StringToKind("blob")
	}
}

func BenchmarkIndexHashAdd(b *testing.B) {
	index := make(map[pool.OID]IndexInfo)

	// Time, the hash generation.
	for i := 0; i < b.N; i++ {
		oid := pool.IntOID(i)
		kind := pool.StringToKind("blob")
		index[*oid] = IndexInfo{offset: uint32(i), kind: kind}
	}
}

func BenchmarkIndexHashLookup(b *testing.B) {
	index := make(map[pool.OID]IndexInfo)

	// Time, the hash generation.
	for i := 0; i < b.N; i++ {
		oid := pool.IntOID(i)
		kind := pool.StringToKind("blob")
		index[*oid] = IndexInfo{offset: uint32(i), kind: kind}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		oid := pool.IntOID(i)
		item, ok := index[*oid]
		if !ok {
			b.Fatal("Key not found")
		}
		if item.offset != uint32(i) {
			b.Fatal("Value mismatch")
		}
	}
}

// Benchmark using encoding/binary vs normal encoding.
func BenchmarkIndexBinary(b *testing.B) {
	var buf bytes.Buffer

	for i := 0; i < b.N; i++ {
		if (i & 65535) == 0 {
			buf.Reset()
		}
		item := uint32(i)
		err := binary.Write(&buf, binary.LittleEndian, &item)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark our version.
func BenchmarkIndexLocalBinary(b *testing.B) {
	var buf bytes.Buffer

	for i := 0; i < b.N; i++ {
		if (i & 65536) == 0 {
			buf.Reset()
		}
		err := writeLE32(&buf, uint32(i))
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Copied from the pool version, but local since it isn't exported,
// and to allow us to remove it based on the benchmark results.
func writeLE32(w io.Writer, item uint32) (err error) {
	var buf [4]byte
	/*
		binary.LittleEndian.PutUint32(buf[:], item)
	*/
	buf[0] = byte(item & 0xFF)
	buf[1] = byte((item >> 8) & 0xFF)
	buf[2] = byte((item >> 16) & 0xFF)
	buf[3] = byte((item >> 24) & 0xFF)
	_, err = w.Write(buf[:])
	return
}
