package keactrl

import (
	keaconfig "isc.org/stork/daemoncfg/kea"
	"isc.org/stork/datamodel/daemonname"
)

const (
	ConfigGet    CommandName = "config-get"
	ConfigReload CommandName = "config-reload"
	ConfigSet    CommandName = "config-set"
	ConfigWrite  CommandName = "config-write"
	ListCommands CommandName = "list-commands"
	StatusGet    CommandName = "status-get"
	VersionGet   CommandName = "version-get"
)

func NewCommandConfigSet(config *keaconfig.Config, daemonName daemonname.Name) *Command {
	return NewCommandBase(ConfigSet, daemonName).WithArguments(config)
}
