[bug] marcin

    Fixed parsing "dyndb" BIND 9 configuration statement. Previously,
    BIND 9 configuration parsing failed on this statement causing
    issues with detecting BIND 9 servers.
    (Gitlab #1938)
