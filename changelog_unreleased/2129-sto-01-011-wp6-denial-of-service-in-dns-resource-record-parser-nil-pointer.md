[bug] marcin

    Addressed an issue in the Stork server whereby the server could
    panic as a result of receiving and parsing an empty DNS RR
    from the monitored DNS server over AXFR.
    (Gitlab #2129)
