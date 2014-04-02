package meter

import (
	"fmt"
)

// Generate a nice humanly readable version of the size argument.
func Humanize(value int64) string {
	dsize := float64(value)
	pos := 0

	for dsize > 1024.0 {
		dsize /= 1024.0
		pos++
	}

	return fmt.Sprintf("%6.1f%s", dsize, units[pos])
}

var units = []string{
	"B", "KiB", "MiB", "GiB", "TiB", "PiB", "EiB", "ZiB", "YiB",
}
