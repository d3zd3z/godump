// Store decoder.

package store

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"pool"
)

type walker struct {
	pool     pool.Pool
	handlers map[pool.Kind]handler
}

func Walk(p pool.Pool, root *pool.OID, visit *Visitor) (err error) {
	var walk walker
	walk.pool = p

	walk.handlers = map[pool.Kind]handler{
		pool.StringToKind("back"): walk.backHandler,
	}

	return walk.walk(root, visit)
}

func (walk *walker) walk(oid *pool.OID, visit *Visitor) (err error) {
	ch, err := walk.pool.Search(oid)
	if err != nil {
		return
	}
	if ch == nil {
		err = errors.New(fmt.Sprintf("Unable to read oid from pool: %q", oid.String()))
	}

	hand, ok := walk.handlers[ch.Kind()]
	if !ok {
		err = errors.New(fmt.Sprintf("Unsupported kind %q", ch.Kind().String()))
		return
	}

	return hand(ch, visit)
}

type handler func(chunk pool.Chunk, visit *Visitor) (err error)

func (walk *walker) backHandler(chunk pool.Chunk, visit *Visitor) (err error) {
	pmap, err := decodeProp(chunk.Data())
	if err != nil {
		return
	}
	tDate, ok := pmap.props["_date"]
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

	delete(pmap.props, "_date")

	err = visit.Back(chunk.OID(), date, pmap.props)

	if err != nil {
		if err == Prune {
			// If we've been asked to prune, skip
			// children, but don't return as error.
			err = nil
		}
		return
	}

	// TODO: Walk children.
	return
}
