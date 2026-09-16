[bug] wlipinski

    Fixed a bug where a DHCP option defined as a "record" with the
    array flag set (e.g. slp-directory-agent, a mandatory flag
    followed by one or more IPv4 addresses) misidentified any CSV
    value beyond the record's fixed fields as belonging to the
    record's first field type again, instead of repeating the
    record's last field type. This caused the whole subnet
    containing such an option to be silently dropped during a
    periodic Kea configuration pull.
    (Gitlab #2644)
