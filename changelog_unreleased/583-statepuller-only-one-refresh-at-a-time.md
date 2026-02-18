[bug] slawek

    Fixed a hole that allowed pulling the same agent state many times
    concurrently, which could result in duplicating daemons.
    (Gitlab #583)
