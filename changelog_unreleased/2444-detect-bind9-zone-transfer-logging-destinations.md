[func] marcin

    Stork agent parses BIND 9 logging configuration to determine
    the log files where zone transfer events are logged. The zone
    transfers are tracked in these files when zone transfer tracking
    is enabled using the --enable-xfr-tracking parameter. Still, it
    is possible to specify alternate file locations by explicitly
    setting the --xfr-in-tracking-path and --xfr-out-tracking-path
    command line arguments of the Stork agent.
    (Gitlab #2444)
