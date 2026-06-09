[bug] william

    Correct a bug which prevented the Leases List from updating. Due
    to improper error handling, the entire leasefile update from the
    agent would be ignored if any lease in the update had been seen
    before.
    (Gitlab #2523)
