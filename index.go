// Index mapping hashes to file offsets.

package main

import (
	"bytes"
	"crypto/sha1"
	"io"
	"os"
	"sort"
	"strconv"
)

// As a special hack, store the offsets using a map, with the OID keys
// as strings.  There are lots of copies of the 20-byte OIDs to make
// the strings, but it saves us from having to write the map itself.

type MemoryIndex map[string]int

func newMemoryIndex() MemoryIndex {
	return make(map[string]int)
}

func (idx MemoryIndex) Add(oid OID, offset int) {
	idx[string([]byte(oid))] = offset
}

func (idx MemoryIndex) Lookup(oid OID) (offset int, present bool) {
	offset, present = idx[string([]byte(oid))]
	return
}

type sortMap struct {
	oids    []byte
	offsets []int
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

func (idx MemoryIndex) writeIndex(path string, poolSize uint32) (err os.Error) {
	var items sortMap
	size := len(idx)

	items.oids = make([]byte, 20*size)
	items.offsets = make([]int, size)

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

func intHash(index int) (oid OID) {
	hash := sha1.New()
	io.WriteString(hash, "blob")
	io.WriteString(hash, strconv.Itoa(index))
	return OID(hash.Sum())
}

func main() {
	table := newMemoryIndex()

	limit := 100000
	for i := 1; i < limit; i++ {
		oid := intHash(i)
		table.Add(oid, i)
	}
	// Test that we can find them all.
	for i := 1; i < limit; i++ {
		oid := intHash(i)
		index, present := table.Lookup(oid)
		if !present {
			panic("Missing")
		}
		if index != i {
			panic("Wrong")
		}
		oid[19] ^= 1
		_, present = table.Lookup(oid)
		if present {
			panic("Present")
		}
	}

	err := table.writeIndex("test.idx", 0x12345678)
	if err != nil {
		panic(err)
	}
}
