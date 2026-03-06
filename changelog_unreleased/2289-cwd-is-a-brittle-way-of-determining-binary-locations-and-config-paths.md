[bug] andrei

    The agent can now detect Kea daemons started with a relative
    executable path; previously, it tried to guess that the
    executable would be located in the current working directory
    from which the process was started.
    (Gitlab #2289)
