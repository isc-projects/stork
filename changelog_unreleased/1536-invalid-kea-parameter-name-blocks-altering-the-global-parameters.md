[func] marcin

    Conditionally set ddns-use-conflict-resolution and
    ddns-conflict-resolution-mode, depending on the configured
    Kea version. Previously a user could set one of these
    parameters for the Kea versions that did not support them,
    causing configuration errors.
    (Gitlab #1536)