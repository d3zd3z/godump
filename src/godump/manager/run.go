package manager

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"

	"godump/config"
	"pool"
)

// TODO: Pairing of ops and undo.
// TODO: names and such for the various parts.

func Run(conf *config.Config, args []string) (err error) {
	if len(args) != 1 {
		err = errors.New("Usage: godump managed hostname")
		return
	}

	host := args[0]
	err = validateConfig(conf, host)
	return
}

// The overall sequence
var sequence = []string{
	"lvm-snapshot",
	"mount-snapshot",
	"clean",
	"sure-update",
	"sure-write",
	"rsync",
	"dump"}

// Ensure that everything described in the config file makes sense.
func validateConfig(conf *config.Config, host string) (err error) {
	hinfo, ok := conf.Hosts[host]
	if !ok {
		err = fmt.Errorf("Unknown host %q (not in config file)", host)
		return
	}

	mgr := Manager{conf: conf, host: hinfo}
	// err = mgr.CheckPlainPaths()

	mgr.pool, err = pool.OpenPool(conf.Defaults.Pool)
	if err != nil {
		return
	}
	defer mgr.pool.Close()

	allSteps := make([]Steps, 0)
	for _, fs := range hinfo.Fs {
		steps := make(Steps)
		sd := StepData{Manager: &mgr, fs: fs}
		switch fs.Style {
		case "ext4-lvm":
			steps.Add(&LVMSnapshot{Manager: &mgr, fs: fs})
			steps.Add(&Mount{StepData: sd})
			steps.Add(&CleanStep{StepData: sd})

		case "plain":

		default:
			err = fmt.Errorf("Unsupported fs style: %q", fs.Style)
			return
		}

		steps.Add(&SureUpdateStep{StepData: sd})
		steps.Add(&SureWriteStep{StepData: sd})

		if mgr.host.Mirror != nil {
			steps.Add(&MirrorStep{StepData: sd})
		}

		steps.Add(&DumpStep{StepData: sd})

		allSteps = append(allSteps, steps)
	}

	// Open the log files.
	if mgr.conf.Defaults.Surelog != nil {
		mgr.sureLog, err = openLog(*mgr.conf.Defaults.Surelog)
		if err != nil {
			return
		}
		defer mgr.sureLog.Close()
	} else {
		mgr.sureLog = os.Stdout
	}

	if mgr.conf.Defaults.Rsynclog != nil {
		mgr.rsyncLog, err = openLog(*mgr.conf.Defaults.Rsynclog)
		if err != nil {
			return
		}
		defer mgr.rsyncLog.Close()
	} else {
		mgr.rsyncLog = os.Stdout
	}

	// All of the successfully performed steps, will be undone
	// when we're finished.
	performed := make([]Step, 0)

Outer:
	for _, name := range sequence {
		for _, steps := range allSteps {
			step, ok := steps[name]
			if ok {
				err = step.Setup()
				if err != nil {
					log.Printf("WARN: %s", err)
					break Outer
				}

				performed = append(performed, step)
			}
		}
	}

	// Undo the steps, warning about any errors, but otherwise
	// ignoring them.
	for len(performed) > 0 {
		step := performed[len(performed)-1]
		performed = performed[:len(performed)-1]

		err2 := step.Teardown()
		if err2 != nil {
			log.Printf("WARN: %s", err2)
		}
	}

	return
}

// A set of steps, given by name.
type Steps map[string]Step

func (s Steps) Add(item Step) {
	s[item.Name()] = item
}

type Step interface {
	// The actions themselves.
	Setup() (err error)
	Teardown() (err error)

	// The name of this step, used to sort, and describe.
	Name() string
}

type StepData struct {
	*Manager
	fs *config.FileSystem
}

type Manager struct {
	conf *config.Config
	host *config.Host
	pool pool.Pool

	sureLog  *os.File
	rsyncLog *os.File
}

func (m *Manager) CheckPlainPaths() (err error) {
	return
}

func (m *Manager) snapVol(fs *config.FileSystem) string {
	return "/dev/" + *fs.Vg + "/" + fs.Volume + ".snap"
}

func (m *Manager) snapDest(fs *config.FileSystem) string {
	return "/mnt/snap/" + fs.Volume
}

func (m *Manager) Command(name string) (cmd string, err error) {
	cmd, ok := m.conf.Commands[name]
	if !ok {
		err = fmt.Errorf("Command %q not in config file", cmd)
	}
	return
}

func simpleRun(name string, arg ...string) error {
	log.Printf("Run command: %s %v", name, arg)
	cmd := exec.Command(name, arg...)
	// TODO: Capture the command output for the log?
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Return the directory where the snapshot or backup reside.
func (m *StepData) backupDir() string {
	if m.fs.Vg != nil {
		return m.snapDest(m.fs)
	} else {
		return m.fs.Base
	}
}

// Run the command in the directory to be backed up.
func (m *StepData) inDirRun(name string, arg ...string) error {
	// TODO: Capture in the log, rather than just sending to
	// stdout.
	return m.inDirToRun(os.Stdout, name, arg...)
}

// Run outputting to directory.
// TODO: Consolidate these better.
func (m *StepData) inDirToRun(out io.Writer, name string, arg ...string) error {
	dest := m.backupDir()
	log.Printf("Run command (%s): %s %v", dest, name, arg)
	cmd := exec.Command(name, arg...)
	cmd.Dir = dest
	cmd.Stdout = out
	cmd.Stderr = out
	return cmd.Run()
}
