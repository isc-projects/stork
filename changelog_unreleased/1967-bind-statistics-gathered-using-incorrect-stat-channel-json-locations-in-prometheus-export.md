[bug] marcin

    Fix collecting and exporting some of the BIND 9 statistics
    pertaining to zone transfers. Exporting new statistics to
    Prometheus: XfrReqDone, AXFRReqv4, AXFRReqv6, IXFRReqv4,
    IXFRReqv6.
    (Gitlab #1967)
