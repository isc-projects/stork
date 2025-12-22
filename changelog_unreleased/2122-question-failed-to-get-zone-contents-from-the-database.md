[bug] marcin

    Fixed an issue in the Stork agent which misinterpreted the keyword
    "any" in the BIND 9 listen-on configuration clause. Now, it tries
    to communicate with BIND 9 over the local loopback addresses when
    it finds this keyword.
    (Gitlab #2122)

