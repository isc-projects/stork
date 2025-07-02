[bug] slawek

    Fixed deadlocks on the HA-related tables in the database. Fixed
    infinite appending the same backup server ID to an array in one
    HA-related table.
    (Gitlab #1872)
