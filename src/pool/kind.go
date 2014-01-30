// Chunk kinds.

package pool

import (
	"bytes"
	"encoding/binary"
)

// A kind is a 32-bit value that can be viewed as a string when
// needed.  It is represented little-endian, so that it will encode
// in the textual order, when written little endian.
type Kind uint32

func StringToKind(text string) (result Kind) {
	return BytesToKind([]byte(text))
}

func BytesToKind(text []byte) (result Kind) {
	if len(text) != 4 {
		panic("Invalid kind length")
	}

	buf := bytes.NewBuffer(text)
	err := binary.Read(buf, binary.LittleEndian, &result)
	if err != nil {
		panic("Error reading 32-bit value")
	}
	return
}

func (k Kind) Bytes() []byte {
	var buf bytes.Buffer
	err := binary.Write(&buf, binary.LittleEndian, k)
	if err != nil {
		panic("Error writing 32-bit value")
	}
	return buf.Bytes()
}

func (k Kind) String() string {
	return string(k.Bytes())
}
