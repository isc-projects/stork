package agent

import (
	"github.com/pkg/errors"
	"github.com/shirou/gopsutil/v4/process"
)

// Process interface. Using interface allows to mock the processes.
type Process interface {
	GetPid() int32
	GetName() (string, error)
	GetCmdline() (string, error)
	GetCwd() (string, error)
}

// Wrapper for gopsutil process.
type processWrapper struct {
	process *process.Process
}

var _ Process = (*processWrapper)(nil)

// Returns the process pid.
func (p *processWrapper) GetPid() int32 {
	return p.process.Pid
}

// Returns the process name.
func (p *processWrapper) GetName() (string, error) {
	name, err := p.process.Name()
	err = errors.Wrap(err, "failed to get process name")
	return name, err
}

// Returns the process command line.
func (p *processWrapper) GetCmdline() (string, error) {
	cmdline, err := p.process.Cmdline()
	err = errors.Wrap(err, "failed to get process command line")
	return cmdline, err
}

// Returns the process current working directory.
func (p *processWrapper) GetCwd() (string, error) {
	cwd, err := p.process.Cwd()
	err = errors.Wrap(err, "failed to get process current working directory")
	return cwd, err
}

// Interface to operate on processes.
type ProcessManager interface {
	ListProcesses() ([]Process, error)
}

// Implementation of the ProcessManager interface using gopsutil.
type processManagerWrapper struct{}

var _ ProcessManager = (*processManagerWrapper)(nil)

// Returns the list of processes.
func (plw *processManagerWrapper) ListProcesses() ([]Process, error) {
	processes, err := process.Processes()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get processes")
	}

	result := make([]Process, 0, len(processes))
	for _, p := range processes {
		result = append(result, &processWrapper{process: p})
	}
	return result, nil
}

// Returns the ProcessManager instance.
func NewProcessManager() ProcessManager {
	return &processManagerWrapper{}
}
