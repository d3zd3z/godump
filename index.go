// Index mapping hashes to file offsets.

package main

import (
	"bytes"
	"crypto/sha1"
	"log"
	"io"
	"os"
	"strconv"
)

type Indexer interface {
	Lookup(oid OID) (offset uint32, present bool)
	Add(oid OID, offset uint32)
}

type QueryIndexer interface {
	ForEach(f func(oid OID, offset uint32))
	Len() int
}

type FullIndexer interface {
	Indexer
	QueryIndexer
}

func writeLE32(buf *bytes.Buffer, item uint32) {
	buf.WriteByte(byte(item))
	buf.WriteByte(byte(item >> 8))
	buf.WriteByte(byte(item >> 16))
	buf.WriteByte(byte(item >> 24))
}

func WriteIndex(idx QueryIndexer, path string, poolSize uint32) (err os.Error) {
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
		for offset < size && byte(first) >= oids[20*offset] {
			offset++
		}
		writeLE32(&top, uint32(offset))
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
	for _, offset := range offsets {
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
