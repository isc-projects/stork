* [sec] slawek

    Disabled the TLS 1.0 and 1.1 protocols in the GRPC server of the Stork
    agent. The Stork server communicates with the Stork agent over TLS 1.3 by
    default.
    (Gitlab #1197)
