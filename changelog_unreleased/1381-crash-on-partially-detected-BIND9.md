[bug] slawek

    Fixed a bug that may cause a Stork server crash if the BIND 9
    process was detected but the Stork agent failed to fetch its data
    over RNDC protocol due to insufficient permissions or other
    connectivity problems.
    (Gitlab #1381)
