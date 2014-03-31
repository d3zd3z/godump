// SQLite-based storage pools

package pool

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"os"

	"code.google.com/p/go-uuid/uuid"
	_ "github.com/mattn/go-sqlite3"
)

// Construct a fresh new pool in under the given name.  The name must
// be a name that can be made as a fresh directory.
func CreateSqlPool(path string) (err error) {
	err = os.Mkdir(path, 0755)
	if err != nil {
		return
	}

	err = os.Mkdir(path+"/blobs", 0755)
	if err != nil {
		return
	}

	db, err := sql.Open("sqlite3", path+"/data.db")
	if err != nil {
		return
	}
	defer db.Close()
	err = setSchema(db, &poolSchema)
	if err != nil {
		return
	}

	_, err = db.Exec("insert into props (key, value) values (?, ?)",
		"uuid",
		uuid.New())
	if err != nil {
		return
	}

	return
}

type SqlPool struct {
	base string
	db   *sql.DB
	tx   *sql.Tx
}

// Open an existing storage pool.
func OpenSqlPool(path string) (pf Pool, err error) {
	var pool SqlPool
	pool.base = path

	pool.db, err = sql.Open("sqlite3", path+"/data.db")
	if err != nil {
		return
	}

	err = checkSchema(pool.db, &poolSchema)
	if err != nil {
		pool.db.Close()
		return
	}

	pool.tx, err = pool.db.Begin()
	if err != nil {
		pool.db.Close()
		return
	}

	pf = &pool
	return
}

func (pool *SqlPool) Close() (err error) {
	err = pool.db.Close()
	return
}

func (pool *SqlPool) Flush() (err error) {
	err = pool.tx.Commit()
	if err != nil {
		return err
	}
	pool.tx, err = pool.db.Begin()
	return
}

func (pool *SqlPool) Insert(chunk Chunk) (err error) {
	has, err := pool.Contains(chunk.OID())
	if err != nil || has {
		return
	}

	var zsize uint32
	zdata, present := chunk.ZData()
	if present && len(zdata) < int(chunk.DataLen()) {
		zsize = uint32(len(zdata))
	} else {
		zsize = chunk.DataLen()
		zdata = chunk.Data()
	}

	if zsize > 100000 {
		dir, file := pool.makeName(chunk.OID())
		tmpErr := ioutil.WriteFile(file, zdata, 0644)
		if tmpErr != nil {
			err = os.Mkdir(dir, 0755)
			if err != nil {
				return
			}
			err = ioutil.WriteFile(file, zdata, 0644)
			if err != nil {
				return
			}
		}
		zdata = nil
	}

	_, err = pool.tx.Exec("INSERT OR FAIL INTO blobs (oid, kind, size, zsize, data) VALUES (?, ?, ?, ?, ?)",
		chunk.OID()[:], chunk.Kind().String(),
		chunk.DataLen(),
		zsize,
		zdata)
	return
}

func (pool *SqlPool) Search(oid *OID) (chunk Chunk, err error) {
	row := pool.tx.QueryRow("SELECT kind, size, zsize, data from BLOBS where oid = ?",
		oid[:])
	var kind string
	var size int
	var zsize int
	var data []byte
	err = row.Scan(&kind, &size, &zsize, &data)
	if err != nil {
		return
	}

	if size == 0 {
		chunk = newDataChunk(StringToKind(kind), oid, []byte{})
		return
	}

	if data == nil {
		_, file := pool.makeName(oid)
		data, err = ioutil.ReadFile(file)
		if err != nil {
			return
		}
		if zsize != len(data) {
			panic("Incorrect size read for chunk")
		}
	}

	if size == zsize {
		chunk = newDataChunk(StringToKind(kind), oid, data)
	} else {
		chunk = newCompressedChunk(StringToKind(kind), oid, uint32(size), data)
	}
	return
}

func (pool *SqlPool) Backups() (backups []*OID, err error) {
	result := make([]*OID, 0)
	rows, err := pool.tx.Query("SELECT oid FROM blobs WHERE kind = 'back'")
	if err != nil {
		return
	}
	for rows.Next() {
		var oid []byte
		err = rows.Scan(&oid)
		if err != nil {
			// TODO: Do we need to close?
			return
		}
		var piece OID
		copy(piece[:], oid)
		result = append(result, &piece)
	}
	backups = result
	return
}

func (pool *SqlPool) makeName(oid *OID) (dir, file string) {
	dir = fmt.Sprintf("%s/blobs/%02x", pool.base, oid[0])
	file = fmt.Sprintf("%s/%x", dir, oid[1:])
	return
}

func (pool *SqlPool) Contains(oid *OID) (result bool, err error) {
	row := pool.tx.QueryRow("SELECT COUNT(*) FROM blobs WHERE oid = ?",
		oid[:])

	var count int
	err = row.Scan(&count)
	if err != nil {
		return
	}

	result = (count == 1)
	return
}

var poolSchema = schema{
	version: "1:2014-03-18",
	schema: []string{
		`CREATE TABLE blobs (
			id integer primary key,
			oid blob unique not null,
			kind text,
			size integer,
			zsize integer,
			data blob)`,
		`CREATE INDEX blobs_oid ON blobs(oid)`,
		`CREATE INDEX blobs_backs ON blobs(kind) where kind = 'back'`,
		`CREATE TABLE props (
			key text primary key,
			value text)`,
		`CREATE TABLE filesystems (
			fsid INTEGER PRIMARY KEY,
			uuid TEXT)`,
		`CREATE TABLE ctime_cache (
			fsid INTEGER REFERENCES filesystems (fsid) NOT NULL,
			pino INTEGER NOT NULL,
			expire DOUBLE NOT NULL,
			info BLOB,
			PRIMARY KEY (fsid, pino))`,
	},
}
