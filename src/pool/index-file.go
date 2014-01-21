// File index.

package pool

import (
	"bytes"
	"fmt"
	"os"
	"sort"
)

const indexMagic = "ldumpidx"
const indexVersion = 4

// The RAM index maps OIDs to the offset and kind of the node.
type IndexValue struct {
	Offset uint32
	Kind   string
}

// A lookup index allows things to be looked up in it.
type IndexReader interface {
	Lookup(key *OID) (value IndexValue, present bool)
}

// The builder, in addition to lookup, can extract all of the keys.
type IndexBuilder interface {
	IndexReader
	GetKeys() (keys []OID)
}

type RamIndex map[OID]IndexValue

func (ri RamIndex) Lookup(key *OID) (value IndexValue, present bool) {
	value, present = ri[*key]
	return
}

// WriteIndex exports the given index to a file.  The index file also
// records a size.  This can be used when reading the index back to
// make sure that it completely covers a given pool file.
func WriteIndex(path string, ri IndexBuilder, size uint32) (err error) {

	tmpName := path + ".tmp"
	fd, err := os.Create(tmpName)
	success := false
	defer func() {
		fd.Close()
		if success {
			os.Rename(tmpName, path)
		} else {
			os.Remove(tmpName)
		}
	}()
	if err != nil {
		return
	}

	// Construct and write out the various parts.
	err = writeHeader(fd, size)
	if err != nil {
		return
	}

	keys := ri.GetKeys()
	sort.Sort(OIDSlice(keys))

	err = writeTop(fd, keys)
	if err != nil {
		return
	}

	err = writeHashes(fd, keys)
	if err != nil {
		return
	}

	err = writeOffsets(fd, keys, ri)
	if err != nil {
		return
	}

	err = writeKinds(fd, keys, ri)
	if err != nil {
		return
	}

	success = true
	return
}

func writeHeader(fd *os.File, size uint32) (err error) {
	var raw bytes.Buffer

	// Ignoring the errors is intentional.  bytes.Buffer never
	// returns errors.
	raw.WriteString(indexMagic)
	writeLE32(&raw, indexVersion)
	writeLE32(&raw, size)

	_, err = fd.Write(raw.Bytes())

	return
}

func writeTop(fd *os.File, keys []OID) (err error) {
	var top bytes.Buffer

	offset := 0
	for first := 0; first < 256; first++ {
		for offset < len(keys) && int(keys[offset][0]) <= first {
			offset++
		}
		writeLE32(&top, uint32(offset))
	}

	_, err = fd.Write(top.Bytes())
	return
}

func writeHashes(fd *os.File, keys []OID) (err error) {
	writer := func(buf *bytes.Buffer, i int) {
		buf.Write(keys[i][:])
	}
	err = writeBufferedThings(fd, 1024, len(keys), writer)
	return
}

func writeOffsets(fd *os.File, keys []OID, ri IndexBuilder) (err error) {
	writer := func(buf *bytes.Buffer, i int) {
		v, _ := ri.Lookup(&keys[i])
		writeLE32(buf, v.Offset)
	}
	err = writeBufferedThings(fd, 4096, len(keys), writer)
	return
}

func writeKinds(fd *os.File, keys []OID, ri IndexBuilder) (err error) {
	allKinds := make(map[string]bool)

	for i := range keys {
		v, _ := ri.Lookup(&keys[i])
		allKinds[v.Kind] = true
	}

	kindTable := make([]string, 0, len(allKinds))
	for k := range allKinds {
		kindTable = append(kindTable, k)
	}

	sort.Strings(kindTable)

	kindMap := make(map[string]uint32)
	for i, k := range kindTable {
		kindMap[k] = uint32(i)
	}

	var buf bytes.Buffer
	writeLE32(&buf, uint32(len(kindMap)))
	for _, k := range kindTable {
		buf.WriteString(k)
	}

	_, err = fd.Write(buf.Bytes())
	if err != nil {
		return
	}

	writer := func(buf *bytes.Buffer, i int) {
		v, _ := ri.Lookup(&keys[i])
		buf.WriteByte(byte(kindMap[v.Kind]))
	}
	err = writeBufferedThings(fd, 4096, len(keys), writer)

	return
}

// Write buffered 'things', in groups of 'count'.
func writeBufferedThings(fd *os.File, grouping, count int, add func(buf *bytes.Buffer, index int)) (err error) {
	var buf bytes.Buffer

	subcount := 0
	for i := 0; i < count; i++ {
		add(&buf, i)
		subcount++

		if subcount == grouping {
			_, err = fd.Write(buf.Bytes())
			if err != nil {
				return
			}

			buf.Reset()
			subcount = 0
		}
	}

	if subcount > 0 {
		_, err = fd.Write(buf.Bytes())
		if err != nil {
			return
		}
	}

	return
}

func (ri RamIndex) GetKeys() (keys []OID) {
	keys = make([]OID, 0, len(ri))

	for k, _ := range ri {
		keys = append(keys, k)
	}

	return
}

func IndexMain() {
	ri := make(RamIndex)
	for i := 1; i < 1000000; i++ {
		ri[*IntOID(i)] = IndexValue{uint32(i), makeKind(i)}
	}
	err := WriteIndex("test.idx", ri, uint32(len(ri)))
	if err != nil {
		panic("Error")
	}
	fmt.Printf("Done\n")
}

type OIDSlice []OID

func (p OIDSlice) Len() int           { return len(p) }
func (p OIDSlice) Less(i, j int) bool { return p[i].Compare(&p[j]) < 0 }
func (p OIDSlice) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

// Testing.
var kinds = []string{"blob", "dir0", "dir1", "null", "dir2", "back"}

func makeKind(i int) string {
	return kinds[i%len(kinds)]
}

/*
import (
	"bytes"
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
	high := int(fi.GetTop(topByte)) - 1

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

func (fi *FileIndex) Len() int {
	return int(fi.GetTop(255))
}

func (fi *FileIndex) ForEach(f func(oid OID, offset uint32)) {
	panic("Not implemented")
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
*/
