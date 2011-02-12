// Object ID.

package pool

import (
	"crypto/sha1"
	"bytes"
	"fmt"
	"io"
	"os"
	"strconv"
)

type OID []byte

func (item OID) String() string {
	return fmt.Sprintf("%x", []byte(item))
}

func ParseOID(text string) (oid OID, err os.Error) {
	_, err = fmt.Sscanf(text, "%x", &oid)
	return
}

func (me OID) Compare(other OID) int {
	return bytes.Compare([]byte(me), []byte(other))
}

func intHash(index int) (oid OID) {
	hash := sha1.New()
	io.WriteString(hash, "blob")
	io.WriteString(hash, strconv.Itoa(index))
	return OID(hash.Sum())
}

