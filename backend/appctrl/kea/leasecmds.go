package keactrl

// Lease type specified in the commands.
type LeaseType string

const (
	LeaseTypeNA LeaseType = "IA_NA"
	LeaseTypePD LeaseType = "IA_PD"
)

const (
	Lease4Get            CommandName = "lease4-get"
	Lease6Get            CommandName = "lease6-get"
	Lease4GetByClientID  CommandName = "lease4-get-by-client-id"
	Lease6GetByDUID      CommandName = "lease6-get-by-duid"
	Lease4GetByHostname  CommandName = "lease4-get-by-hostname"
	Lease6GetByHostname  CommandName = "lease6-get-by-hostname"
	Lease4GetByHWAddress CommandName = "lease4-get-by-hw-address"
	StatLease4Get        CommandName = "stat-lease4-get"
	StatLease6Get        CommandName = "stat-lease6-get"
)

// Creates lease4-get command.
func NewCommandLease4Get(ipAddress string, daemons ...DaemonName) *Command {
	return NewCommandBase(Lease4Get, daemons...).WithArgument("ip-address", ipAddress)
}

// Creates lease6-get command.
func NewCommandLease6Get(leaseType LeaseType, ipAddress string, daemons ...DaemonName) *Command {
	return NewCommandBase(Lease6Get, daemons...).
		WithArgument("type", leaseType).
		WithArgument("ip-address", ipAddress)
}
