package store_test

import (
	"fmt"
	"testing"

	"pool"
	"store"
	"tutil"
)

type dirTest struct {
	*tutil.PoolTest
	store.EmptyVisitor
	store.PathTrackerImpl

	last int
}

// Verify that we generate dir blocks correctly.
func TestDirWriter(t *testing.T) {
	var self dirTest
	self.PoolTest = tutil.NewPoolTest(t)
	defer self.Clean()
	self.InitPath()

	dirw := store.NewDirWriter(self.Pool, 1024)

	for i := 1; i < 500; i++ {
		name, id := self.makeData(i)
		err := dirw.Add(name, id)
		if err != nil {
			t.Errorf("Error adding dir block: %q", err)
		}
	}
	oid, err := dirw.Finalize()
	if err != nil {
		t.Errorf("Error finalizing: %q", err)
	}
	err = store.Walk(self.Pool, oid, &self)
	if err != nil {
		t.Errorf("Error walking tree: %q", err)
	}
}

func (self *dirTest) Blob(chunk pool.Chunk) (err error) {
	self.last++
	id := pool.IntOID(self.last)
	name := self.makeName(self.last)
	if name != self.Path("") {
		self.T.Errorf("Name mismatch: %q got %q", name, self.Path(""))
	}
	if chunk.OID().Compare(id) != 0 {
		self.T.Errorf("Wrong child block")
	}
	return
}

func (self *dirTest) makeData(i int) (name string, id *pool.OID) {
	buf := []byte(fmt.Sprintf("%d", i))
	ch := pool.NewChunk("blob", buf)
	err := self.Pool.Insert(ch)
	if err != nil {
		self.T.Errorf("Error writing chunk: %q", err)
	}
	id = ch.OID()
	name = self.makeName(i)
	return
}

func (self *dirTest) makeName(i int) (name string) {
	return pool.MakeRandomSentence(i, i)
}
