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

// Every visitor should track the current path.  Easiest is to include
// PathTrackerImpl to implement this interface.
type PathTracker interface {
	PushPath(name string)
	PopPath()
	Path(base string) string
}

type Visitor interface {
	PathTracker

	// Before any chunk is visited, calls EarlyVisit with the OID.
	// This can both do things with it, but can also return Prune
	// to indicate that this node shouldn't even be decoded.
	EarlyVisit(key *pool.OID) error

	// After reading each chunk, calls Chunk on the chunk itself.
	// This can be used to prune the tree if desired.
	Chunk(chunk pool.Chunk) error

	// Decoded node visitors.  These are called on the decoded
	// contents of nodes.
	Back(root *pool.OID, date time.Time, props map[string]string) error
	Enter(props *PropertyMap) error
	Leave(props *PropertyMap) error
	Open(props *PropertyMap) error
	Close(props *PropertyMap) error
	Node(props *PropertyMap) error
	Blob(chunk pool.Chunk) error
}

// The empty visitor can be included to provide empty default
// implementations.
type EmptyVisitor struct{}

type PathTrackerImpl struct {
	// The visitor also tracks the current path.
	path []string
}

// Empty visitors.

func (self *EmptyVisitor) Back(root *pool.OID, date time.Time, props map[string]string) error {
	return nil
}

func (self *EmptyVisitor) EarlyVisit(key *pool.OID) error { return nil }
func (self *EmptyVisitor) Chunk(pool pool.Chunk) error    { return nil }
func (self *EmptyVisitor) Enter(props *PropertyMap) error { return nil }
func (self *EmptyVisitor) Leave(props *PropertyMap) error { return nil }
func (self *EmptyVisitor) Open(props *PropertyMap) error  { return nil }
func (self *EmptyVisitor) Close(props *PropertyMap) error { return nil }
func (self *EmptyVisitor) Node(props *PropertyMap) error  { return nil }
func (self *EmptyVisitor) Blob(chunk pool.Chunk) error    { return nil }

func (self *PathTrackerImpl) InitPath() {
	self.path = make([]string, 0)
}

func (self *PathTrackerImpl) PushPath(name string) {
	self.path = append(self.path, name)
}

func (self *PathTrackerImpl) PopPath() {
	self.path = self.path[0 : len(self.path)-1]
}

// Get the current path, based off the given pathname (can be "/" or
// ".", or another path).
func (self *PathTrackerImpl) Path(base string) (result string) {
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
