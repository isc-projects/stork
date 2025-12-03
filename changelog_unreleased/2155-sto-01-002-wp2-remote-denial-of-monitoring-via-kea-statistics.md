[bug] marcin

    Address a potential remote denial of monitoring in the Kea
    Prometheus exporter. The exporter could stop working after
    receiving statistics from Kea with malformed subnet, pool
    or prefix pool ID. In addition, the exporter could stop
    working after receiving pool-level statistic at the subnet
    level.
    (Gitlab #2155)
