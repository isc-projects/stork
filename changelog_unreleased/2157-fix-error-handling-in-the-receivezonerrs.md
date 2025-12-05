[bug] marcin

    Fixed error handling in two gRPC streaming calls, one for
    receiving zone contents from the agent, and another one
    for receiving BIND 9 configuration. The errors are now
    correctly interpreted in the Stork server.
    (Gitlab #2157)
