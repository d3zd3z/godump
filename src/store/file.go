package store

import (
	"io"
	"log"
	"os"
	"syscall"

	"pool"
)

// Storing filedata into the store.

func WriteFile(pl pool.Pool, name string) (id *pool.OID, err error) {
	file, err := os.OpenFile(name, os.O_RDONLY|syscall.O_NOATIME, 0)
	if err != nil {
		// Try again, without O_NOATIME, since that is only
		// permissible as either root, or if the file owner
		// matches the current user.
		// TODO: Should we warn about this?
		file, err = os.OpenFile(name, os.O_RDONLY, 0)
	}
	if err != nil {
		return
	}
	defer file.Close()

	ind := NewIndirectWriter(pl, "ind", 256*1024)
	buffer := make([]byte, 256*1024)
	shortCount := 0
	for {
		var n int
		n, err = file.Read(buffer)
		if err == io.EOF {
			err = nil
			break
		}
		if err != nil {
			// TODO: Warn
			return
		}

		if n < len(buffer) {
			shortCount++
			if shortCount > 1 {
				log.Printf("WARN: multiple short reads from %s", name)
			}
		}

		ch := pool.NewChunk("blob", buffer[0:n])
		err = pl.Insert(ch)
		if err != nil {
			return
		}

		err = ind.Add(ch.OID())
		if err != nil {
			return
		}
	}

	return ind.Finalize()
}
