[sec] slawek

    Added a protection against CVE-2024-44905 - an SQL injection
    vulnerable existing in the go-pg library. Added a verification of
    the size of incoming requests to fix the DoS attack vector.
    (Gitlab #1939, #1940)
