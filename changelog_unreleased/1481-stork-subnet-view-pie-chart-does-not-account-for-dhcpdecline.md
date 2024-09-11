[func] marcin

    Handle the case in Stork statistics presentation when the number
    of declined leases is lower than the number of assigned leases.
    In this case, Stork now estimates the number of leases with
    uncertain availability, and the number of free leases. These
    statistics are presented on the pie charts. It also eliminates
    negative lease statistics that were sometimes presented when the
    statistics returned by Kea were wrong.
    (Gitlab #1481)