package store_test

import (
	"errors"
	"fmt"
	"testing"

	"pool"
	"store"
	"tutil"
)

type indTest struct {
	*tutil.PoolTest
	store.EmptyVisitor
	store.PathTrackerImpl

	last int
}

// Verify that the indirect blocks work.  This doesn't verify that the
// indirect blocks are structured correctly, only that they traverse
// correctly.
func TestIndirect(t *testing.T) {
	var self indTest
	self.PoolTest = tutil.NewPoolTest(t)
	defer self.Clean()

	ind := store.NewIndirectWriter(self.Pool, "ind", pool.OIDLen*5)

	for i := 1; i < 500; i++ {
		id := self.makeData(i)
		// log.Printf("Ind %d: %s", i, id.String())
		err := ind.Add(id)
		if err != nil {
			t.Errorf("Error adding indirect block: %q", err)
		}
	}
	oid, err := ind.Finalize()
	if err != nil {
		t.Errorf("Error finializing: %q", err)
	}
	err = store.Walk(self.Pool, oid, &self)
	if err != nil {
		t.Errorf("Error walking tree: %q", err)
	}
}

func (self *indTest) Blob(chunk pool.Chunk) (err error) {
	self.last++
	id := pool.IntOID(self.last)
	if chunk.OID().Compare(id) != 0 {
		err = errors.New("Invalid blob in walk")
		return
	}
	return
}

func (self *indTest) makeData(index int) *pool.OID {
	buf := []byte(fmt.Sprintf("%d", index))
	ch := pool.NewChunk("blob", buf)
	err := self.Pool.Insert(ch)
	if err != nil {
		self.T.Errorf("Error writing chunk: %q", err)
	}
	return ch.OID()
}
