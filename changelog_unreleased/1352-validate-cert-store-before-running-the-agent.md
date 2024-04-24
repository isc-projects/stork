[func] slawek

    Added validation of the existing GRPC certificates before running
    the agent. This prevents the agent from starting if it is not able
    to establish a connection to the server.
    (Gitlab #1352)
