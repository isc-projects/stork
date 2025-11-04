[bug] marcin

    Fix listing DNS servers in the traffic simulator. Previously,
    BIND 9 servers were not listed when other apps were authorized.
    Now, both BIND 9 and PowerDNS servers are correctly listed in
    the simulator.
    (Gitlab #2081)
