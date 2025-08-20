[build] marcin

    Improved detection of PowerDNS config file location. The agent
    tries to find the config file in typical locations. Added the
    command line argument (`--powerdns-path`) and environment variable
    (`STORK_AGENT_POWERDNS_CONFIG`) to specify custom PowerDNS config
    file location.
    (Gitlab #1930)
