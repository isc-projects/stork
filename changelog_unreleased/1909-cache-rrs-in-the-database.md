[bug] marcin

    Zone RRs are cached in the database for a viewed zone. It avoids
    repeated zone transfers (AXFR) when showing zone contents. The
    cached zone contents can be refreshed using zone transfer on
    demand by clicking a button in the UI.
    (Gitlab #1909)
