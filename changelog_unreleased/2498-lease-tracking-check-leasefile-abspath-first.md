[bug] william

    Correct an oversight which prevented Stork from activating Lease
    Tracking when used with Kea versions older than 3.1.0, even when
    Kea is configured with an absolute lease memfile path.
    (Gitlab #2498)
