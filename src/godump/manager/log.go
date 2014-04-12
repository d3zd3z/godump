package manager

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

func openLog(name string) (file *os.File, err error) {
	// Try renaming the log to the backup log
	err = os.Rename(name, name+".bak")
	if err != nil {
		log.Printf("INFO: Renaming %s to .bak %q", name, err)
	}

	return os.Create(name)
}

func (m *StepData) banner(logfile *os.File, kind string) (err error) {
	msg := fmt.Sprintf("--- %s of %s (%s) on %s ---",
		kind, m.fs.Volume, m.snapDest(m.fs), time.Now())
	line := strings.Repeat("-", len(msg))

	_, err = fmt.Fprintln(logfile, line)
	if err != nil {
		return
	}

	_, err = fmt.Fprintln(logfile, msg)
	if err != nil {
		return
	}

	_, err = fmt.Fprintln(logfile, line)
	if err != nil {
		return
	}
	return
}
