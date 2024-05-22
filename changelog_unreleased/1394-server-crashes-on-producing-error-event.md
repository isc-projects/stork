[bug] slawek

    Fixed a server crash that occurred when a few commands were sent to
    Kea at once and some of them (but not all) failed. Stork incorrectly
    handled this case while generating an error event.
    (Gitlab #1394)
