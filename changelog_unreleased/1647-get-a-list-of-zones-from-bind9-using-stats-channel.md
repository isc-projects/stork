[func] marcin

    Implemented a DNS "zone inventory" in the Stork agent. It adds
    the capability for the Stork agent to gather a list of views
    and zones from BIND 9 on startup. This list is stored in the
    agent's memory but is not yet available to the Stork server.
    This change provides no new user-visible capabilities to the
    Stork agent, but lays the foundation for the zone viewer
    feature.
    (Gitlab #1647)