[func] marcin

    Implemented log tracker in the Stork agent. It will be used to
    monitor BIND 9 logs to capture the zone transfer events. However,
    as a generic solution, it can be used in the future for tracking
    any kind of events logged in the files or systemd logs.
    (Gitlab #2391)
