package configreview

// The checker verifying if the stat_cmds hooks library is loaded.
func statCmdsPresence(ctx *ReviewContext) (*Report, error) {
	config := ctx.subjectDaemon.KeaDaemon.Config
	if _, _, present := config.GetHooksLibrary("libdhcp_stat_cmds"); !present {
		r, err := NewReport(ctx, "Consider using the libdhcp_stat_cmds to see more detailed statistics for this daemon in Stork.").
			create()
		return r, err
	}
	return nil, nil
}
