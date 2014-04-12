package manager

type LVMSnapshot StepData

func (m *LVMSnapshot) Setup() (err error) {
	return simpleRun(
		"lvcreate", "-L", "5g", "-n", m.fs.Volume+".snap",
		"-s", "/dev/"+*m.fs.Vg+"/"+m.fs.Volume)
}

func (m *LVMSnapshot) Name() string { return "lvm-snapshot" }

func (m *LVMSnapshot) Teardown() (err error) {
	return simpleRun(
		"lvremove", "-f", m.snapVol(m.fs))
}
