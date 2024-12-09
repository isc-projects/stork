[bug] marcin

    Enabled timeouts for HTTP client connecting to Kea. It should help
    to gracefully handle communication issues between Stork agents and
    Kea servers.
    (Gitlab #1467)