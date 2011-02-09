// File index.

package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
)

// Reads the index file written by MemoryIndex.writeIndex

type FileIndex struct {
	top     []byte
	oids    []byte
	offsets []byte
}

func (fi *FileIndex) GetTop(i int) uint32 {
	tmp := i << 2
	return readLE32(fi.top[tmp : tmp+4])
}

func (fi *FileIndex) GetOID(i int) []byte {
	tmp := i * 20
	return fi.oids[tmp : tmp+20]
}

func (fi *FileIndex) GetOffset(i int) uint32 {
	tmp := i << 2
	return readLE32(fi.offsets[tmp : tmp+4])
}

func (fi *FileIndex) Lookup(oid OID) (offset uint32, present bool) {
	rawOid := []byte(oid)
	low := 0
	topByte := int(rawOid[0])
	if topByte > 0 {
		low = int(fi.GetTop(topByte - 1))
	}
	high := int(fi.GetTop(topByte))

	// Binary search.
	for high >= low {
		mid := low + ((high - low) >> 1)
		v := bytes.Compare(fi.GetOID(mid), rawOid)
		switch {
		case v > 0:
			high = mid - 1
		case v < 0:
			low = mid + 1
		default:
			offset = fi.GetOffset(mid)
			present = true
			return
		}
	}
	return
}

var bigEndian = false

func toLittle(value uint32) uint32 {
	if bigEndian {
		return ((value >> 24) |
			((value >> 16) & 0xFF00) |
			((value & 0xFF00) << 16) |
			((value & 0xFF) << 24))
	}
	return value
}

// TODO: Handling these better?
type IndexError string

func (e IndexError) String() string {
	return "Index error: " + string(e)
}

func readFileIndex(path string, poolSize uint32) (index *FileIndex, err os.Error) {
	fd, err := os.Open(path, os.O_RDONLY, 0)
	if err != nil {
		return
	}
	defer fd.Close()

	rawHeader := make([]byte, 16)
	_, err = io.ReadFull(fd, rawHeader)
	if err != nil {
		return
	}

	if string(rawHeader[0:8]) != "ldumpidx" {
		err = IndexError("Invalid magic header")
		return
	}

	if readLE32(rawHeader[8:12]) != 2 {
		err = IndexError("Unsupported index version")
		return
	}

	if readLE32(rawHeader[12:16]) != poolSize {
		err = IndexError("Index mismatch with pool file size")
		return
	}

	var result FileIndex
	result.top = make([]byte, 1024)
	_, err = io.ReadFull(fd, result.top)
	if err != nil {
		return
	}

	size := int(result.GetTop(255))
	fmt.Printf("Count: %d\n", size)

	result.oids = make([]byte, 20*size)
	_, err = io.ReadFull(fd, result.oids)
	if err != nil {
		return
	}

	result.offsets = make([]byte, 4*size)
	_, err = io.ReadFull(fd, result.offsets)
	if err != nil {
		return
	}

	index = &result
	return
}

func indexFileMain() {
	table, err := readFileIndex("test.idx", 0x12345678)
	if err != nil {
		panic(err)
	}

	const limit = 100000
	for i := 0; i < limit; i++ {
		oid := intHash(i)
		index, present := table.Lookup(oid)
		if !present {
			panic("Missing")
		}
		if int(index) != i {
			panic("Wrong")
		}

		oid[19] ^= 1
		_, present = table.Lookup(oid)
		if present {
			panic("Present")
		}
	}
}

func main() {
	index_main()
	// indexFileMain()
}
