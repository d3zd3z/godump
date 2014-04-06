package dump

import (
	"pool"
)

// A wrapped pool.  Calls the pool below for all operations, but also
// maintains some statistics useful for a progress meter.
type wrappedPool struct {
	child pool.Pool
	sync  func()

	chunkCount int64
	byteCount  int64
	zbyteCount int64
}

func newWrappedPool(child pool.Pool, sync func()) *wrappedPool {
	return &wrappedPool{child: child, sync: sync}
}

func (self *wrappedPool) Close() error { return self.child.Close() }
func (self *wrappedPool) Flush() error { return self.child.Flush() }
func (self *wrappedPool) Contains(oid *pool.OID) (result bool, err error) {
	return self.child.Contains(oid)
}
func (self *wrappedPool) Search(oid *pool.OID) (chunk pool.Chunk, err error) {
	return self.child.Search(oid)
}
func (self *wrappedPool) Backups() (backups []*pool.OID, err error) { return self.child.Backups() }

// TODO: Check if already present, and count that separately.

func (self *wrappedPool) Insert(chunk pool.Chunk) error {
	result := self.child.Insert(chunk)

	self.chunkCount++
	self.byteCount += int64(chunk.DataLen())
	zdata, present := chunk.ZData()
	if present {
		self.zbyteCount += int64(len(zdata))
	} else {
		self.zbyteCount += int64(chunk.DataLen())
	}

	self.sync()

	return result
}
