// Backups.

package dump

import (
	"fmt"
	"log"
	"os"
	"path"
	"strconv"
	"syscall"
	"time"

	"pool"
	"store"
)

type backupState struct {
	pool pool.Pool

	rootDev uint64
}

func Run(pl pool.Pool, path string, props map[string]string) (err error) {
	log.Printf("Backing up %q", path)

	var self backupState
	self.pool = pl

	self.Backup(path, props)

	return
}

func (self *backupState) Backup(path string, props map[string]string) (err error) {
	now := time.Now()

	rootFi, err := os.Lstat(path)
	if err != nil {
		return
	}

	// TODO: Get the filesystem UUID here.
	self.rootDev = rootFi.Sys().(*syscall.Stat_t).Dev

	headId, err := self.directory(path, rootFi)
	if err != nil {
		return
	}

	back := store.NewPropertyMap("back")
	for k, v := range props {
		back.Props[k] = v
	}
	back.Props["hash"] = headId.String()

	// The backup date property is in 'ms' since the start of unix
	// time.
	back.Props["_date"] = strconv.FormatInt(now.UnixNano()/1000000, 10)

	id, err := self.writeNode("back", back)
	if err != nil {
		return
	}

	err = self.pool.Flush()
	if err != nil {
		return
	}

	log.Printf("Backup complete: %s", id.String())
	return
}

func (self *backupState) directory(dirPath string, dirFi os.FileInfo) (oid *pool.OID, err error) {
	var children []os.FileInfo
	if dirFi.Sys().(*syscall.Stat_t).Dev == self.rootDev {
		children, err = Readdir(dirPath)
		if err != nil {
			return
		}
	} else {
		// Crossing a device, act as if we have no children.
		children = make([]os.FileInfo, 0)
	}

	writer := store.NewDirWriter(self.pool, 256*1024)

	for _, child := range children {
		raw := child.Sys().(*syscall.Stat_t)
		mode := raw.Mode

		var id *pool.OID
		// log.Printf("  mode: %o, dir?: %s", mode, isMode(mode, syscall.S_IFDIR))
		if isMode(mode, syscall.S_IFREG) {
			// log.Printf("f %s/%s", dirPath, child.Name())
			id, err = self.regularFile(path.Join(dirPath, child.Name()), child)
		} else if isMode(mode, syscall.S_IFDIR) {
			// log.Printf("D %s/%s", dirPath, child.Name())
			id, err = self.directory(path.Join(dirPath, child.Name()), child)
			if err != nil {
				return
			}
		} else {
			// log.Printf("- %s/%s", dirPath, child.Name())
			id, err = self.plainNode(path.Join(dirPath, child.Name()), child)
			if err != nil {
				return
			}
		}

		err = writer.Add(child.Name(), id)
		if err != nil {
			return
		}
	}

	childId, err := writer.Finalize()
	if err != nil {
		return
	}
	props := encodeProps(dirFi)
	props.Props["children"] = childId.String()

	return self.writeNode("node", props)
}

func (self *backupState) regularFile(name string, fi os.FileInfo) (oid *pool.OID, err error) {
	data, err := store.WriteFile(self.pool, name)
	if err != nil {
		return
	}

	props := encodeProps(fi)
	props.Props["data"] = data.String()

	return self.writeNode("node", props)
}

func (self *backupState) plainNode(name string, fi os.FileInfo) (oid *pool.OID, err error) {
	props := encodeProps(fi)

	if props.Kind == "LNK" {
		props.Props["target"], err = os.Readlink(name)
		if err != nil {
			return
		}
	}

	return self.writeNode("node", props)
}

func (self *backupState) writeNode(kind string, node *store.PropertyMap) (oid *pool.OID, err error) {
	ch := pool.NewChunk(kind, node.Encode())
	err = self.pool.Insert(ch)
	if err != nil {
		return
	}

	oid = ch.OID()
	return
}

// Given 'stat' information for a file, encode the textual backup
// properties that will be written to the backup.
func encodeProps(fi os.FileInfo) (result *store.PropertyMap) {
	raw := fi.Sys().(*syscall.Stat_t)

	addDev := false
	var kind string
	mode := raw.Mode
	switch {
	case isMode(mode, syscall.S_IFREG):
		kind = "REG"
	case isMode(mode, syscall.S_IFDIR):
		kind = "DIR"
	case isMode(mode, syscall.S_IFCHR):
		kind = "CHR"
		addDev = true
	case isMode(mode, syscall.S_IFBLK):
		kind = "BLK"
		addDev = true
	case isMode(mode, syscall.S_IFIFO):
		kind = "FIFO"
	case isMode(mode, syscall.S_IFLNK):
		kind = "LNK"
	case isMode(mode, syscall.S_IFSOCK):
		kind = "SOCK"
	default:
		panic("Unknown file type")
	}

	result = store.NewPropertyMap(kind)

	result.Props["mode"] = strconv.FormatUint(uint64(mode & ^uint32(syscall.S_IFMT)), 10)
	result.Props["dev"] = strconv.FormatUint(raw.Dev, 10)
	result.Props["ino"] = strconv.FormatUint(raw.Ino, 10)
	result.Props["nlink"] = strconv.FormatUint(raw.Nlink, 10)
	result.Props["uid"] = strconv.FormatUint(uint64(raw.Uid), 10)
	result.Props["gid"] = strconv.FormatUint(uint64(raw.Gid), 10)
	result.Props["size"] = strconv.FormatUint(uint64(raw.Size), 10)
	result.Props["mtime"] = fmt.Sprintf("%d.%09d", raw.Mtim.Sec, raw.Mtim.Nsec)
	result.Props["ctime"] = fmt.Sprintf("%d.%09d", raw.Ctim.Sec, raw.Ctim.Nsec)

	if addDev {
		result.Props["rdev"] = strconv.FormatUint(raw.Rdev, 10)
	}

	return
}

func isMode(mode uint32, match uint32) bool {
	return (mode & syscall.S_IFMT) == match
}
