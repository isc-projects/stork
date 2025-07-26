[bug] marcin

    The stork-agent gracefully deals with the situation when the
    statistics-channel configuration in BIND 9 lacks the allow
    statement. It logs an error requesting that the allow statement
    is included in the configuration.
    (Gitlab #1925)
