[sec] slawek

    The server no longer reveals the correct agent token when the token
    specified in the ping call via REST API is invalid. Previously, this
    endpoint could be used to discover a valid agent token. However,
    the risk was minimal because it required hijacking the server token
    first.
    (Gitlab #1340)
