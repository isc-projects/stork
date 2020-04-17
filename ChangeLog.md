* 57 [func] marcin

    Fixed a bug in the HA service detection when new Kea app was
    being added. The visible side effect of this bug was the lack
    of the link to the remote server app in the HA status view
    in the UI.
    (Gitlab #240)

* 56 [func] godfryd

    Added links to Grafana. Added web page for managing global
    settings.
    (Gitlab #231)

* 55 [bug] godfryd

    Fixed starting Stork server: now if password to database
    is set to empty it does not ask for password in terminal.
    It asks only when the STORK_DATABASE_PASSWORD environment
    variable does not exist.
    (Gitlab #203)

* 54 [func] marcin

    Improved Kea High Availability status monitoring. The status is
    cached in the database and thus it is available even if the
    HA partners are offline. The presented status now includes
    connectivity status between Stork and the Kea servers, the
    time of the last failover event and others.
    (Gitlab #226)

* 53 [func] godfryd

    Added a dashboard presenting DHCP and DNS overview.
    (Gitlab #226)

* 52 [func] godfryd

    Added links to BIND 9 manual and Kea manual in Help menu.
    (Gitlab #221)

* 51 [bug] matthijs

    Added querying named stats from Bind 9 apps periodically.
    (Gitlab #211)

Stork 0.6.0 released on 2020-04-03.

* 50 [bug] marcin

    Corrected a bug which caused unexpected deletion of the
    host reservations fetched from the Kea configuration
    files.
    (Gitlab #225)

* 49 [func] matthijs

    Updated Prometheus & Grafana in the demo installation with BIND 9.

    Implemented BIND 9 exporter in Go and embedded it in Stork
    Agent for showing Cache Hit Ratio.

    Implemented DNS traffic simulator as web app for the demo
    installation. Internally it runs a single query with dig, or
    starts flamethrower (a DNS performance tool) for selected server
    with indicated parameters.
    (Gitlab #10)

* 48 [doc] marcin, sgoldlust

    Documented the use of Host Reservations in Stork ARM.
    (Gitlab #223)

* 47 [func] marcin

    Stork server periodically fetches host reservations from the Kea
    instances having host_cmds hooks library loaded.
    (Gitlab #214)

* 46 [func] marcin

    Host reservations are listed and the UI. It is possible to filter
    reservations by reserved IP address or host identifier value.
    (Gitlab #210)

* 45 [func] matthijs

    Retrieve some cache statistics from named and show Cache Hit
    Ratio on the dashboard.
    (Gitlab #64)

* 44 [func] godfryd

    Added possibility to run Stork server without Nginx or Apache,
    ie. static files can be served by Stork server. Still it is
    possible to run Stork server behind Nginx or Apache which
    will do reverse proxy or serve static files.
    (Gitlab #200)

* 43 [func] marcin

    Implemented data model for IP reservations and detection of IP
    reservations specified within a Kea configuration file. Detected
    reservations are not yet used in the UI.
    (Gitlab #188, #206)

* 42 [func] godfryd

    Prepared scripts for building native RPM and deb packages
    with Stork server and Stork agent (total 4 packages).
    They are prepared for Ubuntu 18.04 and CentOS 8.
    (Gitlab #187)

* 41 [func] godfryd

    Added settings in Stork. They are stored in database, in setting
    table. No UI for settings yet.
    (Gitlab #169)

* 40 [func] godfryd

    Exposed access to API docs and ARM docs in new Help menu.
    (Gitlab #199)

* 39 [func] matthijs

    Update the data model such that applications can have multiple
    access points.  Parse named.conf to detect both "control"
    and "statistics" access point.
    (Gitlab #170)

Stork 0.5.0 released on 2020-03-06.

* 38 [doc] tomek

    Updated Stork ARM with regards to networks view, installation
    instructions and Java, Docker dependencies.
    (Gitlab #163, #183)

* 37 [bug] marcin

    Improved shared network detection mechanism to take into account
    the family of the subnets belonging to the shared network. This
    prevents the issue whereby two IPv4 and IPv6 subnets belonging
    to separate shared networks having the same name would be shown
    as belonging to the same shared network in the UI.
    (Gitlab #180)

* 36 [func] godfryd

    Added presenting IP addresses utilization within subnets and
    subnet statistics, e.g. a number of assigned addresses, in the UI
    (subnets and shared networks pages). Statistics are fetched
    from the monitored Kea apps periodically and can be manually
    refreshed in the UI.
    (Gitlab #178, #185)

* 35 [func] marcin

    Corrected a bug in the Stork server which caused failures when
    parsing prefix delegation pools from the Kea configurations.
    The Server subsequently refused to monitor the Kea apps including
    prefix delegation pools.
    (Gitlab #179)

* 34 [func] godfryd

    Added support for Prometheus & Grafana in the demo installation.
    Added preconfigured Prometheus & Grafana containers to docker-compose.
    Added Kea and BIND 9 Prometheus exporters.

    Implementated Kea exporter in Go and embedded it in Stork Agent.
    It is based on kea_exporter in python:
    https://github.com/mweinelt/kea-exporter

    Implemented DHCP simulator as web app for the demo installation.
    Internally it starts perfdhcp for selected subnet with indicated
    parameters.
    (Gitlab #167)

* 33 [func] marcin

    New data model is now used by the server to hold the information
    about the subnets and shared networks. There is no visible change
    to the UI yet. This change mostly affects how the data is stored
    in the database.
    (Gitlab #172)

* 32 [func] marcin

    Created data model for shared networks, subnets and pools and
    implemented mechanism to match configurations of Kea apps with
    these structures in the database. This mechanism is not yet used
    by the server when adding new apps via the UI.
    (Gitlab #165)

* 31 [func] godfryd

    Added querying lease stats from Kea apps periodically.
    Stats are not yet presented in the UI.
    (Gitlab #166)

* 30 [func] marcin

    Created data model for services and implemented a mechanism to
    to automatically associate a new Kea application with a High
    Availability service when the application is configured to use
    High Availability. This mechanism is not yet used by the server
    when the Kea application is added via the UI. The usage of
    this mechanism will be added in the future tickets.
    (Gitlab #137)

* 29 [func] godfryd

    Added initial support for DHCP shared networks. They are presented
    on dedicated page. Subnets page now is also presenting subnets
    that belong to shared networks.
    (Gitlab #151)

Stork 0.4.0 released on 2020-02-05.

* 28 [doc] tomek

    Subnets inspection is now documented.
    (Gitlab #149)

* 27 [func] matthijs

    Show more status information for named: up time, last reloaded,
    number of zones.
    (Gitlab #140)

* 26 [func] godfryd

    Added initial support for DHCP subnets. They are presented
    on dedicated page and on apps' pages. For now only these subnets
    are listed which do not belong to shared networks.
    (Gitlab #47)

* 25 [func] matthijs

    Improve getting configuration of the BIND 9 application.
    Stork now retrieves the control address and port from
    named.conf, as well as the rndc key, and uses this to interact
    with the named daemon.
    (Gitlab #130)

* 24 [bug] godfryd

    Apps are now deleted while the machine is being deleted.
    (Gitlab #123)

Stork 0.3.0 released on 2020-01-10.

* 23 [func] godfryd

    Added presenting number of all and misbehaving applications
    on the dashboard page. If there are no applications added yet,
    the dashboard redirects to the list of connected machines.
    (Gitlab #120)

* 22 [doc] marcin

    Updated Stork ARM. Added documentation of the High Availability
    status monitoring with Kea. Added new sections describing applications
    management.
    (Gitlab #122)

* 21 [func] godfryd

    Added new Rake tasks to build and start two containers
    with Kea instances running as High Availability partners.
    (Gitlab #126)

* 20 [func] matthijs

    Add BIND 9 application to Stork.  Detects running BIND 9
    application by looking for named process.  Uses rndc to retrieve
    version information.
    (Gitlab #106)

* 19 [func] marcin

    Kea High Availability status is presented on the Kea application page.
    (Gitlab #110)

* 18 [func] marcin

    Logged user can now change his/her password. Also, users can be
    associated with one of the two default permission groups: super-admin
    and admin. The former can manage users' accounts. The latter is not
    allowed to manage other users' accounts.
    (Gitlab #97)

* 17 [func] marcin

    Implemented a mechanism by which it is possible to send a command
    from the Stork server to Kea via Stork Agent and Kea Control
    Agent.
    (Gitlab #109)

Stork 0.2.0 released on 2019-12-04.

* 16 [bug] marcin

    Fixed an issue with closing a tab on the user management page.
    (Gitlab #100)

* 15 [doc] tomek

    Users and machines management is now documented in the Stork ARM.
    (Gitlab #99)

* 14 [doc] sgoldlust

    Introduced new Stork logo in the documentation.
    (Gitlab #95)

* 13 [build] tomek

    Extended the build system to be able to run on MacOS. Also updated
    installation instructions regarding how to build and run Stork natively.
    (Gitlab #87)

* 12 [func] marcin

    Enabled creation and editing of Stork user accounts in the UI.
    (Gitlab #25)

* 11 [func] marcin

    Stork server automatically migrates the database schema to the latest
    version upon startup.
    (Gitlab #33)

Stork 0.1.0 released on 2019-11-06.

* 10 [doc] marcin

    Updated ARM with a description how to sign in to the system using the
    default administrator account.
    (Gitlab #84)

* 9 [doc] tomek

    Initial ARM version added.
    (Gitlab #27)

* 8 [func] marcin

    Enabled sign-in/sign-out mechanism with HTTP sessions based on cookies.
    The default admin account has been created with default credentials.
    (Gitlab #22)

* 7 [func] godfryd

    Added initial implementation of the page which allows for adding new machines
    and listing them. The missing part of this implementation is the actual storage
    of the machines in the database. In addition, the agent has been extended to
    return a state of the machine.
    (Gitlab #23)

* 6 [func] godfryd

    Added initial implementation of Stork Agent. Implemented basic communication
    between Stork Agent and Stork Server using gRPC (Server initiates connection
    to Agent).
    (Gitlab #26)

* 5 [func] marcin

    Added stork-db-migrate tool to be used for migrating the database
    schema between versions and returning the current schema version
    number. Also, added basic schema with SQL tables holding system
    users and session information.
    (Gitlab #20)

* 4 [doc] tomek

    Added several text files: AUTHORS (lists project authors and contributors), ChangeLog.md
    (contains all new user visible changes) and CONTRIBUTING.md (Contributor's guide, explains how
    to get your patches accepted in Stork project in a seamless and easy way.
    (Gitlab #17)

* 3 [func] godfryd

   Added Swagger-based API for defining ReST API to Stork server.
   Added initial Web UI based on Angular and PrimeNG. Added Rakefile
   for building whole solution. Removed gin-gonic dependency.
   (Gitlab #19)

* 2 [build] godfryd

   Added initial framework for backend, using go and gin-gonic.
   (Gitlab #missing)

* 1 [func] franek

   Added initial proposal for Grafana dashboard.
   (Gitlab #6)


For complete code revision history, see
	http://gitlab.isc.org/isc-projects/stork

LEGEND
* [bug]   General bug fix.  This is generally a backward compatible change,
          unless it's deemed to be impossible or very hard to keep
	      compatibility to fix the bug.
* [build] Compilation and installation infrastructure change.
* [doc]   Update to documentation. This shouldn't change run time behavior.
* [func]  new feature.  In some cases this may be a backward incompatible
	      change, which would require a bump of major version.
* [sec]   Security hole fix. This is no different than a general bug
          fix except that it will be handled as confidential and will cause
 	      security patch releases.
* [perf]  Performance related change.

*: Backward incompatible or operational change.
