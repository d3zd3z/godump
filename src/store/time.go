package store

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Decode a timestamp, which may contain a fractional part.
func DecodeTimestamp(text string) (result time.Time, err error) {
	parts := strings.Split(text, ".")

	var sec, nsec int64

	switch len(parts) {
	case 1:
	case 2:
		nsec, err = strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			return
		}

		// Convert time to NS.
		if len(parts[1]) > 9 {
			// TODO: We could just discard the extra
			// digits.
			err = errors.New("Fractional part longer than 9 digits")
			return
		}

		for i := len(parts[1]); i < 9; i++ {
			nsec *= 10
		}
	default:
		err = errors.New(fmt.Sprintf("Invalid timestamp %q", text))
		return
	}

	sec, err = strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return
	}

	result = time.Unix(sec, nsec)
	return
}
