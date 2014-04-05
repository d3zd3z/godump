package dump

import (
	"log"
	"os"
	"path"
	"sort"

	"linuxdir"
)

// Reading a directory, and getting the stat information for all of
// the nodes.

func Readdir(dirName string) (entries []os.FileInfo, err error) {
	base, err := linuxdir.Readdir(dirName)
	if err != nil {
		return
	}

	sort.Sort(byInode(base))

	entries = make([]os.FileInfo, 0, len(base))

	for _, info := range base {
		name := path.Join(dirName, info.Name)
		var fi os.FileInfo
		fi, err = os.Lstat(name)
		if err != nil {
			// Skip the entry, and warn.
			log.Printf("WARN: Unable to stat: %s (%s)", name, err)
			continue
		}

		entries = append(entries, fi)
	}

	// After reading, sort all of the entries by name.
	sort.Sort(byName(entries))

	return
}

type byInode []linuxdir.Dirent

func (a byInode) Len() int           { return len(a) }
func (a byInode) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byInode) Less(i, j int) bool { return a[i].Ino < a[j].Ino }

type byName []os.FileInfo

func (a byName) Len() int           { return len(a) }
func (a byName) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byName) Less(i, j int) bool { return a[i].Name() < a[j].Name() }
