[bug] slawek, marcin, andrei

    Prevented Stork tool from auto-discovering migration files. This
    feature has never been necessary, but it could raise an error if the
    Stork user can't access the searched directory.
    (Gitlab #1439)
