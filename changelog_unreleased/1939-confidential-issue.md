[sec] slawek

    Added a verification of the size of incoming requests to fix the DoS
    attack vector. Added a patch for an integer overflow bug in go-pg
    library to prevent any potential vulnerability that could base on it
    as CVE-2024-44905.
    (Gitlab #1939, #1940)
