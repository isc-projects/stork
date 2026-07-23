[func] marcin

    Improved tracking zone transfers in the BIND 9 logs. Two new
    transfer states are determined: up-to-date and failed. The former
    marks the zone transfers for which the secondary found that it
    already has a current copy of the zone. The latter marks the
    transfers for which the status was neither success nor up to date.
    In both cases, the tracker also captures subsequent log messages
    containing the transfer statistics, so they can be displayed in
    the UI.
    (Gitlab #2579)
