[bug] slawek

    Fixed Kea, BIND 9, and PowerDNS detection failing to parse the CLI
    flags when the path to the binary contained a directory named same
    as the binary (e.g., /var/lib/named/sbin/named).
    (Gitlab #2374)
