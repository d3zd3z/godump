package store

import (
	"pool"
)

// Directory writer.

type DirWriter struct {
	pool  pool.Pool
	ind   *IndirectWriter
	limit int

	current []byte
}

// Build a writer that writes the contents of a directory.  The limit
// is the maximum bytes in a given directory chunk.
func NewDirWriter(p pool.Pool, limit int) *DirWriter {
	var self DirWriter

	self.pool = p
	self.ind = NewIndirectWriter(p, "dir", limit)
	self.limit = limit
	self.current = make([]byte, 0, limit)
	return &self
}

func (self *DirWriter) Add(name string, child *pool.OID) (err error) {
	myLen := 2 + len(name) + pool.OIDLen

	if len(self.current)+myLen > self.limit {
		err = self.ship()
		if err != nil {
			return err
		}
	}

	pos := len(self.current)
	tmp := self.current[0 : pos+myLen]

	nameLen := len(name)
	tmp[pos] = byte(nameLen >> 8)
	tmp[pos+1] = byte(nameLen)
	pos += 2

	copy(tmp[pos:pos+nameLen], name)
	pos += nameLen

	copy(tmp[pos:pos+pool.OIDLen], child[:])

	self.current = tmp
	return
}

func (self *DirWriter) Finalize() (id *pool.OID, err error) {
	err = self.ship()
	if err != nil {
		return
	}

	return self.ind.Finalize()
}

func (self *DirWriter) ship() (err error) {
	if len(self.current) == 0 {
		return
	}

	ch := pool.NewChunk("dir ", self.current)
	err = self.pool.Insert(ch)
	if err != nil {
		return
	}

	err = self.ind.Add(ch.OID())
	if err != nil {
		return
	}

	self.current = self.current[0:0]
	return
}
