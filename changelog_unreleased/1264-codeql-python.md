[build] tomek

    Fixed several smaller Python issues reported by CodeQL and other Python linters. The
    `gen_kea_config.py` tool used in demo now takes an optional `--seed` parameter that,
    if specified, will initiate the PRNG to given value. This allows to use repeat
    randomized test runs, if necessary.
    (Gitlab #1264)
