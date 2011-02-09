// Index mapping hashes to file offsets.

package main

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
)

// In-memory maps from hashes to keys.

// Extract a numbered nibble from an oid.
func getNibble(oid OID, index int) int {
	num := oid[index>>1]
	if (index & 1) == 0 {
		num >>= 4
	}
	return int(num & 0x0f)
}

type Hasher interface {
	Lookup(oid OID, index int) (offset uint32, present bool)
	Add(oid OID, index int, offset uint32) Hasher
	Show(level int)
}

type trieNode [16]Hasher

type trieLeaf struct {
	oid    OID
	offset uint32
}

type trieEmpty int

var emptyCell = new(trieEmpty)

func (node *trieEmpty) Lookup(oid OID, index int) (offset uint32, present bool) { return }
func (node *trieLeaf) Lookup(oid OID, index int) (offset uint32, present bool) {
	if node.oid.Compare(oid) == 0 {
		offset = node.offset
		present = true
	}
	return
}
func (node *trieNode) Lookup(oid OID, index int) (offset uint32, present bool) {
	offset, present = node[getNibble(oid, index)].Lookup(oid, index+1)
	return
}

// The Add method returns the new node, which might be a new type if it has to be split.
func (node *trieEmpty) Add(oid OID, index int, offset uint32) Hasher {
	return &trieLeaf{oid, offset}
}

func (node *trieLeaf) Add(oid OID, index int, offset uint32) Hasher {
	if node.oid.Compare(oid) == 0 {
		panic("Duplicate add")
	}
	var child trieNode
	for i := 0; i < 16; i++ {
		child[i] = emptyCell
	}

	child[getNibble(node.oid, index)] = node
	return child.Add(oid, index, offset)
}

func (node *trieNode) Add(oid OID, index int, offset uint32) Hasher {
	piece := getNibble(oid, index)
	node[piece] = node[piece].Add(oid, index+1, offset)
	return node
}

type MemoryIndex struct {
	box Hasher
}

func NewMemoryIndex() Indexer {
	var mi MemoryIndex
	InitMemoryIndex(&mi)
	return &mi
}

func InitMemoryIndex(mi *MemoryIndex) {
	mi.box = emptyCell
}

func (mi *MemoryIndex) Add(oid OID, offset uint32) {
	mi.box = mi.box.Add(oid, 0, offset)
}

func (mi *MemoryIndex) Lookup(oid OID) (offset uint32, present bool) {
	offset, present = mi.box.Lookup(oid, 0)
	return
}

func (mi *MemoryIndex) Show() {
	mi.box.Show(0)
}

func (node *trieEmpty) Show(level int) {
	fmt.Printf("%*s\u03d5\n", 2*level, "")
}
func (node *trieLeaf) Show(level int) {
	fmt.Printf("%*sleaf: %x -> %d\n", 2*level, "", []byte(node.oid), node.offset)
}
func (node *trieNode) Show(level int) {
	fmt.Printf("%*snode\n", 2*level, "")
	for i := 0; i < 16; i++ {
		node[i].Show(level + 1)
	}
}

type Indexer interface {
	Lookup(oid OID) (offset uint32, present bool)
	Add(oid OID, offset uint32)
}

// As a special hack, store the offsets using a map, with the OID keys
// as strings.  There are lots of copies of the 20-byte OIDs to make
// the strings, but it saves us from having to write the map itself.

// This seems rather inefficient, so we'll try to come up with
// something better, preferrably that keeps the data sorted, knows
// that the data won't have deletes.

type HashMemoryIndex map[string]uint32

func newHashMemoryIndex() HashMemoryIndex {
	return make(map[string]uint32)
}

func (idx HashMemoryIndex) Add(oid OID, offset uint32) {
	idx[string([]byte(oid))] = offset
}

func (idx HashMemoryIndex) Lookup(oid OID) (offset uint32, present bool) {
	offset, present = idx[string([]byte(oid))]
	return
}

type sortMap struct {
	oids    []byte
	offsets []uint32
}

func (items *sortMap) Len() int { return len(items.offsets) }
func (items *sortMap) Less(i, j int) bool {
	ibase := 20 * i
	ia := items.oids[ibase : ibase+20]
	jbase := 20 * j
	ja := items.oids[jbase : jbase+20]
	return bytes.Compare(ia, ja) < 0
}
func (items *sortMap) Swap(i, j int) {
	items.offsets[i], items.offsets[j] = items.offsets[j], items.offsets[i]

	ibase := 20 * i
	ia := items.oids[ibase : ibase+20]
	jbase := 20 * j
	ja := items.oids[jbase : jbase+20]

	tmp := make([]byte, 20)
	copy(tmp, ia)
	copy(ia, ja)
	copy(ja, tmp)
}

func writeLE32(buf *bytes.Buffer, item uint32) {
	buf.WriteByte(byte(item))
	buf.WriteByte(byte(item >> 8))
	buf.WriteByte(byte(item >> 16))
	buf.WriteByte(byte(item >> 24))
}

func (idx HashMemoryIndex) writeIndex(path string, poolSize uint32) (err os.Error) {
	var items sortMap
	size := len(idx)

	items.oids = make([]byte, 20*size)
	items.offsets = make([]uint32, size)

	base := 0
	ibase := 0
	for oid, offset := range idx {
		copy(items.oids[base:], oid)
		base += 20
		items.offsets[ibase] = offset
		ibase++
	}

	sort.Sort(&items)

	// TODO: Write to temp, and rename.
	fd, err := os.Open(path, os.O_WRONLY|os.O_CREAT|os.O_TRUNC, 0644)
	if err != nil {
		return
	}
	defer fd.Close()

	var header bytes.Buffer
	header.WriteString("ldumpidx")
	writeLE32(&header, 2)
	writeLE32(&header, poolSize)
	_, err = header.WriteTo(fd)
	if err != nil {
		return
	}

	// The top-level index is the offsets of the ranges for the
	// search.
	var top bytes.Buffer
	offset := 0
	for first := 0; first < 256; first++ {
		// Write the first oid that is larger than the given index.
		for offset < size && byte(first) >= items.oids[20*offset] {
			offset++
		}
		writeLE32(&top, uint32(offset))
	}
	_, err = top.WriteTo(fd)
	if err != nil {
		return
	}

	_, err = fd.Write(items.oids)
	if err != nil {
		return
	}

	// Lastly, write the offset table.
	var otable bytes.Buffer
	for _, offset := range items.offsets {
		writeLE32(&otable, uint32(offset))
	}
	_, err = otable.WriteTo(fd)
	if err != nil {
		return
	}

	return
}

type OID []byte

const hexDigits = "0123456789abcdef"

func (item OID) String() string {
	var result [40]byte
	for i, ch := range ([]byte)(item) {
		result[2*i] = hexDigits[ch>>4]
		result[2*i+1] = hexDigits[ch&0x0f]
	}

	return string(result[:])
}

func (me OID) Compare(other OID) int {
	return bytes.Compare([]byte(me), []byte(other))
}

func intHash(index int) (oid OID) {
	hash := sha1.New()
	io.WriteString(hash, "blob")
	io.WriteString(hash, strconv.Itoa(index))
	return OID(hash.Sum())
}

func index_main() {
	table := NewMemoryIndex()
	// table := newHashMemoryIndex()

	limit := 1000000
	for i := 0; i < limit; i++ {
		oid := intHash(i)
		// fmt.Printf("Add %x -> %d\n", []byte(oid), i)
		table.Add(oid, uint32(i))
	}

	// table.Show()

	// Test that we can find them all.
	for i := 0; i < limit; i++ {
		oid := intHash(i)
		index, present := table.Lookup(oid)
		if !present {
			panic("Missing")
		}
		if index != uint32(i) {
			panic("Wrong")
		}
		oid[19] ^= 1
		_, present = table.Lookup(oid)
		if present {
			panic("Present")
		}
	}

	/*
		err := table.writeIndex("test.idx", 0x12345678)
		if err != nil {
			panic(err)
		}
	*/
}
