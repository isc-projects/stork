[bug] marcin

    Stork no longer erases Kea configuration parameters it does not
    recognize when it updates Kea configuration. This is important
    when Stork version is behind Kea version, and new parameters
    were introduced to Kea.
    (Gitlab #2412)
