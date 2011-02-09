// File index.

package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

// Reads the index file written by MemoryIndex.writeIndex

type FileIndex struct {
	top     []uint32
	oids    []byte
	offsets []uint32
}

func (fi *FileIndex) GetOID(i int) []byte {
	tmp := i * 20
	return fi.oids[tmp : tmp+20]
}

func (fi *FileIndex) Lookup(oid OID) (offset uint32, present bool) {
	rawOid := []byte(oid)
	low := 0
	topByte := int(rawOid[0])
	if topByte > 0 {
		low = int(fi.top[topByte-1])
	}
	high := int(fi.top[topByte]) - 1

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
			offset = fi.offsets[mid]
			present = true
			return
		}
	}
	return
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

	var header IndexHeader
	err = binary.Read(fd, binary.LittleEndian, &header)
	if err != nil {
		return
	}

	if string(header.Magic[:]) != "ldumpidx" {
		err = IndexError("Invalid magic header")
		return
	}

	if header.Version != 2 {
		err = IndexError("Unsupported index version")
		return
	}

	if header.PoolSize != poolSize {
		err = IndexError("Index mismatch with pool file size")
		return
	}

	var result FileIndex
	result.top = make([]uint32, 256)
	err = binary.Read(fd, binary.LittleEndian, result.top)
	if err != nil {
		return
	}

	size := int(result.top[255])
	fmt.Printf("Count: %d\n", size)

	result.oids = make([]byte, 20*size)
	_, err = io.ReadFull(fd, result.oids)
	if err != nil {
		return
	}

	result.offsets = make([]uint32, size)
	err = binary.Read(fd, binary.LittleEndian, result.offsets)
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

	const limit = 1000000
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
	// index_main()
	indexFileMain()
}
