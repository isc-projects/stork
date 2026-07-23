[bug] slawek

    Fixed a crash that happened when the gRPC connection to a daemon was
    closed while it was still monitored. Skip lease fetching for
    inactive and not monitored daemons.
    (Gitlab #2447)
