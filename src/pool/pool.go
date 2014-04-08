// Storage pool management.

package pool

import (
	"database/sql"
	"fmt"
	"os"
)

type Pool interface {
	Close() (err error)
	Flush() (err error)
	Insert(chunk Chunk) (err error)
	Contains(oid *OID) (result bool, err error)

	// Scan for the given chunk.  Non-nil error indicates an
	// error.  The 'chunk' will be non-nil if it could be found.
	Search(oid *OID) (chunk Chunk, err error)

	// Return the OID's of all backups written.
	Backups() (backups []*OID, err error)
}

func OpenPool(base string) (pf Pool, err error) {
	fi, err := os.Stat(base + "/data.db")
	if err != nil || !fi.Mode().IsRegular() {
		err = fmt.Errorf("Does not appear to be pool: '%s'", err)
		return
	}

	return OpenSqlPool(base)
}

// Some pools may have an underlying SQL database.  If this is the
// case, return that transaction handle for that database (which
// should be valid until the next "flush").  Otherwise, returns nil to
// indicate there is no database handle.
func GetSql(p Pool) (handle *sql.Tx) {
	hand, ok := p.(SqlablePool)
	if !ok {
		return
	}

	return hand.GetSqlTx()
}

type SqlablePool interface {
	GetSqlTx() *sql.Tx
}
