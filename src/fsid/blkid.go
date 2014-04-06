package fsid

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"unicode"
)

// Block ID management

type Blkid struct {
	loaded bool

	entries []*entry
	byUuid  map[string]*entry
	byDevId map[uint64]*entry
}

type entry struct {
	devName string
	devId   uint64
	fields  map[string]string
}

// Attempt to load the block ID database.  Runs the 'blkid' program
// and captures its output.
func (self *Blkid) Load() (err error) {
	if self.loaded {
		return
	}

	self.entries = make([]*entry, 0)
	self.byUuid = make(map[string]*entry)
	self.byDevId = make(map[uint64]*entry)

	cmd := exec.Command("blkid")
	outp, err := cmd.StdoutPipe()
	if err != nil {
		return
	}

	err = cmd.Start()
	if err != nil {
		return
	}

	reader := bufio.NewReader(outp)

	for {
		var line string
		line, err = reader.ReadString('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			return
		}

		err = self.addLine(line)
		if err != nil {
			return
		}
	}

	err = cmd.Wait()
	if err != nil {
		return
	}

	self.loaded = true

	return
}

// Look up a single entry in the blockID database, if present.
// Returns the UUID, if it is found.  Sets 'ok' to true if the id was
// present.
func (self *Blkid) ByDevId(devId uint64) (uuid string, ok bool) {
	ent, ok := self.byDevId[devId]
	if !ok {
		return
	}
	uuid, ok = ent.fields["UUID"]
	return
}

// Parse a single line of output from blkid.
// The expected form is:
//  devicename ':' SPACE
//  {  KEY '=' quoted-string SPACE  }
// note that the last key/value pair is followed by a space.
func (self *Blkid) addLine(line string) (err error) {
	var ent entry

	ent.fields = make(map[string]string)

	pos := strings.Index(line, ": ")
	if pos < 0 {
		msg := fmt.Sprintf("Blkid output has no device name: %q", line)
		err = errors.New(msg)
		return
	}

	ent.devName = line[:pos]
	line = line[pos+2:]

	fi, err := os.Stat(ent.devName)
	if err != nil {
		return
	}
	ent.devId = fi.Sys().(*syscall.Stat_t).Rdev

	for len(line) > 0 {
		// Stop when we reach the newline at the end.
		if line[0] == '\n' {
			break
		}

		// Grab the key.
		a := strings.IndexFunc(line, notID)
		if a < 0 {
			msg := fmt.Sprintf("blkid: Invalid identifier: %q", line)
			err = errors.New(msg)
			return
		}
		if line[a] != '=' {
			msg := fmt.Sprintf("blkid: Expecting '\n': %q", line)
			err = errors.New(msg)
			return
		}
		key := line[:a]
		line = line[a+1:]

		// The value should be a quoted string.
		if line[0] != '"' {
			msg := fmt.Sprintf("blkid: Expecting '\"': %q", line)
			err = errors.New(msg)
			return
		}
		line = line[1:]
		b := strings.Index(line, "\" ")
		if b < 0 {
			msg := fmt.Sprintf("blkid: Expecting end of quoted string: %q", line)
			err = errors.New(msg)
			return
		}
		value := line[:b]
		line = line[b+2:]
		ent.fields[key] = value
	}

	self.entries = append(self.entries, &ent)

	uuid, ok := ent.fields["UUID"]
	if ok {
		self.byUuid[uuid] = &ent
	}

	self.byDevId[ent.devId] = &ent

	return
}

// Negation of characters used in the blkid identifiers.
func notID(r rune) bool {
	return !(unicode.IsUpper(r) || r == '_')
}
