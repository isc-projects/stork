[func] marcin

    Stork agent parses BIND 9 logging configuration to determine
    the log files where zone transfer events are logged. The zone
    transfers are tracked in these files when zone transfer tracking
    is enabled using the --enable-xfr-tracking parameter. With this
    mechanism it is no longer necessary to specify log locations
    using the --xfr-in-tracking-path and --xfr-out-tracking-path
    parameters. These parameters can still be used to override the
    locations determined from the BIND 9 configuration, if necessary.
    (Gitlab #2444)
