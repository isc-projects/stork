[func] marcin

    Implemented zone transfers monitoring in the agent. The agent
    now collects the information about ongoing and completed zone
    transfers, and holds it in memory. Since there is no gRPC API
    to retrieve the zone transfer information from the agent, the
    information is not yet available to the server. The API and the
    necessary Stork server updates will be implemented in the future
    GL issues.
    (Gitlab #2393)
