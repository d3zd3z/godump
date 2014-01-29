// Chunk kinds.

package pool

// A kind is a 32-bit value that can be viewed as a string when
// needed.
type Kind uint32

func NewKind(text string) Kind {
	if len(text) != 4 {
		panic("Invalid kind length")
	}

	return Kind(uint32(text[0]) | (uint32(text[1]) << 8) | (uint32(text[2]) << 16) | (uint32(text[3]) << 24))
}

func (k Kind) String() string {
	result := make([]byte, 4)
	result[0] = byte(k & 0xFF)
	result[1] = byte((k >> 8) & 0xFF)
	result[2] = byte((k >> 16) & 0xFF)
	result[3] = byte((k >> 24) & 0xFF)
	return string(result)
}
