[func] marcin

    Optimized detection of the BIND 9 and PowerDNS servers in the Stork
    agent. It avoids parsing configuration files if they haven't changed
    since the last detection.
    (Gitlab #2200)
