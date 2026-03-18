[bug] slawek

    Fixed BIND 9 detection failing to parse -t (chroot) and
    -c (config) flags when the path to the named binary
    contained a directory component called "named" (e.g.,
    /var/lib/named/sbin/named).
    (Gitlab #2410)
