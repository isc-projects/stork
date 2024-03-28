[bug] slawek

    Fixed filtering global host reservations with a combination of other
    filters. Previously, global hosts were appended to the hosts
    returned by other filters. Now, a subset of global hosts matching
    other filters is returned.
    (Gitlab #1282)
