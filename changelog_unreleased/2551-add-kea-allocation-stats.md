[func] sebastien.degroof

    The Prometheus Kea Exporter now collects Kea's lease allocation
    and lease allocation failure statistics, consistent with how
    address statistics are exposed. Both global and per-subnet samples
    are supported.
    (Gitlab #2551)
