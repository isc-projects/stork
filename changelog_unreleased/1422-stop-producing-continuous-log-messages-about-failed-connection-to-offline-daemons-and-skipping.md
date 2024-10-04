[func] slawek

    Refactored the application detection loop in the Stork agent to
    prevent continuous logging of the same detection errors. Removed the
    log message about successfully finding the BIND 9 configuration.
    Stork agent no longer logs the RNDC key status for BIND 9 statistics
    channel.
    (Gitlab #1422, #1388, #1384)
