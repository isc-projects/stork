[bug] slawek

    Decreased the logging level for a message produced when it is unable
    to find a subnet reported in the Kea statistics response to prevent
    bloating the logs if the stale subnets are included.
    (Gitlab #2382)
