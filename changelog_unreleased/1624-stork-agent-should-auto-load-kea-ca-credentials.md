[func] slawek

    Stork agent now retrieves the Basic Auth credentials from the Kea
    CA configuration file. It is no longer necessary to provide the
    JSON file with a login and password to the Kea RestAPI. The agent
    selects credentials with a "stork" user name or prefix.
    If no user is found, it uses the first credentials entry.
    (Gitlab #1624)
