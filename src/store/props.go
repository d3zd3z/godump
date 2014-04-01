package store

import (
	"bytes"
	"errors"
	"fmt"
)

// Property list conversion.
// This is a fairly simplistic encoding format.

type propertyMap struct {
	kind  string
	props map[string]string
}

func decodeProp(data []byte) (pmap propertyMap, err error) {
	pmap.props = make(map[string]string)

	buf := bytes.NewBuffer(data)
	pmap.kind, err = readString8(buf)
	if err != nil {
		return
	}

	for buf.Len() > 0 {
		var kind string
		kind, err = readString8(buf)
		var value string
		value, err = readString16(buf)
		pmap.props[kind] = value
	}

	return
}

func readString8(buf *bytes.Buffer) (result string, err error) {
	size, err := buf.ReadByte()
	if err != nil {
		return
	}
	return readString(buf, int(size))
}

func readString16(buf *bytes.Buffer) (result string, err error) {
	s1, err := buf.ReadByte()
	if err != nil {
		return
	}
	s2, err := buf.ReadByte()
	if err != nil {
		return
	}
	return readString(buf, (int(s1)<<8)|int(s2))
}

func readString(buf *bytes.Buffer, size int) (result string, err error) {
	tmp := buf.Next(size)
	if len(tmp) != size {
		err = errors.New("Short read in buffer")
		return
	}
	result = string(tmp)
	return
}

// For debugging.
func (p *propertyMap) Print() {
	fmt.Printf("Props: %q\n", p.kind)
	for k, v := range p.props {
		fmt.Printf("  %q: %q\n", k, v)
	}
}
