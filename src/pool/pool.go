// Storage pool management.

package pool

import (
	"errors"
	"fmt"
	"os"
)

type Pool interface {
	Close() (err error)
	Flush() (err error)
	Insert(chunk Chunk) (err error)
	Contains(oid *OID) (result bool, err error)

	// Scan for the given chunk.  Non-nil error indicates an
	// error.  The 'chunk' will be non-nil if it could be found.
	Search(oid *OID) (chunk Chunk, err error)

	// Return the OID's of all backups written.
	Backups() (backups []*OID, err error)
}

func OpenPool(base string) (pf Pool, err error) {
	fi, err := os.Stat(base + "/data.db")
	if err != nil || !fi.Mode().IsRegular() {
		err = errors.New(fmt.Sprintf("Does not appear to be pool: '%s'", err))
		return
	}

	return OpenSqlPool(base)
}

/*
import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"pdump"
	"strings"
)

// Regenerate the index for the given file.
func reIndexFile(pool string, index string) (err os.Error) {
	pfile, err := os.Open(pool, os.O_RDONLY, 0)
	if err != nil {
		return
	}
	defer pfile.Close()

	log.Printf("Regenering index for %s", pool)

	var ri RamIndex

	var header chunkHeader
	pos := int64(0)
	for {
		opos := pos
		pos, err = readChunkHeader(pfile, pos, &header)
		if err == os.EOF {
			pos = opos
			break
		}
		if err != nil {
			return
		}

		ri.Add(header.oid, uint32(opos))
	}

	err = WriteIndex(&ri, index, uint32(pos))

	return
}

// To start with, just open each pool file.  Eventually, we will
// probably need to only open the ones that get used to avoid running
// out of descriptors.

type PoolFile struct {
	path  string
	index ReadIndexer
	fd    *os.File
}

type FilePool struct {
	files []*PoolFile
}

func openPoolFile(base string, fi *os.FileInfo) (pf *PoolFile, err os.Error) {
	// log.Printf("Pool file %x\n", fi.Size)
	if fi.Size > 0x7FFFFFFF {
		panic("Pool file > 2GB")
	}
	size := uint32(fi.Size)
	poolPath := base + "/" + fi.Name
	indexPath := base + "/" + fi.Name[:len(fi.Name)-5] + ".idx"
	index, err := readFileIndex(indexPath, size)
	if err != nil {
		err = reIndexFile(poolPath, indexPath)
		if err != nil {
			log.Fatalf("Index file doesn't match pool file: %s (%v)", indexPath, err)
		}
		index, err = readFileIndex(indexPath, size)
	}
	if err != nil {
		log.Fatalf("Unable to regenerate index: %s (%v)", indexPath, err)
	}

	pf = &PoolFile{path: poolPath, index: index}
	return
}

func (fp *FilePool) ReadChunk(oid OID) (chunk Chunk, err os.Error) {
	found := false
	var pf *PoolFile
	var offset uint32
	for _, pf = range fp.files {
		var present bool
		offset, present = pf.index.Lookup(oid)
		if present {
			found = true
			break
		}
	}
	if !found {
		err = os.NewError("Chunk not found")
		return
	}

	if pf.fd == nil {
		pf.fd, err = os.Open(pf.path, os.O_RDONLY, 0)
		if err != nil {
			return
		}
	}

	chunk, _, err = ReadChunk(pf.fd, int64(offset))
	return
}

func PoolMain() {
	var pool FilePool

	base := "npool"
	names, err := ioutil.ReadDir(base)
	if err != nil {
		panic(err)
	}
	count := 0
	for _, fi := range names {
		if strings.HasSuffix(fi.Name, ".data") {
			pf, err := openPoolFile(base, fi)
			if err != nil {
				log.Fatalf("Unable to open pool file: %s/%s", base, fi.Name)
			}
			count += pf.index.Len()
			pool.files = append(pool.files, pf)
		}
	}
	log.Printf("%d objects present", count)

	backup, err := ParseOID("2c2a76962b12353f4777517a10d847eff35993be")
	if err != nil {
		log.Fatalf("Invalid OID: %v", err)
	}

	pool.readBackup(backup)
	return

	chunk, err := pool.ReadChunk(backup)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Found chunk '%s'\n", chunk.Kind())
	pdump.Dump(chunk.Data())

	var prop Properties
	if err = xml.Unmarshal(bytes.NewBuffer(chunk.Data()), &prop); err != nil {
		panic(err)
	}
	fmt.Printf("props: %#v\n", prop)
	for key, value := range prop.Map() {
		fmt.Printf("%q=%q\n", key, value)
	}
}

// Read a backup.
func (pool *FilePool) readBackup(oid OID) {
	chunk, err := pool.ReadChunk(oid)
	if err != nil {
		panic(err)
	}
	if string(chunk.Kind()) != "back" {
		log.Fatal("Backup was of improper chunk type")
	}

	var prop Properties
	if err = xml.Unmarshal(bytes.NewBuffer(chunk.Data()), &prop); err != nil {
		panic(err)
	}
	atts := prop.Map()
	root, err := ParseOID(atts["hash"])
	if err != nil {
		panic(err)
	}

	pool.walk(root)
}

func (pool *FilePool) walk(oid OID) {
	chunk, err := pool.ReadChunk(oid)
	if err != nil {
		panic(err)
	}
	log.Printf("root : %s\n", chunk.Kind())
	var node Node
	if err = xml.Unmarshal(bytes.NewBuffer(chunk.Data()), &node); err != nil {
		panic(err)
	}
	// pdump.Dump(chunk.Data())
	fmt.Printf("node\u00b7kind=%q\n", node.Kind)
	for key, value := range node.Map() {
		fmt.Printf("%q=%q\n", key, value)
	}
}

// Properties.
type Properties struct {
	Comment string
	Entry   []Entry
	props   map[string]string
}

func (p *Properties) Map() map[string]string {
	if p.props == nil {
		p.props = decodeEntryList(p.Entry)
	}
	return p.props
}

func decodeEntryList(entries []Entry) map[string]string {
	props := make(map[string]string)
	for i := range entries {
		props[entries[i].Key] = entries[i].Value
	}
	return props
}

type Node struct {
	Kind  string "attr"
	Entry []Entry
	props map[string]string
}

type Entry struct {
	Key   string "attr"
	Value string "innerxml"
}

func (p *Node) Map() map[string]string {
	if p.props == nil {
		p.props = decodeEntryList(p.Entry)
	}
	return p.props
}
*/
