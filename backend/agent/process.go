package agent

import (
	"slices"

	"github.com/pkg/errors"
	"github.com/shirou/gopsutil/v4/process"
	"isc.org/stork/datamodel/daemonname"
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
	getExe() (string, error)
	getName() (string, error)
	getPid() int32
	getParentPid() (int32, error)
	getDaemonName() daemonname.Name
}

// Wrapper for gopsutil process. It implements the supportedProcess interface.
type processWrapper struct {
	process    *process.Process
	daemonName daemonname.Name
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

// Returns the process path to the executable.
func (p *processWrapper) getExe() (string, error) {
	exe, err := p.process.Exe()
	err = errors.Wrapf(err, "failed to get process executable path for pid %d", p.getPid())
	return exe, err
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

// Converts a process name to a daemon name. If the process name
// is not recognized, it returns an empty string.
func (p *processWrapper) getDaemonName() daemonname.Name {
	return p.daemonName
}

// An interface for listing the supported processes. It can be mocked in the
// unit tests.
type processLister interface {
	listProcesses() ([]supportedProcess, error)
}

// A default implementation of the processLister interface.
type processListerImpl struct {
	// Mapping between process name and its daemon name.
	supportedProcesses map[string]daemonname.Name
}

// Lists the supported processes using gopsutil library.
func (impl *processListerImpl) listProcesses() ([]supportedProcess, error) {
	processes, err := process.Processes()
	if err != nil {
		return nil, errors.Wrap(err, "failed to list running processes")
	}
	var listedProcesses []supportedProcess
	for _, p := range processes {
		name, err := p.Name()
		if err != nil {
			// No permission to get the process name.
			continue
		}
		daemonName, isSupported := impl.supportedProcesses[name]
		if !isSupported {
			continue
		}

		listedProcesses = append(listedProcesses, &processWrapper{
			process: p, daemonName: daemonName,
		})
	}
	return listedProcesses, nil
}

// An instance listing the supported processes and filtering out their child
// processes. Some DNS servers (e.g., NSD) spawn many child processes. The
// agent must not treat child processes as distinct daemons. Therefore, the
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
		lister: &processListerImpl{
			supportedProcesses: map[string]daemonname.Name{
				"kea-dhcp4":      daemonname.DHCPv4,
				"kea-dhcp6":      daemonname.DHCPv6,
				"kea-d2":         daemonname.D2,
				"kea-ctrl-agent": daemonname.CA,
				"named":          daemonname.Bind9,
				"pdns_server":    daemonname.PDNS,
			},
		},
	}
}
