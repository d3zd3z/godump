// Storage pool management.

package main

import (
	"io/ioutil"
	"log"
	"os"
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

type PoolFile struct {
	index ReadIndexer
}

func openPoolFile(base string, fi *os.FileInfo) (pf *PoolFile, err os.Error) {
	// log.Printf("Pool file %x\n", fi.Size)
	if fi.Size > 0x7FFFFFFF {
		panic("Pool file > 2GB")
	}
	size := uint32(fi.Size)
	indexPath := base + "/" + fi.Name[:len(fi.Name)-5] + ".idx"
	index, err := readFileIndex(indexPath, size)
	if err != nil {
		err = reIndexFile(base+"/"+fi.Name, indexPath)
		if err != nil {
			log.Fatalf("Index file doesn't match pool file: %s (%v)", indexPath, err)
		}
		index, err = readFileIndex(indexPath, size)
	}
	if err != nil {
		log.Fatalf("Unable to regenerate index: %s (%v)", indexPath, err)
	}

	pf = &PoolFile{index}
	return
}

func poolMain() {
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
		}
	}
	log.Printf("%d objects present", count)
}
