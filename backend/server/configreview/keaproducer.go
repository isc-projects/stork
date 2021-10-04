package configreview

func statCmdsPresence(ctx *reviewContext) *report {
	return &report{
		issue: "stat_cmds hooks library must be used to display DHCP statistics",
	}
}
