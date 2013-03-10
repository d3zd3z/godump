package pool_test

import "bytes"
import "compress/zlib"
import "crypto/rand"
import "encoding/base64"
import "errors"
import "fmt"
import "io"
import "testing"
import "pool"
import "os"

func TestChunkSanity(t *testing.T) {
	for _, size := range makeSizes() {
		c := pool.MakeRandomChunk(size)

		if c.DataLen() != uint32(size) {
			t.Error("Size mismatch in chunk")
		}

		zd, present := c.ZData()
		if present {
			rd, err := zlib.NewReader(bytes.NewBuffer(zd))
			if err != nil {
				t.Errorf("Error decompressing: '%s'", err)
			}
			var dataBuf bytes.Buffer
			io.Copy(&dataBuf, rd)
			rd.Close()
			data := dataBuf.Bytes()
			if bytes.Compare(data, c.Data()) != 0 {
				t.Error("Mismatch decompressing")
			}
		}
	}
}

func testChunkIO(t *testing.T) {
	for _, size := range makeSizes() {
		c := pool.MakeRandomChunk(size)
		var buf bytes.Buffer
		pool.ChunkWrite(c, &buf)

		c2, pad, err := pool.ChunkRead(&buf)
		if err != nil {
			t.Errorf("Can't read chunk '%s'", err)
		}

		if pad != len(buf.Bytes()) {
			t.Error("Padding is incorrect")
		}

		// Check pad is all zeros.
		for _, b := range buf.Bytes() {
			if b != 0 {
				t.Error("Pad byte is not 0")
			}
		}

		if !bytes.Equal(c.Kind(), c2.Kind()) {
			t.Error("Read kind is incorrect")
		}

		if c.OID().Compare(c2.OID()) != 0 {
			t.Error("OID read is incorrect")
		}

		if !bytes.Equal(c.Data(), c2.Data()) {
			t.Error("Data mismatch")
		}

		if c.DataLen() != c2.DataLen() {
			t.Error("Data length mismatch")
		}
	}
}

func notTestChunkIO(t *testing.T) {
	tmp, err := makeTempDir()
	if err != nil {
		t.Errorf("Unable to make temp dir: '%s'", err)
	}
	defer os.RemoveAll(tmp)
	t.Errorf("tmp = '%s'\n", tmp)
}

// Generate interesting sizes.  Basically, we want the powers of two,
// along with 1 greater and 1 lesser than each of them, with the
// duplicates removed.
func makeSizes() []int {
	sizes := make([]int, 0)
	seen := make(map[int]bool)

	add := func(size int) {
		if !seen[size] {
			seen[size] = true
			sizes = append(sizes, size)
		}
	}

	var base uint
	for base = 0; base < 18; base++ {
		size := 1 << base
		add(size - 1)
		add(size)
		add(size + 1)
	}

	return sizes
}

// Create a fresh tmpdir in /tmp.
func makeTempDir() (name string, err error) {
	// A multiple of 3 for the length prevents padding.
	buf := make([]byte, 15)

	for retry := 0; retry < 5; retry++ {
		var n int
		n, err = io.ReadFull(rand.Reader, buf)
		if n != len(buf) || err != nil {
			if err == nil {
				err = errors.New("Error generating random number")
			}
			return
		}
		name = fmt.Sprintf("%s/test-%s", os.TempDir(),
			base64.URLEncoding.EncodeToString(buf))

		err = os.Mkdir(name, 0755)
		if err == nil {
			// If no error, return success.
			return
		}
	}
	// Too many retries, go ahead and leave the EEXIST.
	return
}
