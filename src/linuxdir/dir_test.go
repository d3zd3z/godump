package linuxdir_test

import (
	"fmt"
	"os"
	"sort"
	"syscall"
	"testing"

	"linuxdir"
)

// Read using the linuxdir and with the Go builtin, and make sure that
// the inode numbers match.  Use a stable directory, such as
// "/usr/bin" that will be reasonably large, but also unlikely to
// change during the test.
func TestDir(t *testing.T) {
	linuxy, err := linuxdir.Readdir("/usr/bin")
	if err != nil {
		t.Errorf("Error reading /usr/bin: %q", err)
	}

	fd, err := os.Open("/usr/bin")
	if err != nil {
		t.Errorf("Error opening /usr/bin as dir: %q", err)
	}
	gos, err := fd.Readdir(-1)
	fd.Close()
	if err != nil {
		t.Errorf("Error reading from /usr/bin/: %q", err)
	}

	if len(linuxy) != len(gos) {
		t.Errorf("Read different count from 'os' and 'linuxdir'")
	}

	sort.Sort(LinuxyName(linuxy))
	sort.Sort(GosName(gos))

	// Compare them to make sure we got the same names, and same
	// inod numbers.
	for i := range linuxy {
		if linuxy[i].Name != gos[i].Name() {
			t.Errorf("Name mismatch")
		}

		if linuxy[i].Ino != gos[i].Sys().(*syscall.Stat_t).Ino {
			t.Errorf("Inode mismatch in: %s", linuxy[i].Name)
		}
	}

	fmt.Printf("linuxy: %d, gos: %d\n", len(linuxy), len(gos))
}

type LinuxyName []linuxdir.Dirent

func (a LinuxyName) Len() int           { return len(a) }
func (a LinuxyName) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a LinuxyName) Less(i, j int) bool { return a[i].Name < a[j].Name }

type GosName []os.FileInfo

func (a GosName) Len() int           { return len(a) }
func (a GosName) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a GosName) Less(i, j int) bool { return a[i].Name() < a[j].Name() }
