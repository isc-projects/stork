[bug] slawek

    Fixed a bug that could cause the server to panic during user
    authentication via LDAP. Authentication could be rejected if the
    user's Distinguished Name differed from the value known to the Stork
    server.
    (Gitlab #2303)
