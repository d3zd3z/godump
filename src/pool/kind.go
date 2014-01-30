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

// The program typically has a small number of kinds, so just memoize
// the ones we use.
var kindMap map[string]Kind

func init() {
	kindMap = make(map[string]Kind)
}

func StringToKind(text string) (result Kind) {

	result, ok := kindMap[text]
	if ok {
		return
	}

	if len(text) != 4 {
		panic("Invalid kind length")
	}

	buf := bytes.NewBuffer([]byte(text))
	err := binary.Read(buf, binary.LittleEndian, &result)
	if err != nil {
		panic("Error reading 32-bit value")
	}

	kindMap[text] = result

	return
}

func BytesToKind(text []byte) (result Kind) {
	return StringToKind(string(text))
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
