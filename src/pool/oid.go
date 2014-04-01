// Object ID.

package pool

import (
	"bytes"
	"crypto/sha1"
	"errors"
	"fmt"
	"io"
	"strconv"
)

type OID [20]byte

func (item *OID) String() string {
	return fmt.Sprintf("%x", item[:])
}

func ParseOID(text string) (oid *OID, err error) {
	tmp := make([]byte, len(oid))
	count, err := fmt.Sscanf(text, "%x", &tmp)
	if err != nil {
		return
	}
	if count != 1 {
		err = errors.New("Unable to parse OID")
		return
	}
	if len(tmp) != len(oid) {
		err = errors.New("Short textual OID")
		return
	}
	var result OID
	copy(result[:], tmp)
	oid = &result
	return
}

func (me *OID) Compare(other *OID) int {
	return bytes.Compare(me[:], other[:])
}

func BlobOID(kind string, data []byte) (oid *OID) {
	if len(kind) != 4 {
		panic("blob kind must be 4 bytes long")
	}
	hash := sha1.New()
	count, err := io.WriteString(hash, kind)
	if count != 4 || err != nil {
		panic("Unable to write kind")
	}

	count, err = hash.Write(data)
	if count != len(data) || err != nil {
		panic("Unable to write data")
	}
	var result OID
	hash.Sum(result[:0])
	return &result
}

// Extract the next OID out of the bytes.Buffer.
func OIDFromBytes(buf *bytes.Buffer) (oid *OID, err error) {
	tmp := buf.Next(20)
	if len(tmp) != 20 {
		err = errors.New("Short read of OID from buffer")
		return
	}
	var result OID
	copy(result[:], tmp)
	oid = &result
	return
}

// For tests, it's handy to be able to make hashes based on integers.
func IntOID(index int) (oid *OID) {
	return BlobOID("blob", []byte(strconv.Itoa(index)))
}
