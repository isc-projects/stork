[sec] slawek

    Improved the security of the agent's GRPC certificate generation.
    The server now ignores IP addresses and DNS names provided in the
    Subject Alternative Name (SAN) fields of the Certificate Signing
    Request (CSR). Instead, they are assigned on the server side
    based on the agent's address. It blocks the attacker from requesting
    a certificate for any IP address or DNS names under the guise of
    registering a legitimate agent.
    (Gitlab #2564)
