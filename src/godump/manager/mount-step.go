package manager

type Mount struct {
	StepData
}

func (m *Mount) Setup() (err error) {
	return simpleRun(
		"mount", m.snapVol(m.fs), m.snapDest(m.fs))
}

func (m *Mount) Teardown() (err error) {
	return simpleRun(
		"umount", m.snapDest(m.fs))
}

func (m *Mount) Name() string { return "mount-snapshot" }
