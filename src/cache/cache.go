package cache

import (
	"database/sql"
	"math/rand"
	"strconv"
	"time"

	"pool"
)

type Cache struct {
	tx *sql.Tx

	fsid int

	// The time that expiration dates are based on.  For backups,
	// this will be the time as of the start of the backup.  For
	// regenerated caches, this can be based off of the start of
	// the backup (which may cause entries to be expired as they
	// are written).  NewCache will set this to the current time,
	// but it can be changed if desired.
	BaseTime time.Time

	// The time we created the cache.  Entries that expire before
	// this time will be discarded.
	now time.Time
}

// Cached information for a given directory.
type DirInfo struct {
	// The inode of the directory itself.
	Ino uint64

	// A mapping from file inode number to some info about that
	// file.
	Files map[uint64]*FileInfo
}

type FileInfo struct {
	// The inode number of this file.
	Ino uint64

	// The file's ctime.
	Ctime time.Time

	// The OID of the file contents.
	Data *pool.OID

	// An expiration time for this info.
	Expire time.Time
}

func NewCache(tx *sql.Tx, fsUuid string) (result *Cache, err error) {
	var self Cache

	self.tx = tx

	// Get the fsid.
	_, err = tx.Exec("INSERT OR IGNORE INTO filesystems (uuid) values (?)",
		fsUuid)
	if err != nil {
		return
	}

	row := tx.QueryRow("SELECT fsid FROM filesystems WHERE uuid = ?",
		fsUuid)
	err = row.Scan(&self.fsid)
	if err != nil {
		return
	}

	self.now = time.Now()
	self.BaseTime = self.now

	result = &self
	return
}

func NewDirInfo(ino uint64) *DirInfo {
	return &DirInfo{
		Ino:   ino,
		Files: make(map[uint64]*FileInfo),
	}
}

// Write out the cache information for a given directory.
func (self *Cache) UpdateDir(di *DirInfo) (err error) {
	// First, figure out the associated directory.

	_, err = self.tx.Exec("INSERT OR IGNORE INTO ctime_dirs (fsid, pino) values (?, ?)",
		self.fsid, di.Ino)
	if err != nil {
		return
	}

	// TODO: Can we just get the rowID back?
	row := self.tx.QueryRow("SELECT pkey FROM ctime_dirs WHERE fsid = ? AND pino = ?",
		self.fsid, di.Ino)
	var pkey int
	err = row.Scan(&pkey)
	if err != nil {
		return
	}

	// Remove any existing entries.
	_, err = self.tx.Exec("DELETE FROM ctime_cache WHERE pkey = ?",
		pkey)
	if err != nil {
		return
	}

	// Insert all of the files.
	stmt, err := self.tx.Prepare("INSERT INTO ctime_cache (pkey, ino, expire, ctime, oid) values (?, ?, ?, ?, ?)")
	if err != nil {
		return
	}

	for _, fi := range di.Files {
		_, err = stmt.Exec(pkey, fi.Ino, fi.Expire.UnixNano(), fi.Ctime.UnixNano(), fi.Data[:])
		if err != nil {
			return
		}
	}

	return
}

// Read the cache data for a given directory.
func (self *Cache) GetDir(ino uint64) (di *DirInfo, err error) {
	var dir DirInfo

	dir.Ino = ino
	dir.Files = make(map[uint64]*FileInfo)

	rows, err := self.tx.Query(`
		SELECT ino, ctime, expire, oid
		FROM ctime_cache NATURAL JOIN ctime_dirs NATURAL JOIN filesystems
		WHERE fsid = ? AND pino = ?`,
		self.fsid, ino)
	if err != nil {
		return
	}

	for rows.Next() {
		var ino uint64
		var ctimeText, expireText string
		var data []byte
		err = rows.Scan(&ino, &ctimeText, &expireText, &data)
		if err != nil {
			return
		}

		var ctime, expire time.Time
		ctime, err = decodeTime(ctimeText)
		if err != nil {
			return
		}
		expire, err = decodeTime(expireText)
		if err != nil {
			return
		}
		if len(data) != pool.OIDLen {
			panic("Incorrect OID length")
		}
		var oid pool.OID
		copy(oid[:], data)

		file := &FileInfo{
			Ino:    ino,
			Ctime:  ctime,
			Expire: expire,
			Data:   &oid}
		dir.Files[ino] = file
	}

	di = &dir
	return
}

// Set an expire time for the given fileinfo.  Currently, this is a
// random value between 2-6 weeks after the time of the backup.
func (self *Cache) SetExpire(fi *FileInfo) {
	week := time.Hour * 24 * 7
	age := time.Duration(rand.Int63n(4*int64(week))) + 2*week

	fi.Expire = self.BaseTime.Add(age)
}

func decodeTime(text string) (t time.Time, err error) {
	num, err := strconv.ParseInt(text, 10, 64)
	if err != nil {
		return
	}

	t = time.Unix(0, num)
	return
}
