// The decoder visitor.

package store

import (
	"errors"
	"time"

	"pool"
)

// The visitors can return this specific 'Prune' error to indicate
// that children should not be visited.
var Prune = errors.New("Prune backup")

type Visitor struct {
	Back func(root *pool.OID, date time.Time, props map[string]string) error
}

func NewVisitor() *Visitor {
	var v Visitor
	v.Back = BackEmptyVisitor

	return &v
}

// Empty visitors.
func BackEmptyVisitor(root *pool.OID, date time.Time, props map[string]string) error {
	return nil
}
