package configreview

// The producer verifying if the stat_cmds hooks library is loaded.
func statCmdsPresence(ctx *reviewContext) (*report, error) {
	config := ctx.subjectDaemon.KeaDaemon.Config
	if _, _, present := config.GetHooksLibrary("libdhcp_stat_cmds"); !present {
		r, err := newReport(ctx, "Consider using the libdhcp_stat_cmds to see more detailed statistics for this daemon in Stork.").
			create()
		return r, err
	}
	return nil, nil
}
