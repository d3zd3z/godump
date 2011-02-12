// Index mapping hashes to file offsets.

package pool

import (
	"bytes"
	"encoding/binary"
	"log"
	"os"
)

type ReadIndexer interface {
	Lookup(oid OID) (offset uint32, present bool)
	Len() int
	ForEach(f func(oid OID, offset uint32))
}

type Indexer interface {
	ReadIndexer
	Add(oid OID, offset uint32)
}

type IndexHeader struct {
	magic    [8]byte
	version  uint32
	poolSize uint32
}

// A multi-index is simply a set of indexes that is each searched.  Foreach combines them in order.
type MultiIndex []ReadIndexer

func (mi MultiIndex) Lookup(oid OID) (offset uint32, present bool) {
	for _, ind := range mi {
		offset, present = ind.Lookup(oid)
		if present {
			return
		}
	}
	return
}

func (mi MultiIndex) Len() (length int) {
	for _, ind := range mi {
		length += ind.Len()
	}
	return
}

func (mi MultiIndex) ForEach(f func(oid OID, offset uint32)) {
	panic("TODO")
}

func WriteIndex(idx ReadIndexer, path string, poolSize uint32) (err os.Error) {
	size := idx.Len()

	oids := make([]byte, 20*size)
	offsets := make([]uint32, size)

	base := 0
	ibase := 0
	idx.ForEach(func(oid OID, offset uint32) {
		copy(oids[base:], []byte(oid))
		base += 20
		offsets[ibase] = offset
		ibase++
	})

	tmpPath := path + ".tmp"

	fd, err := os.Open(tmpPath, os.O_WRONLY|os.O_CREAT|os.O_TRUNC, 0644)
	if err != nil {
		return
	}
	defer fd.Close()

	var header IndexHeader
	copy(header.magic[:], "ldumpidx")
	header.version = 2
	header.poolSize = poolSize
	err = binary.Write(fd, binary.LittleEndian, &header)
	if err != nil {
		return
	}

	// The top-level index is the offsets of the ranges for the
	// search.
	var top bytes.Buffer
	offset := 0
	for first := 0; first < 256; first++ {
		// Write the first oid that is larger than the given index.
		for offset < size && byte(first) >= oids[20*offset] {
			offset++
		}
		binary.Write(&top, binary.LittleEndian, uint32(offset))
	}
	_, err = top.WriteTo(fd)
	if err != nil {
		return
	}

	_, err = fd.Write(oids)
	if err != nil {
		return
	}

	// Lastly, write the offset table.
	var otable bytes.Buffer
	binary.Write(&otable, binary.LittleEndian, offsets)
	_, err = otable.WriteTo(fd)
	if err != nil {
		return
	}

	fd.Sync()
	fd.Close()
	err = os.Rename(tmpPath, path)

	return
}

func index_main() {
	// table := NewMemoryIndex()
	// table := newHashMemoryIndex()
	table := NewRamIndex()

	const limit = 1000000
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
			log.Panicf("Missing: %d", i)
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

	err := WriteIndex(table, "test.idx", 0x12345678)
	if err != nil {
		panic(err)
	}
}
