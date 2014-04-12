package manager

type CleanStep struct {
	StepData
}

func (m *CleanStep) Setup() (err error) {
	if m.fs.Clean == nil {
		return
	}

	return m.inDirRun(*m.fs.Clean, m.backupDir())
}

func (m *CleanStep) Teardown() error { return nil }
func (m *CleanStep) Name() string    { return "clean" }
