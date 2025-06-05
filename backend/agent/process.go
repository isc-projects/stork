package agent

import (
	"slices"

	"github.com/pkg/errors"
	"github.com/shirou/gopsutil/v4/process"
)

var (
	_ processLister    = (*processListerImpl)(nil)
	_ supportedProcess = (*processWrapper)(nil)
)

// An interface to a process detected by the agent. Using the interface
// allows for mocking listing the processes in the unit tests.
type supportedProcess interface {
	getCmdline() (string, error)
	getCwd() (string, error)
	getName() (string, error)
	getPid() int32
	getParentPid() (int32, error)
}

// Wrapper for gopsutil process. It implements the supportedProcess interface.
type processWrapper struct {
	process *process.Process
}

// Returns the process command line.
func (p *processWrapper) getCmdline() (string, error) {
	cmdline, err := p.process.Cmdline()
	err = errors.Wrapf(err, "failed to get process command line for pid %d", p.getPid())
	return cmdline, err
}

// Returns the process current working directory.
func (p *processWrapper) getCwd() (string, error) {
	cwd, err := p.process.Cwd()
	err = errors.Wrapf(err, "failed to get process current working directory for pid %d", p.getPid())
	return cwd, err
}

// Returns the process pid.
func (p *processWrapper) getPid() int32 {
	return p.process.Pid
}

// Returns the parent pid of the parent process.
func (p *processWrapper) getParentPid() (int32, error) {
	ppid, err := p.process.Ppid()
	err = errors.Wrapf(err, "failed to get process parent pid for pid %d", p.getPid())
	return ppid, err
}

// Returns the process name.
func (p *processWrapper) getName() (string, error) {
	name, err := p.process.Name()
	err = errors.Wrapf(err, "failed to get process name for pid %d", p.getPid())
	return name, err
}

// Convenience function checking if the detected process is supported by the agent.
func isSupportedProcess(p *process.Process) bool {
	name, _ := p.Name()
	return name == keaProcName || name == namedProcName
}

// An interface for listing the supported processes. It can be mocked in the
// unit tests.
type processLister interface {
	listProcesses() ([]supportedProcess, error)
}

// A default implementation of the processLister interface.
type processListerImpl struct{}

// Lists the supported processes using gopsutil library. It returns only the
// processes supported by the agent (apps that can be monitored by the agent).
func (impl *processListerImpl) listProcesses() ([]supportedProcess, error) {
	processes, err := process.Processes()
	if err != nil {
		return nil, errors.Wrap(err, "failed to list running processes")
	}
	var listedProcesses []supportedProcess
	for _, p := range processes {
		if isSupportedProcess(p) {
			listedProcesses = append(listedProcesses, &processWrapper{process: p})
		}
	}
	return listedProcesses, nil
}

// An instance listing the supported processes and filtering out their child
// processes. Some DNS servers (e.g., NSD) spawn many child processes. The
// agent must not treat child processes as distinct apps. Therefore, the
// manager only selects top-level processes, removing the ones having parent
// PID matching a PID of another process.
type ProcessManager struct {
	lister processLister
}

// Lists supported processes and filters out their child processes.
func (pm *ProcessManager) ListProcesses() ([]supportedProcess, error) {
	processes, err := pm.lister.listProcesses()
	if err != nil {
		return nil, err
	}
	var acceptedCandidates []supportedProcess
	for _, candidate := range processes {
		// For each candidate process, check if the parent PID matches a
		// PID of another process. If it does, the candidate is a child process
		// and is not added to the list of accepted candidates.
		ppid, err := candidate.getParentPid()
		if err != nil {
			continue
		}
		if !slices.ContainsFunc(processes, func(p supportedProcess) bool {
			return candidate.getPid() != p.getPid() && ppid == p.getPid()
		}) {
			acceptedCandidates = append(acceptedCandidates, candidate)
		}
	}
	return acceptedCandidates, nil
}

// Returns the ProcessManager instance.
func NewProcessManager() *ProcessManager {
	return &ProcessManager{
		lister: &processListerImpl{},
	}
}
