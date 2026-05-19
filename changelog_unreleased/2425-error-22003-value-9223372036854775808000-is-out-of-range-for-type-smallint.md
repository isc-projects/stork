[bug] slawek

    Added clipping the utilization value when it exceeds the smallint
    (2 bytes) range to avoid interrupting statistics fetching. Such a
    value indicates duplication daemons in the Stork database or
    overlapping subnet pools for Kea daemons that are not combined into
    an HA pair.
    (Gitlab #2425)
