[bug] marcin

    Root zone is now properly fetched from the agent and stored in the
    database. Previously, fetching the zones from the agent failed
    when the DNS server configuration on the agent's machine included
    a root zone.
    (Gitlab #1870)
