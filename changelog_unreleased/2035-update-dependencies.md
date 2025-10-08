[sec] ! slawek

    Updated Go to version 1.24.8.
    Changed how the Stork server generates GRPC certificates for
    compatibility with the new revision of the cryptographic library.
    Stork now filters out the DNS names that are not valid domains when
    preparing the Subject Alternate Name (SAN) certificate field. All
    previously generated certificates are incompatible if they contain
    invalid DNS names that must be fixed by re-registering the Stork
    agents.
    (Gitlab #2035)
