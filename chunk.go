// Backup chunks.

package main

import (
	"fmt"
	"bytes"
	"compress/zlib"
	"io"
	"os"
	"sync"
)

type Chunk interface {
	Kind() []byte
	OID() []byte
	Data() []byte
	DataLen() uint32
	ZData() (zdata []byte, present bool)
}

var ChunkError = os.NewError("Error reading chunk")
var chunkMagic = ([]byte)("adump-pool-v1.1\n")

func readLE32(piece []byte) uint32 {
	return uint32(piece[0]) |
		(uint32(piece[1]) << 8) |
		(uint32(piece[2]) << 16) |
		(uint32(piece[3]) << 24)
}

type sharedChunk struct {
	kind []byte
	oid  []byte
}

func (ch *sharedChunk) Kind() []byte { return ch.kind }
func (ch *sharedChunk) OID() []byte  { return ch.oid }

type compressedChunk struct {
	sharedChunk
	dataLen uint32
	zdata   []byte
	getData func() []byte
}

func (ch *compressedChunk) Data() []byte          { return ch.getData() }
func (ch *compressedChunk) DataLen() uint32       { return ch.dataLen }
func (ch *compressedChunk) ZData() ([]byte, bool) { return ch.zdata, true }

// Construct a new chunk out of compressed data.
func NewCompressedChunk(kind, oid []byte, dataLen uint32, zdata []byte) Chunk {
	var data []byte

	getData := func() {
		var dataBuf bytes.Buffer
		r, _ := zlib.NewReader(bytes.NewBuffer(zdata))
		io.Copy(&dataBuf, r)
		r.Close()
		data = dataBuf.Bytes()
		if len(data) != int(dataLen) {
			panic("Wrong length on decompress")
		}
	}
	var once sync.Once
	return &compressedChunk{
		sharedChunk{kind, oid},
		dataLen, zdata,
		func() []byte { once.Do(getData); return data }}
}

type dataChunk struct {
	sharedChunk
	data     []byte
	getZData func() (zdata []byte, present bool)
}

func (ch *dataChunk) Data() []byte          { return ch.data }
func (ch *dataChunk) DataLen() uint32       { return uint32(len(ch.data)) }
func (ch *dataChunk) ZData() ([]byte, bool) { return ch.getZData() }

func NewDataChunk(kind, oid []byte, data []byte) Chunk {
	var zdata []byte
	present := false

	getZData := func() {
		var zbuf bytes.Buffer
		w, _ := zlib.NewWriter(&zbuf)
		io.Copy(w, bytes.NewBuffer(data))
		w.Close()
		tmp := zbuf.Bytes()
		if len(tmp) < len(data) {
			zdata = tmp
			present = true
		}
	}
	var once sync.Once
	return &dataChunk{
		sharedChunk{kind, oid},
		data,
		func() ([]byte, bool) {
			once.Do(getZData)
			return zdata, present
		}}
}

type chunkHeader struct {
	kind       []byte
	oid        []byte
	payloadLen uint32
	dataLen    uint32
}

// Read the header of a chunk at the given offset.
func readChunkHeader(fd *os.File, pos int64, header *chunkHeader) (nextPos int64, err os.Error) {
	var raw [48]byte
	_, err = fd.ReadAt(raw[:], pos)
	if err != nil {
		return
	}

	pos += 48
	if !bytes.Equal(raw[0:16], chunkMagic) {
		err = ChunkError
		return
	}

	payloadLen := readLE32(raw[16:20])
	dataLen := readLE32(raw[20:24])
	kind := make([]byte, 4)
	copy(kind, raw[24:28])
	oid := make([]byte, 20)
	copy(oid, raw[28:48])

	nextPos = (pos + int64(payloadLen) + 15) &^ 15
	header.kind = kind
	header.oid = oid
	header.payloadLen = payloadLen
	header.dataLen = dataLen
	return
}

// Read a chunk from the file, at the given position, returns, the
// chunk, the position where the next chunk should be, and any error.
func ReadChunk(fd *os.File, pos int64) (chunk Chunk, newPos int64, err os.Error) {
	var header chunkHeader
	newPos, err = readChunkHeader(fd, pos, &header)
	if err != nil {
		return
	}

	payload := make([]byte, header.payloadLen)
	_, err = fd.ReadAt(payload, pos+48)
	if err != nil {
		return
	}

	if header.dataLen == 0xFFFFFFFF {
		chunk = NewDataChunk(header.kind, header.oid, payload)
	} else {
		chunk = NewCompressedChunk(header.kind, header.oid, header.dataLen, payload)
	}

	return
}

func main2() {
	cfile, err := os.Open("npool/pool-data-0015.data", os.O_RDONLY, 0)
	if err != nil {
		panic(err)
	}

	count := 0
	dlen := 0
	zlen := 0
	pos := int64(0)
	for {
		chunk, npos, err := ReadChunk(cfile, pos)
		pos = npos
		if err == os.EOF {
			break
		}
		if err != nil {
			panic(err)
		}

		count++
		dlen += int(chunk.DataLen())
		zdata, present := chunk.ZData()
		if present {
			zlen += len(zdata)
		}

		// fmt.Printf("nextPos: 0x%x\n", pos)
		// fmt.Printf("data\n%q\n", chunk.Data())
	}
	fmt.Printf("%d chunks\n%d bytes\n%d zbytes\n", count, dlen, zlen)
}

func main3() {
	cfile, err := os.Open("npool/pool-data-0015.data", os.O_RDONLY, 0)
	if err != nil {
		panic(err)
	}
	defer cfile.Close()

	var header chunkHeader
	pos := int64(0)
	count := 0
	for {
		npos, err := readChunkHeader(cfile, pos, &header)
		pos = npos
		if err == os.EOF {
			break
		}
		if err != nil {
			panic(err)
		}

		count++
	}
	fmt.Printf("%d chunks\n", count)
}
