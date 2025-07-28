[bug] william

    Fixed issue where Stork would incorrectly report that lease
    persistence was disabled on a Kea server, when it was actually
    enabled and set to use the default memfile path.
    (Gitlab #667)
