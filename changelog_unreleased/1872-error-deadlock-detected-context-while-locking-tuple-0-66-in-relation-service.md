[bug] slawek

    Fixed deadlocks in the database while detecting Kea HA services
    including backup servers. They were caused by infinitely
    appending the same backup server ID to one of the database
    tables.
    (Gitlab #1872)
