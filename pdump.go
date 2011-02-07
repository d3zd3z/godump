// Dump bytes as ascii.

package main

import (
	"bytes"
	"fmt"
	"os"
)

func pdump(data []byte) {
	var line bytes.Buffer
	var ascii bytes.Buffer

	length := len(data)
	offset := 0
	for length > 0 {
		line.Reset()
		ascii.Reset()

		lineBase := offset &^ 15
		line.WriteString(fmt.Sprintf("%08x: ", lineBase))
		ascii.WriteRune('|')

		for pos := lineBase; pos < lineBase+16; pos++ {
			if pos < offset || pos >= offset+length {
				line.WriteString("   ")
				ascii.WriteRune(' ')
			} else {
				ch := data[pos]
				line.WriteString(fmt.Sprintf("%02x ", ch))
				if ch >= 32 && ch <= 126 {
					ascii.WriteRune(int(ch))
				} else {
					ascii.WriteRune('.')
				}
			}

			if (pos & 15) == 7 {
				line.WriteRune(' ')
			}
		}
		ascii.WriteRune('|')

		line.WriteRune(' ')
		ascii.WriteRune('\n')
		line.WriteTo(os.Stdout)
		ascii.WriteTo(os.Stdout)

		oldOffset := offset
		offset = (offset + 16) &^ 15
		length -= offset - oldOffset
	}
}
