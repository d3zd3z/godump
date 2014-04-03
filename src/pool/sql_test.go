// Test SQL pools.

package pool_test

import (
	"bytes"
	"os"
	"testing"

	// "pdump"
	"pool"
	"tutil"
)

func TestCreate(t *testing.T) {
	tmp, err := makeTempDir()
	if err != nil {
		t.Errorf("Unable to make temp dir: '%s'", err)
	}
	defer os.RemoveAll(tmp)

	base := tmp + "/pool"
	err = pool.CreateSqlPool(base)
	if err != nil {
		t.Errorf("Unable to create pool: '%s'", err)
	}

	fi, err := os.Stat(base + "/data.db")
	if err != nil {
		t.Errorf("Error finding /data.db: '%s'", err)
	}
	if !fi.Mode().IsRegular() {
		t.Errorf("Database is not regular file")
	}
	return
}

type PoolTest struct {
	*tutil.PoolTest

	known []pool.Chunk
}

func NewPoolTest(t *testing.T) (pt *PoolTest) {
	var result PoolTest
	result.PoolTest = tutil.NewPoolTest(t)

	result.known = make([]pool.Chunk, 0)

	return &result
}

func (pt *PoolTest) Clean() {
	pt.PoolTest.Clean()
}

func (pt *PoolTest) Insert(index int) {
	// TODO: These, unfortunately, always compress well enough to
	// keep them small enough to put directly in the database.
	ch := pool.MakeRandomChunk(index)
	err := pt.Pool.Insert(ch)
	if err != nil {
		pt.T.Errorf("Error inserting chunk: '%s'", err)
	}
	pt.known = append(pt.known, ch)
}

func (pt *PoolTest) InsertRandom(size int) {
	// Generate with a block of pseudo-random data.
	buf := make([]byte, size)
	state := uint32(size)

	for i := range buf {
		state = ((state * 1103515245) + 12345) & 0x7fffffff
		buf[i] = byte(state >> 16)
	}
	ch := pool.NewChunk("blob", buf)
	// pdump.Dump(buf)
	err := pt.Pool.Insert(ch)
	if err != nil {
		pt.T.Errorf("Error inserting chunk: '%s'", err)
	}
	pt.known = append(pt.known, ch)
}

func (pt *PoolTest) Check() {
	for _, ch := range pt.known {
		result, err := pt.Pool.Contains(ch.OID())
		if err != nil {
			pt.T.Errorf("Error checking if pool contains blob.")
		}
		if !result {
			pt.T.Errorf("Pool should contain blob.")
		}

		// Make sure we can read the chunk as well.
		ch2, err := pt.Pool.Search(ch.OID())
		if err != nil {
			pt.T.Errorf("Error reading chunk")
		}
		if ch2 == nil {
			pt.T.Errorf("Did not find previously inserted chunk")
		}
		if ch.Kind() != ch2.Kind() || bytes.Compare(ch.Data(), ch2.Data()) != 0 {
			pt.T.Errorf("Chunk did not reread correctly")
		}
	}
}

func (pt *PoolTest) Flush() {
	err := pt.Pool.Flush()
	if err != nil {
		pt.T.Errorf("Error flushing: '%s'", err)
	}
}

func TestBasic(t *testing.T) {
	pt := NewPoolTest(t)
	defer pt.Clean()

	for _, sz := range makeSizes() {
		pt.Insert(sz)
		if sz > 16 {
			pt.InsertRandom(sz)
		}
	}
	pt.Flush()
	pt.Check()
}
