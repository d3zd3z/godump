package manager

import (
	"godump/dump"
)

// Perform the backup to the pool.
type DumpStep struct {
	StepData
}

func (m *DumpStep) Setup() (err error) {
	props := make(map[string]string)

	props["fs"] = m.fs.Volume
	return dump.Run(m.pool, m.backupDir(), props)
}

func (m *DumpStep) Teardown() error { return nil }
func (m *DumpStep) Name() string    { return "dump" }
