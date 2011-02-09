// Specialized mapping between sha1 hashes in integer offsets.  This
// is actually slightly slower than hash tables (depending on the key
// size), but uses significantly less memory.

package main

import (
	"bytes"
	"sort"
)

type RamIndex struct {
	oids    bytes.Buffer
	offsets []uint32
	index   [256][]int
}

func (r *RamIndex) Add(oid OID, offset uint32) {
	r.oids.Write([]byte(oid))
	r.offsets = append(r.offsets, offset)
	r.updateIndex(oid)
}

func (r *RamIndex) updateIndex(oid OID) {
	index, pos, found := r.findKey(oid)

	if found {
		panic("Duplicate hash added")
	}

	index = append(index, 0)
	r.index[oid[0]] = index
	copy(index[pos+1:], index[pos:len(index)-1])
	index[pos] = len(r.offsets) - 1
}

// Do a binary search for the oid in the appropriate index.  Pos will
// be either the position of the oid, or where it should be inserted.
// 'found' will indicate if it is a match.
func (r *RamIndex) findKey(oid OID) (index []int, pos int, found bool) {
	index = r.index[oid[0]]
	oids := r.oids.Bytes()

	pos = sort.Search(len(index), func(i int) bool {
		base := 20 * index[i]
		return bytes.Compare([]byte(oid), oids[base:base+20]) <= 0
	})

	if pos < len(index) {
		tmp := 20 * index[pos]
		if bytes.Compare([]byte(oid), oids[tmp:tmp+20]) == 0 {
			found = true
		}
	}
	return
}

func (r *RamIndex) Lookup(oid OID) (offset uint32, present bool) {
	index, pos, present := r.findKey(oid)
	if present {
		offset = r.offsets[index[pos]]
	}
	return
}

func NewRamIndex() Indexer {
	var ri RamIndex
	return &ri
}
