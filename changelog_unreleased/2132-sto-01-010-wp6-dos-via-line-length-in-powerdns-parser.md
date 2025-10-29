[bug] marcin

    Maximum size of the PowerDNS parser buffer is 16kB. The initial
    buffer size is 512B. It is aimed at reducing the memory usage
    during PowerDNS configuration parsing.
    (Gitlab #2132)
