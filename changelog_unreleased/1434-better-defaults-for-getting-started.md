[func] marcin

    Stork server detects the directory with static UI files relative
    to the stork-server binary when the rest-static-files-dir is
    not specified. It eliminates the need to specify this parameter
    when Stork server is installed in non-standard directory.
    (Gitlab #1434)