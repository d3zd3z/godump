// Backup chunks.

package pool

import (
	"bytes"
	"compress/zlib"
	"errors"
	"io"
	"sync"
)

type Chunk interface {
	Kind() Kind
	OID() *OID
	Data() []byte
	DataLen() uint32
	ZData() (zdata []byte, present bool)
}

var ChunkError = errors.New("Error reading chunk")
var chunkMagic = []byte("adump-pool-v1.1\n")
var padding = make([]byte, 16)

// Data associated with any kind of chunk.
type sharedChunk struct {
	kind Kind
	oid  *OID
}

func (ch *sharedChunk) Kind() Kind { return ch.kind }
func (ch *sharedChunk) OID() *OID  { return ch.oid }

// Data chunks are ones started with the data originally uncompressed.
type dataChunk struct {
	sharedChunk
	data     []byte
	getZData func() (zdata []byte, present bool)
}

func (ch *dataChunk) Data() []byte          { return ch.data }
func (ch *dataChunk) DataLen() uint32       { return uint32(len(ch.data)) }
func (ch *dataChunk) ZData() ([]byte, bool) { return ch.getZData() }

func newDataChunk(kind Kind, oid *OID, data []byte) Chunk {
	var zdata []byte
	present := false

	getZData := func() {
		var zbuf bytes.Buffer
		w := zlib.NewWriter(&zbuf)
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

// Construct a chunk from a piece of internal data.
func NewChunk(kind string, data []byte) Chunk {
	if len(kind) != 4 {
		panic("Chunk kind must be 4 characters")
	}
	oid := BlobOID(kind, data)
	return newDataChunk(NewKind(kind), oid, data)
}

// Performing Chunk IO.

type chunkHeader struct {
	kind       []byte
	oid        *OID
	payloadLen uint32
	dataLen    uint32
}

// Write the Chunk encoded to the given writer.
func ChunkWrite(ch Chunk, w io.Writer) (err error) {
	var header bytes.Buffer
	_, err = header.Write(chunkMagic)
	if err != nil {
		return
	}

	zdata, hasZ := ch.ZData()

	var clen, uclen uint32
	var payload []byte

	if hasZ {
		clen = uint32(len(zdata))
		uclen = ch.DataLen()
		payload = zdata
	} else {
		clen = ch.DataLen()
		uclen = 0xFFFFFFFF
		payload = ch.Data()
	}

	err = writeLE32(&header, clen)
	if err != nil {
		return
	}

	err = writeLE32(&header, uclen)
	if err != nil {
		return
	}

	_, err = header.Write([]byte(ch.Kind().String()))
	if err != nil {
		return
	}

	_, err = header.Write(ch.OID()[:])
	if err != nil {
		return
	}

	_, err = w.Write(header.Bytes())
	if err != nil {
		return
	}

	_, err = w.Write(payload)
	if err != nil {
		return
	}

	padLen := 15 & -len(payload)
	if padLen > 0 {
		_, err = w.Write(padding[:padLen])
	}

	return
}

func writeLE32(w io.Writer, item uint32) (err error) {
	var buf [4]byte
	buf[0] = byte(item & 0xFF)
	buf[1] = byte((item >> 8) & 0xFF)
	buf[2] = byte((item >> 16) & 0xFF)
	buf[3] = byte((item >> 24) & 0xFF)
	_, err = w.Write(buf[:])
	return
}

func readLE32(piece []byte) uint32 {
	return uint32(piece[0]) |
		(uint32(piece[1]) << 8) |
		(uint32(piece[2]) << 16) |
		(uint32(piece[3]) << 24)
}

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
func newCompressedChunk(kind Kind, oid *OID, dataLen uint32, zdata []byte) Chunk {
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

// Read the header of a chunk at the given offset.
// TODO: Can this use encoding/binary.
func readChunkHeader(rd io.Reader, header *chunkHeader) (err error) {
	var raw [48]byte
	_, err = io.ReadFull(rd, raw[:])
	if err != nil {
		return
	}

	if !bytes.Equal(raw[0:16], chunkMagic) {
		err = ChunkError
		return
	}

	payloadLen := readLE32(raw[16:20])
	dataLen := readLE32(raw[20:24])
	kind := make([]byte, 4)
	copy(kind, raw[24:28])
	header.oid = new(OID)
	copy(header.oid[:], raw[28:48])

	header.kind = kind
	header.payloadLen = payloadLen
	header.dataLen = dataLen

	return
}

// Read a chunk from the reader.  Also returns an amount of padding
// that can be used to skip to the next chunk.
func ChunkRead(rd io.Reader) (chunk Chunk, pad int, err error) {
	var header chunkHeader
	err = readChunkHeader(rd, &header)
	if err != nil {
		return
	}

	payload := make([]byte, header.payloadLen)
	_, err = io.ReadFull(rd, payload)
	if err != nil {
		return
	}

	if header.dataLen == 0xFFFFFFFF {
		chunk = newDataChunk(NewKind(string(header.kind)), header.oid, payload)
	} else {
		chunk = newCompressedChunk(NewKind(string(header.kind)), header.oid, header.dataLen, payload)
	}

	pad = 15 & -int(header.payloadLen)

	return
}
