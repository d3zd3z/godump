// Store decoder.

package store

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"

	"pool"
)

type walker struct {
	pool     pool.Pool
	handlers map[pool.Kind]handler

	// The current visitor.
	visit *Visitor
}

func Walk(p pool.Pool, root *pool.OID, visit *Visitor) (err error) {
	var self walker
	self.pool = p
	self.visit = visit

	self.handlers = map[pool.Kind]handler{
		pool.StringToKind("back"): self.backHandler,
		pool.StringToKind("node"): self.nodeHandler,
		pool.StringToKind("dir "): self.dirHandler,
		pool.StringToKind("null"): self.nullHandler,

		pool.StringToKind("dir0"): self.indHandler,
		pool.StringToKind("dir1"): self.indHandler,
		pool.StringToKind("dir2"): self.indHandler,
		pool.StringToKind("ind0"): self.indHandler,
		pool.StringToKind("ind1"): self.indHandler,
		pool.StringToKind("ind2"): self.indHandler,
		pool.StringToKind("ind3"): self.indHandler,

		pool.StringToKind("blob"): self.blobHandler,
	}

	return self.walk(root)
}

func (self *walker) walk(oid *pool.OID) (err error) {

	err = self.visit.EarlyVisit(oid)
	if err != nil {
		err = dePrune(err)
		return
	}

	ch, err := self.pool.Search(oid)
	if err != nil {
		return
	}
	if ch == nil {
		err = errors.New(fmt.Sprintf("Unable to read oid from pool: %q", oid.String()))
	}

	err = self.visit.Chunk(ch)
	if err != nil {
		err = dePrune(err)
		return
	}

	hand, ok := self.handlers[ch.Kind()]
	if !ok {
		log.Printf("Unsupported kind %q", ch.Kind().String())
		// err = errors.New(fmt.Sprintf("Unsupported kind %q", ch.Kind().String()))
		return
	}

	return hand(ch)
}

type handler func(chunk pool.Chunk) (err error)

func (self *walker) backHandler(chunk pool.Chunk) (err error) {
	pmap, err := decodeProp(chunk.Data())
	if err != nil {
		return
	}
	tDate, ok := pmap.Props["_date"]
	if !ok {
		err = errors.New(fmt.Sprintf("Backup record for %q has no _date property", chunk.OID().String()))
		return
	}
	iDate, err := strconv.ParseInt(tDate, 10, 64)
	if err != nil {
		err = errors.New(fmt.Sprintf("Invalid _date property %q: %s", tDate, err))
		return
	}

	date := time.Unix(iDate/1000, (iDate%1000)*1000000)

	delete(pmap.Props, "_date")

	err = self.visit.Back(chunk.OID(), date, pmap.Props)

	if err != nil {
		err = dePrune(err)
		return
	}

	hash, ok := pmap.Props["hash"]
	if !ok {
		err = errors.New("'hash' property not present in 'back' node")
		return
	}
	id, err := pool.ParseOID(hash)
	if err != nil {
		return
	}

	err = self.walk(id)
	return
}

func (self *walker) nodeHandler(chunk pool.Chunk) (err error) {
	pmap, err := decodeProp(chunk.Data())
	if err != nil {
		return
	}

	switch pmap.Kind {
	case "DIR":
		// Pruning 'enter' prevents 'leave' from being called.
		err = self.visit.Enter(pmap)
		if err != nil {
			err = dePrune(err)
			return
		}

		var children *pool.OID
		children, err = pool.ParseOID(pmap.Props["children"])
		if err != nil {
			return
		}

		err = self.walk(children)
		if err != nil {
			return
		}

		err = self.visit.Leave(pmap)
		if err != nil {
			return
		}

	case "REG":
		// Pruning 'Open' prevents Close from being called.
		err = self.visit.Open(pmap)
		if err != nil {
			err = dePrune(err)
			return
		}

		var data *pool.OID
		data, err = pool.ParseOID(pmap.Props["data"])
		if err != nil {
			return
		}

		err = self.walk(data)
		if err != nil {
			return
		}

		err = self.visit.Close(pmap)
		if err != nil {
			return
		}

	default:
		err = self.visit.Node(pmap)
		if err != nil {
			return
		}
	}

	return
}

// The dirnode for the direct children.
func (self *walker) dirHandler(chunk pool.Chunk) (err error) {
	buf := bytes.NewBuffer(chunk.Data())

	for buf.Len() > 0 {
		var name string
		name, err = readString16(buf)
		if err != nil {
			return
		}

		var oid *pool.OID
		oid, err = pool.OIDFromBytes(buf)
		if err != nil {
			return
		}
		// fmt.Printf("%s %q\n", oid.String(), name)

		self.visit.pushPath(name)
		err = self.walk(oid)
		self.visit.popPath()
		if err != nil {
			return
		}
	}
	return
}

// Both directory and file indirect blocks are the same format, just a
// bunch of OIDs.
func (self *walker) indHandler(chunk pool.Chunk) (err error) {
	buf := bytes.NewBuffer(chunk.Data())

	for buf.Len() > 0 {
		var oid *pool.OID
		oid, err = pool.OIDFromBytes(buf)
		if err != nil {
			return
		}

		err = self.walk(oid)
		if err != nil {
			return
		}
	}
	return
}

func (self *walker) blobHandler(chunk pool.Chunk) (err error) {
	return self.visit.Blob(chunk)
}

// Null means either empty file, or empty directory.  In either case,
// there is nothing to do.
func (self *walker) nullHandler(chunk pool.Chunk) (err error) {
	return
}

// Removes the pruning on an error, if the error is 'Prune'.
// Otherwise, just returns the error unmodified.
func dePrune(err error) error {
	if err == Prune {
		return nil
	} else {
		return err
	}
}
