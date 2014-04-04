package store

import (
	"fmt"

	"pool"
)

// Indirect block management.

type IndirectWriter struct {
	pool   pool.Pool
	prefix string
	limit  int

	tree []level
}

type level []byte

// Build a tracker for indirect blocks.  The 'prefix' should be a 3
// character prefix used to generate the kinds for the various levels.
// The 'limit' is the byte limit for the indirect blocks, before
// another level is used.
func NewIndirectWriter(p pool.Pool, prefix string, limit int) *IndirectWriter {
	var self IndirectWriter

	self.pool = p
	self.prefix = prefix
	self.limit = (limit / pool.OIDLen) * pool.OIDLen

	self.tree = make([]level, 0)
	return &self
}

// Record a given OID into the indirect.  This will be added to an
// level 0 indirect block.
func (self *IndirectWriter) Add(oid *pool.OID) (err error) {
	return self.push(oid, 0)
}

func (self *IndirectWriter) Finalize() (oid *pool.OID, err error) {
	if len(self.tree) == 0 {
		ch := pool.NewChunk("null", []byte{})
		err = self.pool.Insert(ch)
		if err != nil {
			return
		}

		oid = ch.OID()
		return
	}

	// Flush out all of the levels.  It is important to check the
	// len each time, since the tree can grow as it is purged.
	for i := 0; i < len(self.tree); i++ {
		err = self.room(i, true)
		if err != nil {
			return
		}
	}

	var result pool.OID
	copy(result[:], self.tree[len(self.tree)-1][0:pool.OIDLen])
	oid = &result
	return
}

func (self *IndirectWriter) push(oid *pool.OID, level int) (err error) {
	err = self.room(level, false)
	if err != nil {
		return
	}

	self.tree[level].Add(oid)
	return
}

// Ensure there is room to add a node at the given level.  If 'purge'
// is set, flush down to only a single node.
func (self *IndirectWriter) room(level int, purge bool) (err error) {
	// If we don't have any nodes at this level, we need a new
	// level.
	if level >= len(self.tree) {
		self.tree = append(self.tree, newLevel(self.limit))
	}

	llimit := self.limit
	if purge {
		if level == len(self.tree)-1 {
			llimit = 2 * pool.OIDLen
		} else {
			llimit = 1 * pool.OIDLen
		}
	}

	if len(self.tree[level]) >= llimit {
		ch := pool.NewChunk(self.kindName(level), self.tree[level])
		// log.Printf("Writing indirect: level=%d (%s)", level, ch.OID().String())
		// pdump.Dump(self.tree[level])
		err = self.pool.Insert(ch)
		if err != nil {
			return
		}

		err = self.push(ch.OID(), level+1)
		if err != nil {
			return
		}

		self.tree[level] = self.tree[level][0:0]
	}
	return
}

func newLevel(limit int) level {
	return make([]byte, 0, limit)
}

func (self *level) Add(oid *pool.OID) {
	pos := len(*self)
	*self = (*self)[0 : pos+pool.OIDLen]
	copy((*self)[pos:pos+pool.OIDLen], oid[:])
}

func (self *IndirectWriter) kindName(level int) string {
	return fmt.Sprintf("%s%d", self.prefix, level)
}
