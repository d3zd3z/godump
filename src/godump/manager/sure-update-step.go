package manager

type SureUpdateStep struct {
	StepData
}

func (m *SureUpdateStep) Setup() (err error) {
	cmd, err := m.Command("gosure")
	if err != nil {
		return
	}
	return m.inDirRun(cmd, "update")
}

func (m *SureUpdateStep) Teardown() error { return nil }
func (m *SureUpdateStep) Name() string    { return "sure-update" }

// Write the data back, at least if we're LVM.
type SureWriteStep struct {
	StepData
}

func (m *SureWriteStep) Setup() (err error) {
	cmd, err := m.Command("gosure")
	if err != nil {
		return
	}

	m.banner(m.sureLog, "sure")
	err = m.inDirToRun(m.sureLog, cmd, "signoff")
	if err != nil {
		return
	}

	if m.fs.Vg == nil {
		return
	}

	// Copy the surefile back.
	return simpleRun("echo", "--",
		"-p",
		m.snapDest(m.fs)+"/2sure.dat.gz",
		m.fs.Base+"/2sure.dat.gz")
}

func (m *SureWriteStep) Teardown() error { return nil }
func (m *SureWriteStep) Name() string    { return "sure-write" }
