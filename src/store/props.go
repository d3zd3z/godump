package store

import (
	"bytes"
	"errors"
	"fmt"
	"sort"
	"strconv"
)

// Property list conversion.
// This is a fairly simplistic encoding format.

type PropertyMap struct {
	Kind  string
	Props map[string]string
}

func NewPropertyMap(kind string) *PropertyMap {
	return &PropertyMap{Kind: kind, Props: make(map[string]string)}
}

func decodeProp(data []byte) (pmap *PropertyMap, err error) {
	var result PropertyMap
	result.Props = make(map[string]string)

	buf := bytes.NewBuffer(data)
	result.Kind, err = readString8(buf)
	if err != nil {
		return
	}

	for buf.Len() > 0 {
		var kind string
		kind, err = readString8(buf)
		if err != nil {
			return
		}

		var value string
		value, err = readString16(buf)
		if err != nil {
			return
		}

		result.Props[kind] = value
	}

	pmap = &result
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
func (p *PropertyMap) Print() {
	fmt.Printf("Props: %q\n", p.Kind)
	for k, v := range p.Props {
		fmt.Printf("  %q: %q\n", k, v)
	}
}

// Encode the given properties to a block of bytes.  The properties
// will be encoded in lexicographical order by key so that the same
// properties will always encode the same way.
func (p *PropertyMap) Encode() (result []byte) {
	var buf bytes.Buffer

	writeString8(&buf, p.Kind)

	keys := make([]string, 0, len(p.Props))
	for k := range p.Props {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		writeString8(&buf, k)
		writeString16(&buf, p.Props[k])
	}

	return buf.Bytes()
}

func writeString8(buf *bytes.Buffer, text string) {
	if len(text) > 255 {
		panic("String is too long")
	}
	buf.WriteByte(byte(len(text)))
	buf.WriteString(text)
}

func writeString16(buf *bytes.Buffer, text string) {
	size := len(text)
	if size > 0xffff {
		panic("String is too long")
	}
	buf.WriteByte(byte(size >> 8))
	buf.WriteByte(byte(size))
	buf.WriteString(text)
}

// User utilities for extracting properties.
func (self *PropertyMap) GetInt(name string) (value int, err error) {
	text, ok := self.Props[name]
	if !ok {
		err = errors.New(fmt.Sprintf("Missing property: %q", name))
		return
	}
	tmp, err := strconv.ParseInt(text, 10, 32)
	if err != nil {
		return
	}

	value = int(tmp)
	return
}
