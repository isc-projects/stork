[bug] slawek

    Added configurable timeouts for database read/write operations.
    These settings may be useful to avoid the read or write hangs when
    the server looses connectivity to the database
    (Gitlab #1436)
