package store

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
)

// Property list conversion.
// This is a fairly simplistic encoding format.

type PropertyMap struct {
	Kind  string
	Props map[string]string
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
