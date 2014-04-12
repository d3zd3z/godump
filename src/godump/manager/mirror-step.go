package manager

// Mirror the data using rsync
type MirrorStep struct {
	StepData
}

func (m *MirrorStep) Setup() (err error) {
	m.banner(m.rsyncLog, "rsync")

	return m.inDirToRun(
		m.rsyncLog,
		"rsync", "-aiHX", "--delete",
		m.backupDir()+"/",
		*m.host.Mirror+"/"+m.fs.Volume)
}

func (m *MirrorStep) Teardown() (err error) { return nil }
func (m *MirrorStep) Name() string          { return "rsync" }
