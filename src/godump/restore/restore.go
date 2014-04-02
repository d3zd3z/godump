// Restore a backup.

package restore

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"meter"
	"pool"
	"store"
)

type restoreState struct {
	base  string
	visit *store.Visitor

	// Current open file.
	file *os.File

	// Part of progress meter.
	chunkCount int64
	byteCount  int64
	zbyteCount int64
}

func newRestoreState(base string) *restoreState {
	var self restoreState

	self.base = base
	self.visit = store.NewVisitor()

	self.visit.Open = self.open
	self.visit.Close = self.close
	self.visit.Enter = self.enter
	self.visit.Leave = self.leave
	self.visit.Blob = self.blob
	self.visit.Chunk = self.chunk

	return &self
}

func Run(pl pool.Pool, id *pool.OID, path string) (err error) {
	state := newRestoreState(path)

	err = store.Walk(pl, id, state.visit)
	if err != nil {
		return
	}
	meter.Sync(state, true)
	return
}

func (self *restoreState) open(props *store.PropertyMap) (err error) {
	self.file, err = os.OpenFile(self.Path(),
		os.O_WRONLY|os.O_CREATE|os.O_EXCL,
		0600)
	return
}

func (self *restoreState) close(props *store.PropertyMap) (err error) {
	err = self.file.Close()
	if err != nil {
		return
	}
	return restoreReg(self.Path(), props)
}

func (self *restoreState) enter(props *store.PropertyMap) (err error) {
	// TODO: Do we want to special case the root directory should
	// exist, or should we always restore into a new dir, and
	// require things to be moved out later.
	name := self.Path()
	err = os.Mkdir(name, 0700)
	return
}

func (self *restoreState) leave(props *store.PropertyMap) (err error) {
	return restoreReg(self.Path(), props)
}

// Restore the stats on the given file.
func restoreReg(path string, props *store.PropertyMap) (err error) {
	if isRoot {
		var uid, gid int
		uid, err = props.GetInt("uid")
		if err != nil {
			return
		}

		gid, err = props.GetInt("gid")
		if err != nil {
			return
		}

		err = os.Chown(path, uid, gid)
		if err != nil {
			return
		}
	}

	mode, err := props.GetInt("mode")
	if err != nil {
		return
	}
	err = os.Chmod(path, os.FileMode(mode))
	if err != nil {
		return
	}

	return restoreTime(path, props)
}

// Restore the timestamp on the given node.
func restoreTime(path string, props *store.PropertyMap) (err error) {
	when, err := decodeTimestamp(props.Props["mtime"])
	if err != nil {
		return
	}
	err = os.Chtimes(path, when, when)
	return
}

// Decode a timestamp, which may contain a fractional part.
func decodeTimestamp(text string) (result time.Time, err error) {
	parts := strings.Split(text, ".")

	var sec, nsec int64

	switch len(parts) {
	case 1:
	case 2:
		nsec, err = strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			return
		}

		// Convert time to NS.
		if len(parts[1]) > 9 {
			// TODO: We could just discard the extra
			// digits.
			err = errors.New("Fractional part longer than 9 digits")
			return
		}

		for i := len(parts[1]); i < 9; i++ {
			nsec *= 10
		}
	default:
		err = errors.New(fmt.Sprintf("Invalid timestamp %q", text))
		return
	}

	sec, err = strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return
	}

	result = time.Unix(sec, nsec)
	return
}

func (self *restoreState) blob(chunk pool.Chunk) (err error) {
	_, err = self.file.Write(chunk.Data())
	return
}

func (self *restoreState) Path() string {
	return self.visit.Path(self.base)
}

func (self *restoreState) chunk(chunk pool.Chunk) (err error) {
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
	result = make([]string, 3)

	result[0] = fmt.Sprintf("   %9d chunks", self.chunkCount)
	result[1] = fmt.Sprintf("   %s bytes", meter.Humanize(self.byteCount))
	result[2] = fmt.Sprintf("   %s zbytes (%5.1f%%)", meter.Humanize(self.zbyteCount),
		100.0*float64(self.zbyteCount)/float64(self.byteCount))
	return
}

// Holds whether we are running as root or not.
var isRoot bool

func init() {
	isRoot = os.Geteuid() == 0
}
