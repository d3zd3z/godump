// Just store the RAM mapping using the OID.

package pool

/*
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

// Call the function for each node in the index, visiting them in
// lexicographical order.
func (r *RamIndex) ForEach(f func(oid OID, offset uint32)) {
	oids := r.oids.Bytes()
	for base := 0; base < 256; base++ {
		index := r.index[base]
		for _, num := range index {
			tmp := num * 20
			f(OID(oids[tmp:tmp+20]), r.offsets[num])
		}
	}
}

func (r *RamIndex) Len() int {
	return len(r.offsets)
}

func NewRamIndex() Indexer {
	var ri RamIndex
	return &ri
}
*/
