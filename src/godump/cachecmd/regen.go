package cachecmd

import (
	"database/sql"
	"errors"
	"strconv"
	"time"

	"cache"
	"pool"
	"store"
)

// Set this to verify cache entries as they are updated.  The
// verification isn't currently correct, however, so will need to be
// fixed before this is actually useful.
var verifyCache = false

type regenState struct {
	store.PathTrackerImpl
	store.EmptyVisitor

	tx    *sql.Tx
	cache *cache.Cache

	uuid string
	dirs []*cache.DirInfo

	cwd *cache.DirInfo
}

func regen(pl pool.Pool, oid *pool.OID) (err error) {
	var self regenState
	self.InitPath()
	self.dirs = make([]*cache.DirInfo, 0)

	self.tx = pool.GetSql(pl)
	if self.tx == nil {
		err = errors.New("Pool type doesn't contain SQL database")
		return
	}

	err = store.Walk(pl, oid, &self)
	if err != nil {
		return
	}

	err = pl.Flush()
	return
}

// Called at the top of the backup.  Extract the fs-uid out of the
// backup.
func (self *regenState) Back(root *pool.OID, date time.Time, props map[string]string) (err error) {
	uuid, ok := props["fsuuid"]
	if !ok {
		err = errors.New("Backup doesn't contain fsuuid property")
		return
	}
	self.uuid = uuid

	self.cache, err = cache.NewCache(self.tx, uuid)
	if err != nil {
		return
	}

	// Base the cache time on the time of the backup, not the
	// current time.
	self.cache.BaseTime = date

	return
}

func (self *regenState) Enter(props *store.PropertyMap) (err error) {
	ino, err := strconv.ParseUint(props.Props["ino"], 10, 64)
	if err != nil {
		return
	}
	info := cache.NewDirInfo(ino)

	self.dirs = append(self.dirs, info)
	self.cwd = info
	return
}

func (self *regenState) Leave(props *store.PropertyMap) (err error) {
	info := self.cwd
	self.dirs = self.dirs[:len(self.dirs)-1]

	if len(self.dirs) > 0 {
		self.cwd = self.dirs[len(self.dirs)-1]
	} else {
		self.cwd = nil
	}

	err = self.cache.UpdateDir(info)
	if err != nil {
		return
	}

	if verifyCache {
		// Reload and compare?
		var d2 *cache.DirInfo
		d2, err = self.cache.GetDir(info.Ino)
		if err != nil {
			return
		}
		if len(d2.Files) != len(info.Files) {
			// Actually, this is kind of to be expected with old,
			// since the expire should already be considered.
			err = errors.New("Incorrect reread result")
			return
		}
	}
	return
}

func (self *regenState) Open(props *store.PropertyMap) (err error) {
	var fi cache.FileInfo

	fi.Ino, err = strconv.ParseUint(props.Props["ino"], 10, 64)
	if err != nil {
		return
	}

	fi.Ctime, err = store.DecodeTimestamp(props.Props["ctime"])
	if err != nil {
		return
	}

	fi.Data, err = pool.ParseOID(props.Props["data"])
	if err != nil {
		return
	}

	self.cache.SetExpire(&fi)
	self.cwd.Files[fi.Ino] = &fi

	err = store.Prune
	return
}
