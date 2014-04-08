// Restore a backup.

package restore

import (
	"errors"
	"fmt"
	"log"
	"os"
	"syscall"

	"meter"
	"pool"
	"store"
)

type restoreState struct {
	base string

	// Current open file.
	file *os.File

	// Part of progress meter.
	chunkCount int64
	byteCount  int64
	zbyteCount int64
	fileCount  int64
	dirCount   int64

	store.PathTrackerImpl
	store.EmptyVisitor
}

func Run(pl pool.Pool, id *pool.OID, path string) (err error) {
	var state restoreState
	state.base = path
	state.InitPath()

	err = store.Walk(pl, id, &state)
	if err != nil {
		return
	}
	meter.Sync(&state, true)
	return
}

func (self *restoreState) Open(props *store.PropertyMap) (err error) {
	self.file, err = os.OpenFile(self.FullPath(),
		os.O_WRONLY|os.O_CREATE|os.O_EXCL,
		0600)
	self.fileCount++
	meter.Sync(self, false)
	return
}

func (self *restoreState) Close(props *store.PropertyMap) (err error) {
	err = self.file.Close()
	if err != nil {
		return
	}
	return restoreReg(self.FullPath(), props)
}

func (self *restoreState) Enter(props *store.PropertyMap) (err error) {
	// TODO: Do we want to special case the root directory should
	// exist, or should we always restore into a new dir, and
	// require things to be moved out later.
	name := self.FullPath()
	err = os.Mkdir(name, 0700)

	self.dirCount++
	meter.Sync(self, false)
	return
}

func (self *restoreState) Leave(props *store.PropertyMap) (err error) {
	return restoreReg(self.FullPath(), props)
}

func (self *restoreState) Node(props *store.PropertyMap) (err error) {
	switch props.Kind {
	case "LNK":
		err = restoreLink(self.FullPath(), props)
	default:
		log.Printf("TODO: Restore node %q: %s", props.Kind, self.FullPath())
	}
	return
}

// Restore the stats on the given file.
func restoreReg(path string, props *store.PropertyMap) (err error) {
	err = propChown(path, props, os.Chown)
	if err != nil {
		return
	}

	mode, err := props.GetInt("mode")
	if err != nil {
		return
	}
	err = syscall.Chmod(path, uint32(mode))
	if err != nil {
		return
	}

	return restoreTime(path, props)
}

// Restore a symlink.  There isn't an lchmod in Linux, but we can set
// a umask before creating the node.  The link permissions aren't
// useful anyway, but it's nice to restore them correctly.
func restoreLink(path string, props *store.PropertyMap) (err error) {
	mode, err := props.GetInt("mode")
	if err != nil {
		return
	}

	target, ok := props.Props["target"]
	if !ok {
		err = errors.New("Symlink doesn't contain a 'target'")
		return
	}
	oldMode := syscall.Umask(mode & 4095)
	err = os.Symlink(target, path)
	syscall.Umask(oldMode)
	if err != nil {
		return
	}

	err = propChown(path, props, os.Lchown)
	return
}

// Perform an appropriate chown operation.
func propChown(path string, props *store.PropertyMap, chown func(string, int, int) error) (err error) {
	if !isRoot {
		return
	}
	uid, err := props.GetInt("uid")
	if err != nil {
		return
	}

	gid, err := props.GetInt("gid")
	if err != nil {
		return
	}

	err = chown(path, uid, gid)
	return
}

// Restore the timestamp on the given node.
func restoreTime(path string, props *store.PropertyMap) (err error) {
	when, err := store.DecodeTimestamp(props.Props["mtime"])
	if err != nil {
		return
	}
	err = os.Chtimes(path, when, when)
	return
}

func (self *restoreState) Blob(chunk pool.Chunk) (err error) {
	_, err = self.file.Write(chunk.Data())
	return
}

func (self *restoreState) FullPath() string {
	return self.Path(self.base)
}

func (self *restoreState) Chunk(chunk pool.Chunk) (err error) {
	self.chunkCount++
	self.byteCount += int64(chunk.DataLen())
	zdata, present := chunk.ZData()
	if present {
		self.zbyteCount += int64(len(zdata))
	} else {
		self.zbyteCount += int64(chunk.DataLen())
	}
	meter.Sync(self, false)
	return
}

// Generate the progress meter.
func (self *restoreState) GetMeter() (result []string) {
	result = make([]string, 6)

	result[0] = "----------------------------------------------------------------------"
	result[1] = fmt.Sprintf("   %11d chunks, %9d files, %9d dirs", self.chunkCount, self.fileCount, self.dirCount)
	result[2] = fmt.Sprintf("   %s data", meter.Humanize(self.byteCount))
	result[3] = fmt.Sprintf("   %s zdata (%5.1f%%)", meter.Humanize(self.zbyteCount),
		100.0*float64(self.zbyteCount)/float64(self.byteCount))

	path := self.FullPath()
	if len(path) > 73 {
		path = "..." + path[len(path)-60:]
	}
	result[4] = fmt.Sprintf(" : %q", path)
	result[5] = "----------------------------------------------------------------------"
	return
}

// Holds whether we are running as root or not.
var isRoot bool

func init() {
	isRoot = os.Geteuid() == 0
}
