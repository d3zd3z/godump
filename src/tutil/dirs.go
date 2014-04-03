package tutil

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"testing"
)

type TempDir struct {
	path string
	t    *testing.T
}

func NewTempDir(t *testing.T) (tdir *TempDir) {
	// base := os.TempDir()
	base := "/var/tmp/test-%s" // Use because /tmp is usually a ram fs.

	for count := 1; count < 5; count++ {
		dir := genName(base)
		err := os.Mkdir(dir, 0755)
		if err == nil {
			return &TempDir{path: dir, t: t}
		}
		fmt.Printf("err: %s\n", err)
		if os.IsExist(err) {
			continue
		}
		t.Fatalf("Error creating directory: %s", err)
	}
	t.Fatalf("Unable to create test tmpdir: %s")
	return
}

func (p *TempDir) Clean() {
	err := os.RemoveAll(p.path)
	if err != nil {
		p.t.Fatalf("Unable to clean tmpdir (%q): %s", p.path, err)
	}
}

func (p *TempDir) Path() string {
	return p.path
}

func genName(base string) string {
	buf := make([]byte, 9)
	n, err := rand.Read(buf)
	if err != nil {
		panic("Random error")
	}
	if n != len(buf) {
		panic("Short read")
	}

	return fmt.Sprintf(base, base64.URLEncoding.EncodeToString(buf))
}
