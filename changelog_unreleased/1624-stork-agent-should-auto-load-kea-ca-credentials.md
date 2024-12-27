[func] slawek

    Stork agent now retrieves the Basic Auth credentials from the Kea
    CA configuration file. It is no longer necessary to provide the
    JSON file with a login and password to the Kea RestAPI. The agent
    selects credentials with a user named "stork" or prefixed with
    "stork". If no user is found, it takes the first available entry.
    (Gitlab #1624)
