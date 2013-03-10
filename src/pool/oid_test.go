// Test the OID code.

package pool_test

import "fmt"
import "pool"
import "testing"

func TestOidBasic(t *testing.T) {
	hash, err := pool.ParseOID("0000000000000000000000000000000000000000")
	if err != nil {
		t.Errorf("Error parsing oid: %v", err)
	}
	check(t, hash, "0000000000000000000000000000000000000000")

	hash, err = pool.ParseOID("ffffffffffffffffffffffffffffffffffffffff")
	if err != nil {
		t.Error("Error parsing oid")
	}
	check(t, hash, "ffffffffffffffffffffffffffffffffffffffff")

	hash, err = pool.ParseOID("42")
	if err == nil {
		t.Error("Shouldn't be able to parse oid")
	}

	hash, err = pool.ParseOID("000000000000000000000000000000000000000g")
	if err == nil {
		t.Error("Shouldn't be able to parse oid")
	}
}

func TestOidHashes(t *testing.T) {
	check(t, pool.BlobOID("blob", []byte("This is a sample message")),
		"fc46bae8992795a17f286ddc1743a00a0cd33c0a")
	check(t, pool.BlobOID("blob", []byte("")),
		"0fd0bcfb44f83e7d5ac7a8922578276b9af48746")
	check(t, pool.IntOID(5124), "f2a4cd9a77813d7c49c223739eb8ab5b9bbe71e9")
}

func check(t *testing.T, hash *pool.OID, expected string) {
	if hash.String() != expected {
		fmt.Printf("Got:      '%v'\nexpected: '%v'\n",
			hash.String(), expected)
		t.Error("Mismatch in hash")
	}
}

// Benchmarking the generation of OIDs.
func BenchmarkOIDGen(b *testing.B) {
	for i := 0; i < b.N; i++ {
		pool.IntOID(i)
	}
}

// Storing the hashes into a map.
func BenchmarkOIDStore(b *testing.B) {
	m := make(map[pool.OID]int)
	for i := 0; i < b.N; i++ {
		m[*pool.IntOID(i)] = i
	}
}

// Looking up things in the map.
func BenchmarkOIDFetch(b *testing.B) {
	b.StopTimer()
	m := make(map[pool.OID]int)
	for i := 0; i < b.N; i++ {
		m[*pool.IntOID(i)] = i
	}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		if m[*pool.IntOID(i)] != i {
			b.Error("Problem")
		}
	}
}
