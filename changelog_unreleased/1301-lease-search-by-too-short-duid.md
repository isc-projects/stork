[bug] slawek

    Stork now respects the minimum DUID length of 3 enforced by Kea 2.3.8+ and
    does not query it for leases when too short DUID is specified in the lease
    search box.
    (Gitlab #1301)
