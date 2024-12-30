package keactrl

import keaconfig "isc.org/stork/appcfg/kea"

const (
	ConfigGet       CommandName = "config-get"
	ConfigReload    CommandName = "config-reload"
	ConfigSet       CommandName = "config-set"
	ConfigWrite     CommandName = "config-write"
	ListCommands    CommandName = "list-commands"
	StatisticGet    CommandName = "statistic-get"
	StatisticGetAll CommandName = "statistic-get-all"
	StatusGet       CommandName = "status-get"
	VersionGet      CommandName = "version-get"
)

func NewCommandConfigSet(config *keaconfig.Config, daemonNames ...DaemonName) *Command {
	return NewCommandBase(ConfigSet, daemonNames...).WithArguments(config)
}
