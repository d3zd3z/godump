// Go's directory reading doesn't return the inode number.  By sorting
// entries by inode number before statting them, we can prevent a lot
// of unnecessary seeking on some filesystems.
package linuxdir

// #include <sys/types.h>
// #include <dirent.h>
import "C"

import (
	"io"
	"os"
	"syscall"
	"unsafe"
)

type Dir C.DIR

func Open(name string) (result *Dir, err error) {
	tmp, err := C.opendir(C.CString(name))
	if err != nil {
		return
	}

	result = (*Dir)(tmp)
	return
}

func (p *Dir) Close() {
	C.closedir((*C.DIR)(p))
}

type Dirent struct {
	Name string
	Ino  uint64
}

// Read directory entries.  err will be set to io.EOF at the end.
func (self *Dir) Readdir() (entry Dirent, err error) {
	var buf C.struct_dirent
	var result *C.struct_dirent
	code := C.readdir_r((*C.DIR)(self), &buf, &result)
	if code != 0 {
		err = os.NewSyscallError("Readdir", syscall.Errno(code))
	}

	if result == nil {
		err = io.EOF
		return
	}

	entry = Dirent{Ino: uint64(buf.d_ino)}

	// Extract the name.
	bytes := (*[10000]byte)(unsafe.Pointer(&buf.d_name[0]))
	entry.Name = string(bytes[0:clen(bytes[:])])
	return
}

func clen(n []byte) int {
	for i := 0; i < len(n); i++ {
		if n[i] == 0 {
			return i
		}
	}
	return len(n)
}

// Convenience wrapper to return all of the entries in a given
// directory.  Skips entries named "." or "..".
func Readdir(name string) (entries []Dirent, err error) {
	entries = make([]Dirent, 0)

	dir, err := Open(name)
	if err != nil {
		return
	}
	defer dir.Close()

	for {
		var entry Dirent
		entry, err = dir.Readdir()
		if err == io.EOF {
			// Don't return an error in this case.
			err = nil
			break
		}
		if err != nil {
			return
		}

		if entry.Name == "." || entry.Name == ".." {
			continue
		}

		entries = append(entries, entry)
	}
	return
}
