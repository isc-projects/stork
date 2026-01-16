[func] william

    Added agent API for obtaining a snapshot of all leases from Kea.
    It is disabled by default and is not currently ready for production use.  If
    you want to enable it and use it anyway (to test it and report problems),
    pass `--enable-lease-tracking` or set `STORK_AGENT_ENABLE_LEASE_TRACKING=1`
    when starting the agent.
    (Gitlab #2057)
