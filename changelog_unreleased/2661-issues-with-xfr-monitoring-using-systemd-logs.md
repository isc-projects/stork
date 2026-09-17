[bug] marcin

    Fixed DNS zone transfers tracking using systemd logs. Stork agent
    enforces the use of RFC3339-compliant timestamp format in the
    journalctl output that allows for correctly parsing the timestamps
    pertaining to the zone transfers.
    (Gitlab #2661)
