[sec] slawek

    Added a verification of the size of incoming requests to fix the DoS
    attack vector. Added a patch securing against an integer overflow
    bug in go-pg library (CVE-2024-44905).  This patch prevents
    potential vulnerabilities that could stem from this bug in the
    future.
    (Gitlab #1939, #1940, #1950)
