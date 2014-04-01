// The decoder visitor.

package store

import (
	"bytes"
	"errors"
	"time"

	"pool"
)

// The visitors can return this specific 'Prune' error to indicate
// that children should not be visited.
var Prune = errors.New("Prune backup")

type Visitor struct {
	// Before any chunk is visited, calls EarlyVisit with the OID.
	// This can both do things with it, but can also return Prune
	// to indicate that this node shouldn't even be decoded.
	EarlyVisit func(key *pool.OID) error

	// After reading each chunk, calls Chunk on the chunk itself.
	// This can be used to prune the tree if desired.
	Chunk func(chunk pool.Chunk) error

	// Decoded node visitors.  These are called on the decoded
	// contents of nodes.
	Back  func(root *pool.OID, date time.Time, props map[string]string) error
	Enter func(props *PropertyMap) error
	Leave func(props *PropertyMap) error
	Open  func(props *PropertyMap) error
	Close func(props *PropertyMap) error
	Node  func(props *PropertyMap) error
	Blob  func(chunk pool.Chunk) error

	// The visitor also tracks the current path.
	path []string
}

func NewVisitor() *Visitor {
	var v Visitor

	v.EarlyVisit = EmptyEarlyVisitor
	v.Chunk = EmptyChunkVisitor

	v.Back = BackEmptyVisitor
	v.Enter = EmptyPropVisitor
	v.Leave = EmptyPropVisitor
	v.Open = EmptyPropVisitor
	v.Close = EmptyPropVisitor
	v.Node = EmptyPropVisitor

	v.Blob = EmptyChunkVisitor

	v.path = make([]string, 0)

	return &v
}

// Empty visitors.
func BackEmptyVisitor(root *pool.OID, date time.Time, props map[string]string) error {
	return nil
}

func EmptyPropVisitor(props *PropertyMap) error {
	return nil
}

func EmptyEarlyVisitor(key *pool.OID) error {
	return nil
}

func EmptyChunkVisitor(chunk pool.Chunk) error {
	return nil
}

// Add a path entry.
func (self *Visitor) pushPath(name string) {
	self.path = append(self.path, name)
}

// Remove a path entry.
func (self *Visitor) popPath() {
	self.path = self.path[0 : len(self.path)-1]
}

// Get the current path, based off the given pathname (can be "/" or
// ".", or another path).
func (self *Visitor) Path(base string) (result string) {
	var buf bytes.Buffer

	buf.WriteString(base)

	for _, elt := range self.path {
		if buf.Len() > 0 {
			buf.WriteRune('/')
		}
		buf.WriteString(elt)
	}
	return buf.String()
}
