Stork 2.1.1 released on 2025-02-05.

* 491 [bug] slawek

    Fixed a bug in the "install:*" commands that caused the overriding
    of attributes of the system directories on specific operating
    systems.
    (Gitlab #1614)

* 490 [func] marcin

    Implemented a DNS "zone inventory" in the Stork agent. It adds
    the capability for the Stork agent to gather a list of views
    and zones from BIND 9 on startup. This list is stored in the
    agent's memory but is not yet available to the Stork server.
    This change provides no new user-visible capabilities to the
    Stork agent, but lays the foundation for the zone viewer
    feature.
    (Gitlab #1647)

* 489 [doc] slawek, bscott

    Added an LDAP hook documentation.
    (Gitlab #1652)

* 488 [doc] sgoldlust

    Editorial and grammatical corrections in UI components.
    (Gitlab #1673)

* 487 [build] slawek

    Updated dependencies including the Go 1.23.5, and several
    JavaScript, Python, Ruby and Go packages.
    (Gitlab #1676)

* 486 [func] piotrek

    Reworked filtering in Machines list view. Filters are now unified
    with other Stork views that display tables and allow data filtering.
    It is possible to search for machines globally by keyword regardless
    of the authorization state.
    (Gitlab #1533)

* 485 [func] slawek

    The authentication methods provided by hooks are presented before
    the internal method on the login page.
    (Gitlab #1648)

* 484 [bug] slawek

    Fixed adding a host reservation with a hex identifier delimited by a
    dash.
    (Gitlab #1670)

* 483 [func] slawek

    Stork agent now retrieves the Basic Auth credentials from the Kea
    CA configuration file. It is no longer supported to provide the
    JSON file with a login and password to the Kea REST API. The agent
    selects credentials with a "stork" user name or prefix.
    If no user is found, it uses the first credentials entry.
    (Gitlab #1624)

* 482 [func] slawek

    Added an input to alter the subnet name.
    (Gitlab #1641)

* 481 [bug] marcin

    Enabled timeouts for HTTP client connecting to Kea. It should help
    to gracefully handle communication issues between Stork agents and
    Kea servers.
    (Gitlab #1467)

* 480 [func] piotrek

    The feature which informs about used Kea, BIND 9, and Stork versions
    was improved to download the metadata about current software
    releases from an online source. The user may disable this feature in
    settings so the built-in offline JSON file can be used as a metadata
    source instead.
    (Gitlab #1636)

* 479 [func] piotrek

    Anchors to Kea hooks documentation were improved to be automatically
    generated. This way the anchors will be always up-to-date with new
    Kea hooks.
    (Gitlab #1588)

* 478 [build] tomek

    The native packages now have better descriptions.
    (Gitlab #1618)

Stork 2.1.0 released on 2024-12-11.

* 477 [bug] slawek

    Fixed a bug in the LDAP hook that made connecting to the server over
    the LDAPS (TLS) protocol impossible. Thanks to Cameron Ditchfield
    for proposing a fix.
    (Gitlab #1488)

* 476 [func] slawek

    The subnet's user context is now displayed on the subnet page. The
    "subnet-name" property value of the context is also shown on the
    subnet list.
    (Gitlab #459)

* 475 [func] piotrek

    The "Software versions" view was updated with a list of small
    changes, such as corrected help tooltip text, adjusted notification
    message severity for development releases, and improved badge
    notification displayed in the top menu.
    (Gitlab #1571)

* 474 [build] slawek

    The demo now loads the LDAP hook and starts the OpenLDAP server.
    (Gitlab #1554)

* 473 [bug] tomek

    The Stork agent can now detect the default RNDC key of BIND 9.
    The parser code can now handle algorithm definitions both quoted and
    unquoted.
    (Gitlab #1590)

Stork 2.0.0 released on 2024-11-13.

* 472 [bug] marcin

    Fixed a problem with resetting the database schema to the initial
    version, using the stork-tool db-reset command, when some of the
    database users had no email assigned.
    (Gitlab #1583)

* 471 [perf] slawek

    Solved a problem with consuming abnormally much memory if the Stork
    server monitored multiple Kea servers that simultaneously managed
    thousands of shared networks.
    (Gitlab #1552)

* 470 [func] marcin

    Conditionally set ddns-use-conflict-resolution and
    ddns-conflict-resolution-mode, depending on the configured
    Kea version. Previously a user could set one of these
    parameters for the Kea versions that did not support them,
    causing configuration errors.
    (Gitlab #1536)

* 469 [bug] slawek

    Fixed a migration to downgrade a database version that failed if any
    host reservation was specified both in a configuration file and in a
    database.
    (Gitlab #1553)

* 468 [func] slawek

    Upgraded Grafana and Prometheus versions. Fixed a bug with fetching
    statistics from a single DHCPv6 daemon. Added Grafana links for IPv6
    subnets.
    (Gitlab #1322, #1529)

* 467 [build] slawek

    Build Stork DEB and RPM packages with GLIBC 2.31 to preserve
    compatibility with legacy operating systems.
    (Gitlab #1538)

* 466 [func] piotrek

    Implemented a new feature to inform about used Kea, Bind9
    and Stork versions. There is a new view where all found
    versions are summarized together with information about
    current ISC software releases.
    (Gitlab #222)

* 465 [bug] piotrek

    Fixed a bug where login was not possible using
    the LDAP authentication method.
    (Gitlab #1540)

* 464 [bug] slawek

    Fixed a ping request handling issue in the agent registration flow
    with the server token. It caused an error at the end of the
    registration attempt.
    (Gitlab #1491)

* 463 [func] slawek

    Added a possibility to force users to change a password. The default
    administrator must always change password when logging in for the
    first time.
    (Gitlab #1385)

* 462 [build] marcin

    Changes in DEB packages to use useradd instead of adduser command.
    The former is available by default on deb-based distributions but
    the latter isn't, causing potential issues with installing Stork
    on these systems.
    (Gitlab #1537)

* 461 [func] marcin

    Return errors reported by Kea to a Stork user unsuccessfully
    updating Kea configuration.
    (Gitlab #1535)

* 460 [func] slawek

    Refactored the application detection loop in the Stork agent to
    prevent continuous logging of the same detection errors. Removed the
    log message about successfully finding the BIND 9 configuration.
    Stork agent no longer logs the RNDC key status for BIND 9 statistics
    channel.
    (Gitlab #1422, #1388, #1384)

* 459 [func] marcin

    Stork passwords can contain space characters.
    (Gitlab #1532)

* 458 [func] marcin

    Stork server detects the directory with static UI files relative
    to the stork-server binary when rest-static-files-dir is
    not specified. It eliminates the need to specify this parameter
    when Stork server is installed in a custom directory.
    (Gitlab #1434)

* 457 [build] marcin

    Moved High Availability monitoring panel to the bottom of the
    Kea daemon tab. This change prevents browser scroller's
    position change when the HA status is updated.
    (Gitlab #1472)

* 456 [func] marcin

    Added possibility to set custom welcome message in the Stork
    login page.
    (Gitlab #1431)

Stork 1.19.0 released on 2024-10-02.

* 455 [func] marcin

    RPS statistics displayed with two fractional digits precision.
    (Gitlab #1510)

* 454 [build] andrei

    Stork now installs OpenRC service files on Alpine systems.
    (Gitlab #1457)

* 453 [build] slawek

    Updated dependencies including the Go 1.23.1, Angular 17.3.12,
    and several JavaScript, Python and Ruby packages.
    (Gitlab #1512)

* 452 [bug] slawek

    Fixed a problem with recognizing the wildcard IP address in the inet
    clause in the BIND9 configuration, preventing the Stork agent from
    establishing a connection to the named daemon.
    (Gitlab #1495)

* 451 [func] slawek

    Extended a form for editing global Kea configuration with support
    for altering the global DHCP options.
    (Gitlab #1447)

* 450 [bug] slawek

    Fixed the Kea and BIND 9 group names assigned to the user
    installed with the Stork agent package.
    (Gitlab #1506)

* 449 [bug] slawek

    Fixed a rare bug that caused a Stork server to crash when the Stork
    agent was able to detect the BIND 9 process, but it couldn't fetch
    its configuration due to insufficient rights.
    (Gitlab #1492)

* 448 [bug] marcin

    Restore missing links to the subnets in Grafana.
    (Gitlab #1489)

* 447 [bug] marcin

    Corrected command name for fetching DHCPv6 statistics. The wrong
    command was sent previously causing a failure to fetch this
    statistics.
    (Gitlab #1493)

* 446 [func] marcin

    Handle the case in Stork statistics presentation when the number
    of declined leases is lower than the number of assigned leases.
    In this case, Stork now estimates the number of leases with
    uncertain availability, and the number of free leases. These
    statistics are presented on the pie charts. It also eliminates
    negative lease statistics that were sometimes presented when the
    statistics returned by Kea were wrong.
    (Gitlab #1481)

* 445 [build] tomek

    Fixed a problem with BIND9 containers used in the demo. Some time
    around Aug 2024, the official BIND9 images switched from Debian
    to Alpine.
    (Gitlab #1494)

* 444 [func] marcin

    Implemented a form for editing selected global Kea configuration
    parameters.
    (Gitlab #1366)

* 443 [func] slawek

    Added a Kea checker to verify the Kea Lease Commands Hooks Library
    is loaded in the Kea DHCP daemon.
    (Gitlab #1456)

* 442 [doc] slawek

    Fixed a link in the docs to the script setting up the CloudSmith
    repository on Alpine operating systems.
    (Gitlab #1462)

* 441 [bug] slawek

    Fixed the built-in description of the hook packages. The environment
    variables were not substituted.
    (Gitlab #1470)

* 440 [func] piotrek

    Shared networks list view was refactored to be fully responsive.
    Filtering was reworked. Separate filters are now available
    in the shared networks table header.
    (Gitlab #1464)

* 439 [func] marcin

    Server assignments panel moved to the beginning of the
    subnet and shared network form. It promotes the natural
    flow of editing the subnets and shared networks because
    the assignments selection impacts the later parts of the
    forms.
    (Gitlab #1438)

* 438 [func] piotrek

    Stork UI was adjusted to be fully responsive. Now, it is
    usable on portable devices, e.g., smartphones and tablets.
    (Gitlab #1433)

Stork 1.18.0 released on 2024-08-07.

* 437 [build] slawek

    Changed the tool to build packages from "fpm" to "nfpm". This
    enabled automated build of Stork packages for Alpine Linux.
    (Gitlab #1114)

* 436 [build] slawek

    Changed the default Stork hook directory to be located outside
    '/var' to be compatible with strict Linux security policies.
    (Gitlab #1227)

* 435 [bug] slawek

    Added configurable timeouts for database read/write operations.
    These settings may be useful to avoid the read or write hangs when
    the server looses connectivity to the database.
    (Gitlab #1436)

* 434 [build] slawek

    Fixed the security vulnerabilities reported by the GitHub Dependabot
    and updated dependencies, including Go 1.22.5, PrimeNG 17.18.5, and
    several Python and Ruby packages.
    (Gitlab #1446)

* 433 [bug] slawek

    Fixed the permanent stopping of pullers on the temporary database
    failure.
    (Gitlab #1437)

* 432 [func] marcin

    Added global Kea parameters view accessible from the Kea daemon
    tab.
    (Gitlab #1449)

* 431 [func] marcin

    Display warning message when new machine requests registration.
    (Gitlab #1413)

* 430 [func] slawek

    Minor improvements in the Stork agent register command. Added a CLI
    flag to run the registration in non-interactive mode.
    (Gitlab #1427)

* 429 [doc] slawek

    Updated a section about supported operating systems.
    (Gitlab #1297)

* 428 [bug] slawek, marcin, andrei

    Prevented Stork tool from auto-discovering migration files. This
    feature has never been necessary, but it could raise an error if the
    Stork user can't access the searched directory.
    (Gitlab #1439)

* 427 [build] slawek

    Upgraded the Kea version in system tests to 2.7.0.
    (Gitlab #1350)

* 426 [func] piotrek

    Applied most recent PrimeNG theme Aura Blue. This updates
    Stork UI. Also dark/light mode support was added.
    (Gitlab #1379)

* 425 [bug] marcin

    Delete a subnet from shared network in Kea before deleting the
    subnet. It is a workaround for the Kea issue #3455. Before this
    change, a subnet could silently fail to delete from a Kea server
    when it belonged to a shared network.
    (Gitlab #1425)

* 424 [bug] piotrek

    Fixed a bug with displaying help tooltip in views like
    Machines list or High Availability section of Kea app view.
    The help tip was not displayed in proper location, or was
    not displayed at all.
    (Gitlab #1399)

* 423 [doc] slawek

    Described why Stork allocates so much virtual memory.
    (Gitlab #1389)

* 422 [func] slawek

    The Stork server's Prometheus endpoint exports new metrics, such as
    the number of total and assigned addresses and delegated prefixes.
    (Gitlab #1375)

* 421 [func] marcin

    Shared networks can be selectively deleted using the Stork UI.
    (Gitlab #1405)

* 420 [func] marcin

    Implemented a UI form for creating new shared networks.
    (Gitlab #1403)

* 419 [func] marcin

    Added new page displaying communication issues between the server
    and the monitored machines.
    (Gitlab #1316)

Stork 1.17.0 released on 2024-06-12.

* 418 [bug] slawek

    Added support for big numbers in the statistics introduced in Kea
    2.5.3. Added a new Kea checker to notify about degraded or missing
    capabilities to gather the statistics for the previous Kea versions.
    (Gitlab #1193)

* 417 [doc] tomek

    The security policy document is now available as a separate
    document.
    (Gitlab #1276)

* 416 [sec] marcin

    Added a new setting to disable registering new machines in the Stork
    server.
    (Gitlab #1339)

* 415 [build] slawek

    Fixed the security vulnerabilities reported by the Github Dependabot
    and updated dependencies including Go 1.22.4, Angular 17.3.8,
    PrimeNG 17.17.0, GoSwagger v0.31.0, OpenAPI Generator 7.6.0 and
    several Python and Ruby packages.
    (Gitlab #1380)

* 414 [bug] slawek

    Fixed inconsistency between the utilization presented in the UI and
    returned by the metrics endpoint. The server's metrics endpoint no
    longer returns delayed statistics.
    (Gitlab #1214)

* 413 [func] piotrek

    Reworked filtering on hosts' reservations page. Separate filters
    are now available in the hosts' table header. Applied filters are
    stored in the session storage of the web browser.
    (Gitlab #1265)

* 412 [func] marcin

    Implemented a form for updating shared network parameters.
    (Gitlab #1370)

* 411 [bug] piotrek

    Fixed a bug in UI of the password change form.
    The problem was when user provided New password containing special
    characters e.g. +. Even though New password and Confirm password
    where identical, form validation was failing and user could not
    submit New password change form. Similar issues could be
    experienced when New user account was being created or existing user
    account being edited by an admin. The issue there was also fixed.
    (Gitlab #1275)

* 410 [func] slawek

    Refactored the IP reservation and host tables to associate the
    reservation data with particular daemons that store them. Fixed a
    bug causing duplication of the client class section on the host
    page.
    (Gitlab #1318)

* 409 [bug] slawek

    Fixed a bug that may cause a Stork server crash if the BIND 9
    process was detected but the Stork agent failed to fetch its data
    over RNDC protocol due to insufficient permissions or other
    connectivity problems.
    (Gitlab #1381)

* 408 [bug] slawek

    Fixed a server crash that occurred when a few commands were sent to
    Kea at once and some of them (but not all) failed. Stork incorrectly
    handled this case while generating an error event.
    (Gitlab #1394)

* 407 [func] slawek

    Added new labels for subnet metrics exported to Prometheus to always
    include subnet ID.
    (Gitlab #1323)

* 406 [func] slawek

    Added validation of the existing GRPC certificates before running
    the agent. This prevents the agent from starting if it is not able
    to establish a connection to the server.
    (Gitlab #1352)

* 405 [func] ! robin.berger, slawek

    Separated the bind user domain name (DN) from the root DN used to
    log in users.
    (Gitlab #1325)

* 404 [sec] slawek

    The server no longer reveals the correct agent token when the token
    specified in the ping call via REST API is invalid. Previously, this
    endpoint could be used to discover a valid agent token. However,
    the risk was minimal because it required hijacking the server token
    first.
    (Gitlab #1340)

* 403 [bug] slawek

    Fixed not showing the hostname-only reservations (reservations
    without assigned IP addresses) on the list while the filter was set.
    (Gitlab #1337)

* 402 [bug] slawek

    Fixed the scheme of the server URL in the installation script, which
    was always HTTP, even if the server was configured with SSL.
    (Gitlab #1342)

* 401 [build] marcin

    Upgraded storybook-addon-mock to version 5.0.0. Existing stories
    failed to run with the older version.
    (Gitlab #1359)

* 400 [bug] piotrek

    Fixed a bug in Stork UI with displaying help tooltips on smaller
    displays. Sometimes the header and part of the help tooltip was
    not visible. Now, the whole help tooltip is visible for all screen
    sizes.
    (Gitlab #1305)

* 399 [bug] slawek

    Fixed a problem with improper redirecting after login. If the
    non-logged user entered any subpage rather than the root page, it
    was stuck on the login page after signing in.
    (Gitlab #1355)

Stork 1.16.0 released on 2024-04-05.

* 398 [build] piotrek

    Updated dependencies including the Go 1.22.2, Angular 17.3.2,
    and several Python packages. Updated Alpine CI image to include
    new Go 1.22.2 package.
    (Gitlab #1353)

* 397 [func] piotrek

    Improved UI/UX when Machine is being authorized. This process may
    take some time, so for the time when this operation is in progress,
    machines table UI is greyed out. The same improvement was added
    when authorizing more than one machine at once.
    (Gitlab #1269)

* 396 [func] piotrek

    Added information about empty dataset to tables in views:
    Hosts list, Shared networks list, Subnets list, Users list.
    Now, whenever there is no data to be displayed, the feedback
    about that fact is shown.
    (Gitlab #1307)

* 395 [bug] piotrek

    Fixed a minor UI issue where a longer subnet prefix could break in
    two lines in the subnet bar. Now, the subnet bar should always be
    displayed in one line.
    (Gitlab #1299)

* 394 [build] slawek

    Improved the package installation scripts to avoid turning off the
    Stork systemD service on update.
    (Gitlab #1319)

* 393 [bug] slawek

    Stork now respects the minimum DUID length of 3 enforced by Kea
    2.3.8+ and does not query it for leases when too short DUID is
    specified in the lease search box.
    (Gitlab #1301)

* 392 [func] marcin

    High Availability status for a daemon now contains the state of
    all HA relationships. It facilitates monitoring the hub-and-spoke
    configuration recently implemented in Kea.
    (Gitlab #1274)

* 391 [bug] marcin

    Enabled some dead events for BIND 9 and added the events panel on
    the BIND9 app tab.
    (Gitlab #1303)

* 390 [doc] marcin

    Added reference to the ISC KB article describing how to generate
    the certificates and keys.
    (Gitlab #1341)

* 389 [bug] slawek

    Fixed unavailable Prometheus metrics endpoint of the Stork server in
    the demo environment. Use Stork server logger in the Prometheus
    metrics collector.
    (Gitlab #1289)

* 388 [func] slawek

    Stork now detects a given daemon's inconsistent or duplicated DHCP
    data in different host data sources and displays them as labels on
    the host list.
    (Gitlab #977)

* 387 [bug] slawek

    Fixed filtering global host reservations with a combination of other
    filters. Previously, global hosts were appended to the hosts
    returned by other filters. Now, a subset of global hosts matching
    other filters is returned.
    (Gitlab #1282)

* 386 [bug] slawek

    Fixed setting the application state puller interval. Its input box
    value was not used.
    (Gitlab #1258)

* 385 [func] marcin

    Show an alert in a prominent place about the connectivity issues
    between the Stork server and the monitored machines.
    (Gitlab #1222)

* 384 [build] marcin

    Enabled building Stork and the demo on Apple M3 chipset.
    (Gitlab #1321)

* 383 [bug] marcin

    Prevent a hang in the Stork server shutdown when there are
    open SSE connections.
    (Gitlab #1244)

Stork 1.15.1 released on 2024-03-27.

* 382 [sec] ! slawek

    Fixed CVE-2024-28872 vulnerability.
    (Gitlab #1328)

Stork 1.15.0 released on 2024-02-07.

* 381 [build] slawek

    Fixed the security vulnerabilities reported by the Github Dependabot and
    updated dependencies including the Go 1.21.6, Angular 17, PrimeNG 17,
    OpenAPI Generator 7 and several Python and Ruby packages. Added a
    vulnerability scanner for Python dependencies.
    (Gitlab #1285)

* 380 [ui] andrei

    The help tip on the leases search page now shows a table of criteria that
    leases can be searched by for kea-dhcp4 and kea-dhcp6 respectively.
    (Gitlab #1150)

* 379 [build] tomek

    Fixed several smaller Python issues reported by CodeQL and other Python linters. The
    `gen_kea_config.py` tool used in demo now takes an optional `--seed` parameter that,
    if specified, will initiate the PRNG to given value. This allows to use repeat
    randomized test runs, if necessary.
    (Gitlab #1264)

* 378 [build] andrei

    Coverage reporting was enabled for unit tests in the LDAP hook project.
    (Gitlab #1174)

* 377 [bug] piotrek

    Fixed a bug where Apps list view applied old filtering
    without user's intention. Also fixed a bug where wrong
    Kea/Bind app name was displayed in breadcrumbs.
    (Gitlab #1267)

* 376 [build] andrei

    UI linters for unused variables and unused imports are now run with
    "rake lint:ui". Setting the FIX environment variable enables autofixing, but
    it only works with unused imports. The reported lint errors were fixed.
    (Gitlab #994)

* 375 [doc] andrei

    The command line tools required to install Stork packages via Cloudsmith are
    now documented in the ARM.
    (Gitlab #1147)

* 374 [func] marcin

    Subnets can be selectively deleted using the Stork UI.
    (Gitlab #1284)

* 373 [doc] slawek

    Documented changes provided in the CI docker images for particular tags.
    (#1209)

* 372 [func] marcin

    New subnets in Kea can be now created using the Stork UI.
    (Gitlab #1277)

* 371 [func] slawek

    Covered the application detection loop of the Stork agent with unit tests.
    (Gitlab #1163)

* 370 [bug] slawek

    Address and prefix counters are now approximated to one digit after decimal
    point instead of rounded down.
    (Gitlab #1256)

* 369 [perf] slawek

    Fixed a high memory usage while browsing the subnet and shared network
    lists.
    (Gitlab #1263)

* 368 [func] marcin

    Subnet edit form allows for moving a subnet between shared networks.
    (Gitlab #1271)

* 367 [func] piotrek

    Added a possibilty to authorize more than one machine at once.
    User can select some machines and then authorize them with one click.
    Selection of all visible machines on the page is also possible.
    (Gitlab #1270)

* 366 [func] marcin

    Extended subnet edit form to allow pool-id specification.
    (Gitlab #1225)

* 365 [build] slawek

    Restored compatibility of the Stork hooks with Ubuntu 18.04 and Ubuntu
    20.04.
    (Gitlab #1254)

* 364 [bug] piotrek

    Fixed a bug in host reservation tab page where longer IPv6 address
    could overlap the reservation status information.
    Now, this tab is responsive and the overlap should not be visible
    no matter the screen width.
    (Gitlab #1260)

* 363 [sec] slawek

    Added the SHA256 digest to the RPM packages for compatibility with the FIPS
    mode of RedHat-like operating systems.
    (Gitlab #1171)

* 362 [bug] marcin

    Stork server enforces statistics updates in Kea after a subnet update.
    Applied small fixes in the subnet form validation.
    (Gitlab #1259)

* 361 [bug] piotrek

    Fixed a bug where after clicking the Home breadcrumb anchor, Home page
    was opened in a new tab. Now, it will open in the same tab.
    (Gitlab #1248)

* 360 [bug] piotrek

    Fixed a bug in hosts reservation filtering.
    Now, whenever new filter is applied, pagination is reset
    by default to the first page of the filtered results.
    (Gitlab #917)

* 359 [func] piotrek

    Improved Kea Apps tab view to be fully responsive and
    look good on all types of devices and screen resolutions.
    (Gitlab #1237)

* 358 [bug] slawek

    Fixed the simulator that stopped working due to missing explicitly provided
    dependency.
    (Gitlab #1257)

Stork 1.14.0 released on 2023-12-06.

* 357 [bug] piotrek

    Corrected wrong anchors to Kea hooks ARM docs in
    Kea application page.
    For Kea version >= 2.4 new std-ischooklib-* anchors are
    used.
    (Gitlab #915)

* 356 [doc] slawek

    Added the hook framework documentation.
    (Gitlab #779)

* 355 [bug] slawek

    Restored compatibility of the Stork binaries with Ubuntu 18.04 and Ubuntu
    20.04.
    (Gitlab #1201)

* 354 [sec] marcin

    Upgraded some UI and backend dependencies to remove critical
    vulnerabilities for Stork 1.14.0.
    (Gitlab #1233)

* 353 [bug] slawek

    Fixed issues with specifying the environment file location with the
    --use-env CLI flag. Removed unused --prometheus-bind9-exporter-interval CLI
    flag.
    (Gitlab #1219)

* 352 [build] slawek

    Enhanced the build system to recognize ARM64 architecture on the OpenBSD
    operating system.
    (Gitlab #1231)

* 351 [sec] slawek

    Disabled the TLS 1.0 and 1.1 protocols in the GRPC server of the Stork
    agent. The Stork server communicates with the Stork agent over TLS 1.3 by
    default.
    (Gitlab #1197)

* 350 [func] marcin

    Extended subnet form for specifying relay addresses.
    (Gitlab #1230)

* 349 [build] slawek

    Updated the simulator Python dependencies.
    (Gitlab #1166)

* 348 [func] marcin

    Added a button to manually trigger updating Kea configurations in the
    Stork database from the Kea servers.
    (Gitlab #1206)

* 347 [func] marcin

    Enabled subnet form with the pool management.
    (Gitlab #1208)

* 346 [func] marcin

    Extended the settings page with the ability to specify custom apps
    state puller and metrics collector intervals.
    (Gitlab #1210)

* 345 [doc] marcin

    Documented register command arguments in the stork-agent man file.
    (Gitlab #1158)

* 344 [bug] marcin

    Corrected allowed ranges for T1 Percent and T2 Percent parameters in
    the subnet edit form. Corrected initializing the Valid Lifetime
    parameter value in the subnet form.
    (Gitlab #1195)

* 343 [bug] marcin

    The agent recognizes a new alias output-options introduced in Kea 2.5
    logger configuration.
    (Gitlab #1198)

* 342 [build] slawek

    Changed the convention of preparing the ChangeLog file to avoid merge
    request conflicts.
    (Gitlab #1120)

* 341 [build] slawek

    Fixed the cross-compilation problem for 32-bit ARM architectures caused by
    invalid architecture labels.
    (Gitlab #1169)

Stork 1.13.0 released on 2023-10-11.

* 340 [build] slawek

    Added Rake and CI tasks to build, test, package, and upload the Stork
    hooks.
    (Gitlab #1178)

* 339 [build] marcin

    Downgraded PrimeNG to avoid the bug with dynamic tab menu. See PrimeNG
    issue #13609.
    (Gitlab #1176)

* 338 [func] slawek

    Support for a new hook naming convention. The server-specific hooks should
    use the "stork-server-" prefix (e.g., stork-server-ldap.so).
    (Gitlab #1180)

* 337 [bug] slawek

    The agent no longer includes the TLS credentials in the requests sent to
    the server. Including them caused TLS verification errors during agent
    re-registration.
    (Gitlab #1154)

* 336 [bug] slawek

    Fix displaying statistics of the IPv4 shared networks.
    (Gitlab #1135)

* 335 [func] marcin

    Implemented a form for updating selected subnet parameters.
    (Gitlab #957, #958)

* 334 [bug] slawek

    Fixed the minor filtration issues on hosts, subnets and shared networks
    pages.
    (Gitlab #1140)

* 333 [func] slawek

    Added support for BIND 9 running in the chroot mode.
    (Gitlab #974)

* 332 [func] slawek

    Added support for Postgres 15+. System tests now run on Postgres 16.
    Updated the docs to recommend granting all privileges on the public schema
    for the Stork user to avoid problems with some Postgres 15 (and above)
    installations.
    (Gitlab #1148)

* 331 [build] slawek

    Fixed the security vulnerabilities reported by the Github Dependabot and
    updated dependencies including the Go 1.21, Angular 16, PrimeNG, GoSwagger,
    OpenAPI Generator and several Python and Ruby packages.
    (Gitlab #1142)

* 330 [bug] slawek

    Fixed the problem of missing issues count in the configuration review panel
    header when the number of issues was zero. It resulted in a confusing
    message suggesting that some issues were found.
    (Gitlab #1131, #1141)

* 329 [build] slawek

    Updated the dependency versions used in the CI.
    (Gitlab #689)

* 328 [bug] razvan

    Fix alignment of some UI components (spinner, help tip component).
    (Gitlab #1014)

* 327 [func] slawek

    Links to IPv6 shared networks on the dashboard.
    (Gitlab #1133)

* 326 [build] slawek

    Added support for building Stork components on the ARM architecture. Added
    support for an experimental cross-compilation for arbitrary platforms.
    (Gitlab #381, #472, #893, #1113)

* 325 [doc] slawek

    Removed the documentation section referring to the non-existing Alpine
    script on CloudSmith.
    (Gitlab #1137)

* 324 [func] slawek

    Added support for configuring the Stork server hooks.
    (Gitlab #638)

* 323 [build] razvan

    Updated Kea version to 2.4.0 in demo and system tests.
    (Gitlab #995)

Stork 1.12.0 released on 2023-08-02.

* 322 [func] slawek

    Added a configuration option to handle requests directed to URL
    subdirectory (instead of URL root). Included an example Apache configuration
    file.
    (Gitlab #1039)

* 321 [build] slawek

    Refactored and formatted to the Python codebase to reach the highest
    quality rate from the linter.
    (Gitlab #1044)

* 320 [bug] slawek

    Fixed a bug in the script installing the Stork agent using a server token.
    This bug caused installation errors due to wrong paths to the deployed
    packages. The script no longer requires all Stork agent package formats to
    be deployed in the Stork server, and it will use the available formats for
    the matching operating systems where the Stork agent is installed.
    (Gitlab #932)

* 319 [func] marcin

    Created shared network view in the WebUI that contains shared network
    details, including DHCP parameters and options.
    (Gitlab #1119)

* 318 [func] slawek

    Added a new checker to verify if the Kea Control Agent configuration
    includes the control sockets. Added a descriptive message about possibly
    missing control socket configuration of the Kea DHCP daemons.
    (Gitlab #1045)

* 317 [bug] slawek

    Fixed bug with detecting RNDC key if the -c flag is not used to set the
    config path in BIND 9.
    (Gitlab #1057)

* 316 [func] slawek

    Refactored the log messages produced on the Stork agent initialization to be
    more straightforward and user-friendly.
    (Gitlab #1056)

* 315 [bug] ! slawek

    Enabled gzip compression for all communication between the Stork server and
    agent. It fixes handling a big response of named statistics endpoint.
    Stork accepts payloads (i.e., responses from Kea and BIND 9 endpoints) up
    to 40MiB uncompressed size.
    (Gitlab #1059)

* 314 [bug] slawek

    Fixed a UI problem that caused the IPv6 subnet bars to be unreadable for
    long addresses.
    (Gitlab #1060)

* 313 [bug] slawek

    Fixed a rare crash occurring when the state puller schedules the config
    review for an existing daemon. Set the focus on the login page to the first
    input. Shrank the width of the lease user context viewer to its content.
    Fixed a problem with the DHCP identifier button's width.
    (Gitlab #1053)

* 312 [build] slawek

    Refactored mock file suffixes. The mocks no longer need to be manually
    registered in the Rake file.
    (Gitlab #1006)

* 311 [func] marcin

    Subnet view contains DHCP parameters and options.
    (Gitlab #953)

* 310 [func] marcin

    Filter host reservations by subnet ID.
    (Gitlab #1058)

* 309 [func] marcin

    A tab with the subnet details is opened after clicking on the subnet.
    (Gitlab #931)

* 308 [doc] tomek

    The Stork Developer's Guide is now a stand-alone document. You can
    build it using rake `build:devguide` command.
    (Gitlab #786)

* 307 [bug] marcin

    Fixed the simulator in Stork demo.
    (Gitlab #1054)

Stork 1.11.0 released on 2023-06-07.

* 306 [bug] slawek

    Fixed propagating database password to the session manager.
    (Gitlab #1018)

* 305 [bug] slawek

    Fixed a problem with improperly using the rndc command that prevented
    detecting BIND 9 daemons if the RNDC key was specified outside the default
    RNDC key file. Fixed a rare bug when the Stork server crashes if an error
    occurs on a particular stage of BIND 9 detection.
    (Gitlab #1031)

* 304 [bug] slawek

    Fixed a bug in handling the empty database host that caused user login
    failures in the UI. Fixed rejecting the database usernames containing a
    dash. Fixed printing of the error messages related to the database
    authentication problems.
    (Gitlab #1022)

* 303 [func] razvan

    The lease search page can now display the content of the lease user
    context.
    (Gitlab #999)

* 302 [bug] marcin

    Fixed a bug in the Stork server that resulted in temporarily holding wrong
    DHCP options for an edited host reservation. This problem occurred when
    different options were specified for different servers owning the same
    host reservation. In addition, fixed a similar bug in specifying different
    client classes for different servers.
    (Gitlab #1017)

* 301 [bug] slawek

    Stork resolves now the include statements in the Kea configuration with any
    file extension.
    (Gitlab #1036)

* 300 [bug] slawek

    Fixed a bug in one of the config review checkers that caused a server crash
    if the Kea configuration contained a subnet prefix with a zero mask.
    (Gitlab #1024)

* 299 [bug] marcin

    Fixed a bug in the Stork server that could sometimes cause a modification
    of the timestamps of various records in the database during the update.
    This bug is unlikely to have a real impact on the users.
    (Gitlab #1007)

Stork 1.10.0 released on 2023-04-05.

* 298 [func] marcin

    New and updated host reservations are instantly visible in Stork after
    submitting the form.
    (Gitlab #996)

* 297 [build] slawek

    Fixed the security vulnerabilities reported by the Github Dependabot and
    updated dependencies including the Angular, PrimeNG, GoSwagger and
    OpenAPI Generator.
    (Gitlab #981)

* 296 [bug] slawek

    Fixed the path traversal vulnerability that allowed everyone to check the
    existence of any file on the filesystem.
    (Gitlab #987)

* 295 [bug] slawek

    Fixed fetching the authorization keys from BIND 9 configuration. The key
    value is visible only for super-administrators.
    (Gitlab #997)

* 294 [build] slawek

    Changed the executable paths configured in the default SystemD service
    files to absolute.
    (Gitlab #972)

* 293 [bug] slawek

    Fixed a problem whereby a user not assigned to any groups could not log out.
    (Gitlab #1004)

* 292 [func] slawek

    Added the configuration review checker to verify that the Stork Agent and
    Kea Control Agent communicate over TLS if the Kea Control Agent requires
    the HTTP credentials (i.e., Basic Auth).
    (Gitlab #945)

* 291 [build] slawek

    Upgrade the docker compose used in demo and system tests to V2 version.
    The V1 version is still supported for backward compatibility.
    (Gitlab #979)

* 290 [func] slawek

    Added support for connecting to the Postgres server over sockets. It allows
    securing the connection using the "trust" and "host" authentication modes.
    (Gitlab #858)

* 289 [bug] slawek

    Fixed ignoring URL segments in the Grafana base address.
    (Gitlab #980)

* 288 [bug] razvan

    The content of subnets column is now sorted.
    (Gitlab #855)

* 287 [func] slawek

    Added a human-readable representation of the event level in the dump
    package.
    (Gitlab #971)

* 286 [func] marcin

    Refactored the code pertaining to processing the Kea configuration in the
    Stork server. It introduces no new user-visible functionality, but the
    number of code changes is significant and thus noted in the ChangeLog.
    (Gitlab #942)

* 285 [bug] tomek

    BIND 9 detection code has been expanded and is now more robust. It now can
    also attempt to look at more default locations for config files, use
    named -V to discover built-in locations and also use STORK_AGENT_BIND9_CONFIG
    to explicitly tell where to look for a BIND9 config file. The detection
    process is also now more verbose. Enabling DEBUG logging level may
    help.
    (Gitlab #831)

* 284 [func] slawek

    The Prometheus exporter no longer attempts to communicate with
    non-configured Kea servers. It avoids producing repetitive error logs in
    the Kea Control Agent and the Stork Agent.
    (Gitlab #933)

* 283 [bug] slawek

    Fixed a problem with periodically showing the HA loading indicator when
    High Availability was not configured.
    (Gitlab #969)

* 282 [bug] slawek

    Fixed the problem with displaying subnet utilization bars on the shared
    network page. The bars for IA_NA and IA_PD were always shown, even when
    they had no corresponding subnet pools.
    (Gitlab #970)

* 281 [func] slawek

    Added a preliminary implementation of the hook framework.
    (Gitlab #779)

* 280 [func] slawek

    Implemented a new Kea configuration checker to detect if the subnet
    commands hook is simultaneously used with the configuration backend
    database and suggest replacing it with the configuration backend command
    hook.
    (Gitlab #940)

* 279 [func] slawek

    Added the Kea configuration checkers reporting when there are static
    reservations for all addresses or delegated prefixes in the pools.
    (Gitlab #941)

* 278 [func] slawek

    Added the configuration review checkers to detect common misconfigurations
    related to the HA multi-threading mode. The first checker suggests enabling
    the HA+MT if Kea uses multi-threading, and the second validates that HA
    peers use dedicated ports rather than Kea Control Agent's port when the
    dedicated listeners are enabled.
    (Gitlab #944)

Stork 1.9.0 released on 2023-02-01.

* 277 [bug] slawek

    Fixed a bug that prevented the machines from deleting.
    (Gitlab #928)

* 276 [func] razvan

    Added functionality for deleting users. A super-admin cannot remove
    its own account and the last super-admin user can not be removed.
    (Gitlab #117)

* 275 [bug] slawek

    Fixed the security vulnerabilities reported by the Github Dependabot and
    updated some dependencies. Added some Rake tasks for updating the project
    dependencies. Upgraded the target JavaScript version of the
    output UI bundle to be ES2020 standard-compliant.
    (Gitlab #934)

* 274 [bug] slawek

    Fixed detecting changes in the subnets' configuration. Stork now recognizes
    modifications of the subnet's client class, address pools, and delegated
    prefix pools.
    (Gitlab #927)

* 273 [func] marcin

    DHCP option form automatically shows controls suitable for the selected
    standard option definition. Previously, a user had to know the option
    format and manually add option field controls.
    (Gitlab #937)

* 272 [func] tomek

    Stork agent, server and tool now have logging levels configurable using
    STORK_LOG_LEVEL environment variable. The allowed values are:
    DEBUG, INFO, WARN, ERROR.
    (Gitlab #870)

* 271 [bug] slawek

    Added a waiting indicator presented while loading the HA servers' statuses.
    Previously, an inappropriate message about missing HA configuration was
    displayed.
    (Gitlab #448)

* 270 [func] slawek

    Extended the UI to display the delegated prefix pools and their
    utilizations.
    (Gitlab #186)

* 269 [build] slawek

    Added missing comments for all exported functions and enabled a linter
    rule to make them mandatory.
    (Gitlab #906)

* 268 [func] slawek

    The subnet ID presents a value from the Kea configuration instead of
    the internal Stork database ID.
    (Gitlab #376)

* 267 [build] slawek

    Added govulncheck to the Stork build system, a tool to detect security
    vulnerabilities on the backend.
    (Gitlab #861)

* 266 [bug] slawek

    The events pertaining to the daemons and apps can be filtered by the
    machines running these daemons and apps. Previously, filtering by a
    specific machine or an app did not return some events related to daemons.
    (Gitlab #882)

* 265 [bug] slawek

    Fixed the database migration that interrupted the schema upgrade if the
    database contained any shared networks.
    (Gitlab #196)

* 264 [func] marcin

    Added configuration of the DHCPv4 siaddr, sname, and file fields in the
    Kea host reservations.
    (Gitlab #911)

* 263 [build] andrei

    Added the lint:shell rake task which calls shellcheck on all shell scripts.
    Added a CI step that calls this linter. Improved the shell scripts to
    appease the shellcheck warnings.
    (Gitlab #876)

* 262 [bug] slawek

    Removed duplicated statuses of daemons on the machine page when the
    machine runs multiple applications.
    (Gitlab #900)

* 261 [build] slawek

    Fixed the missing man pages for Stork programs installed by packages.
    (Gitlab #913)

Stork 1.8.0 released on 2022-12-07.

* 260 [build] slawek

    Improved the system dependencies detection. The build system searches now
    in PATH instead of relying on fixed paths.
    (Gitlab #821)

* 259 [func] slawek

    Upgraded Angular to version 14.2.10.
    (Gitlab #903)

* 258 [func] marcin

    Stork server emits an event when it is reloaded with the SIGHUP signal.
    (Gitlab #878)

* 257 [func] slawek

    The configuration review checkers generate the reports even if they find no
    issues. A user can see all checkers executed for a daemon.
    (Gitlab #816)

* 256 [func] marcin

    Client classes can be specified in the host reservation form.
    (Gitlab #884)

* 255 [func] slawek

    Extended Kea Prometheus exporter to handle counters with summarized lease
    statistics.
    (Gitlab #839)

* 254 [func] slawek

    Added a command line switch to the Stork programs to specify that they
    should read the environment variables from the environment files to
    configure. Previously, they could only use the environment files when
    started via systemd. By default, the environment files are located in
    /etc/stork. The new command line switch allows for reading the files from
    a custom location.
    (Gitlab #830)

* 253 [build] slawek

    Extended system tests of host reservations.
    (Gitlab #836)

* 252 [func] slawek

    Fixed a problem resetting pagination while browsing the resources (subnets,
    shared networks, machines, applications) pages.
    (Gitlab #881)

* 251 [func] andrei

    The Kea app page now shows the name of the hook libraries instead of their
    entire path. Each item can be clicked to reveal the full path. Additionally,
    a link to each hook's specific documentation is available inline.
    (Gitlab #142)

* 250 [build] slawek

    Upgraded go-related dependencies.
    (Gitlab #883)

* 249 [bug] andrei

    Added breadcrumbs to the events page.
    (Gitlab #541)

Stork 1.7.0 released on 2022-10-12.

* 248 [bug] slawek

    Fixed a bug continuously producing false disconnect and reconnect events
    when Kea used a host backend without support for listing (e.g., RADIUS).
    (Gitlab #817)

* 247 [bug] slawek

    Fixed a bug in the post-install hook when Stork was installed from the
    packages on Debian with busybox. The package hooks now treat SystemD as
    an optional dependency. The Stork user is joined to the Kea group by
    default. Added installation CI checks for all package types.
    (Gitlab #749, #860, #867, #869)

* 246 [func] marcin

    DHCP high availability state is included in the troubleshooting dump
    tarball.
    (Gitlab #819)

* 245 [func] marcin

    Stork server and agent reload upon receiving the SIGHUP signal.
    (Gitlab #703)

* 244 [bug] marcin

    Fixed a bug on the application page causing wrong daemon tab selection.
    (Gitlab #772)

* 243 [doc] marcin

    Described the configuration review usage in the ARM.
    (Gitlab #847)

* 242 [bug] marcin

    Fixed a bug in the Stork server causing stale HA services when the HA
    configuration has been updated in Kea. It could sometimes result in
    showing outdated HA status on the dashboard page.
    (Gitlab #818)

* 241 [build] slawek

    Integrated Storybook framework with the Stork build system for faster and
    easier UI development.
    (Gitlab #845)

* 240 [func] marcin

    Allow three levels of DHCP options encapsulation when a new host
    reservation is created or an existing host reservation is updated.
    Preliminary support for standard DHCP option definitions has been added.
    A few standard DHCPv6 option definitions have been defined in Stork,
    allowing to recognize option fields returned by the Kea servers
    accurately.
    (Gitlab #837)

Stork 1.6.0 released on 2022-09-07.

* 239 [build] slawek

    Upgraded Angular to version 14.2.0.
    (Gitlab #849)

* 238 [bug] slawek

    Prometheus exporter for BIND 9 sends the HTTP requests with an empty body.
    BIND 9.18 periodically rejects requests that include a body, causing errors
    in gathering the statistics.
    (Gitlab #798)

* 237 [func] slawek

    Added the possibility to enable or disable the configuration review
    checkers globally or only for selected daemons.
    (Gitlab #610)

* 236 [build] slawek

    Refactored the FreeBSD and OpenBSD support for building agent packages.
    (Gitlab #193)

* 235 [func] marcin

    Enable updating host reservations in Stork.
    (Gitlab #838)

* 234 [build] slawek

    Added scripts for building native APK packages with Stork Server and
    Stork Agent. They are prepared for Alpine 3.15.
    (Gitlab #736)

* 233 [func] slawek

    Added two Kea configuration checkers. The first one finds the overlapping
    subnets based on the subnet prefixes. The second checker validates the
    subnet prefixes.
    (Gitlab #763)

* 232 [func] marcin

    Display DHCP options in the host reservation view.
    (Gitlab #827)

* 231 [func] slawek

    Added a system test validating Stork upgrade to the most recent version
    using Cloudsmith repository.
    (Gitlab #746)

* 230 [bug] slawek

    Fixed a problem with running the demo by shell script on macOS.
    (Gitlab #824)

* 229 [bug] slawek

    Corrected a bug in the host puller, causing the Stork Server not to notice
    an update in the reserved hostnames in Kea configuration when neither an
    IP address nor DHCP identifier has been changed.
    (Gitlab #814)

* 228 [build] marcin

    Migrated to PrimeFlex 3.0.1.
    (Gitlab #731)

Stork 1.5.0 released on 2022-07-13.

* 227 [bug] slawek

    Fixed the security vulnerabilities reported by the Github Dependabot.
    (Gitlab #805)

* 226 [func] slawek

    Added a shell script to run the demo only with docker-compose. The users
    can try the Stork with minimal effort.
    (Gitlab #761)

* 225 [func] marcin

    Added multiple enhancements in the form for creating new host
    reservations. The form now checks if the specified IP addresses belong
    to the selected subnet. The maximum size of the DHCP identifier is
    limited to 40 hexadecimal digits. Host reservations list can be refreshed
    with a button above the list.
    (Gitlab #728)

* 224 [doc] marcin

    Documented specification of the DHCP options with host reservations and
    how to delete a host reservation.
    (Gitlab #794)

* 223 [bug] slawek

    Fixed lease utilization statistics calculations for the HA pairs. The
    statistics of the assigned addresses and delegated prefixes are no longer
    doubled. Only the active server's leases are counted.
    (Gitlab #710)

* 222 [build] slawek

    Introduced golang 1.18 and upgraded go-related dependencies.
    (Gitlab #788)

* 221 [func] marcin

    Added an ability to specify DHCP options in the host reservations.
    (Gitlab #725)

* 220 [func] marcin

    Enabled deleting host reservations from the Kea servers running host_cmds
    hook library.
    (Gitlab #785)

* 219 [func] slawek

    Refactored the system tests framework to use Docker instead of LXD. The
    tests are more straightforward, readable, stable, faster, and require less
    disk space.
    (Gitlab #709)

* 218 [bug] slawek

    Ruby dependencies and their structure are now explicitly specified to
    ensure that identical versions are used in all environments to avoid
    problems with incompatible packages.
    (Gitlab #781)

* 217 [bug] slawek

    Changed the permissions of the systemD service files used in DEB and RPM
    packages. They are now more restricted and don't produce the writable
    warning.
    (Gitlab #783)

* 216 [bug] slawek

    Fixed non-visible navbar menus on small resolutions.
    (Gitlab #698)

* 215 [doc] slawek

    Changed the bikesheding font color to black, i.e. the same as all "normal"
    text, to unify with other projects' documentation styles. This has best
    contrast and is way less distracting than it used to be.
    (Gitlab #782)

* 214 [bug] marcin

    Fixed Stork binaries build date injection. Before this fix, the binaries
    reported unset build date.
    (Gitlab #744)

* 213 [bug] marcin

    Fixed broken password strength indicators.
    (Gitlab #740)

Stork 1.4.0 released on 2022-06-01.

* 212 [bug] slawek

    Corrected a bug, which resulted in returning a null value instead of a
    list of events in a machine dump tarball.
    (Gitlab #743)

* 211 [bug] slawek

    The Stork server no longer sends statistics queries to the Kea
    servers not using the stat_cmds hooks library. Sending such
    queries caused unnecessary commands processing by the Kea
    servers and excessive error logs in Stork.
    (Gitlab #742)

* 210 [bug] slawek

    Fixed the rake tasks that ran the database in a Docker container but were
    connecting to the database on localhost.
    (Gitlab #733)

* 209 [func] slawek

    Log a warning message when no monitored application is detected.
    (Gitlab #713)

Stork 1.3.0 released on 2022-05-11.

* 208 [doc] marcin

    Added "Creating Host Reservations" section in the ARM.
    (Gitlab #729)

* 207 [func] slawek

    Refactored the Rakefile responsible for running the Stork demo. The demo
    builds faster, uses the Docker layer caching, and produces smaller images.
    (Gitlab #709)

* 206 [bug] marcin

    Disable escaping special characters in the machine dumps. It improves
    the dumps' readability.
    (Gitlab #665)

* 205 [func] marcin

    It is now possible to create host reservations in Kea using a
    form on the host reservations page.
    (Gitlab #717, #720)

* 204 [build] kpfleming

    Enable automatic Stork services restart with systemd on failure.
    (Gitlab #721)

* 203 [doc] sgoldlust

    Editorial and grammatical corrections in docs, logs, and code comments.
    (Gitlab #718)

* 202 [func] slawek

    Refactored the Rakefile responsible for building, installing, and testing
    Stork. It results in changes to the usage of some rake tasks and the
    specification of their arguments.
    (Gitlab #709)

Stork 1.2.0 released on 2022-03-02.

* 201 [bug] slawek

    Prevent issues with parsing and storing large Kea
    configurations in the Stork database
    (Gitlab #682)

* 200 [bug] marcin

    stork-tool db-create command creates the pgcrypto extension.
    (Gitlab #701)

* 199 [build] marcin

    Updated Angular to version 13.2.2 and PrimeNG to version 13.1.0.
    (Gitlab #690)

Stork 1.1.0 released on 2022-02-02.

* 198 [bug] slawek

    Removed the underline on hover the Stork logo on the login page.
    (Gitlab #669)

* 197 [bug] slawek

    Fix the utilization calculations for address and delegated
    prefix statistics returned by the Kea DHCP servers. Stork
    corrects the calculations by taking into account the
    out-of-pool reservations that are not returned by the
    DHCP servers in the statistics.
    (Gitlab #560)

* 196 [func] marcin

    Improved Stork Server performance while it gathers host reservations
    from the Kea servers. The server only updates the host reservations
    in its database when it detects at least one change in the gathered
    reservations. Moreover, it runs Kea configuration reviews only when
    it detects at least one host reservation change.
    (Gitlab #681)

* 195 [bug] marcin

    Renamed --token command line flag of the stork-agent to --server-token.
    (Gitlab #625)

* 194 [func] marcin

    Display Kea and Bind access points in the app tab.
    (Gitlab #586)

* 193 [doc] sgoldust
    Stork documentation review.
    (Gitlab #646)

* 192 [func] slawek

    Support for the large statistic values. It handles
    address/NA/PD counters for IPv6 subnets, large shared
    networks, and globals. It fixed the problem when these
    counters exceeded the int64 range. Prepared the Stork
    for large statistic values from Kea.
    (Gitlab #670)

* 191 [func] marcin

    stork-tool facilitates db-create and db-password-gen commands
    to conveniently create PostgreSQL database for the Stork Server.
    (Gitlab #620)

* 190 [func] slawek

    Added a flag to the Stork Agent to disable collecting
    per subnet stats from Kea. It allows limiting data shared
    with Prometheus/Grafana.
    (Gitlab #614)

* 189 [func] marcin

    Various DHCP client identifiers (e.g. flex-id, circuit-id) can
    be displayed in a textual format and toggled between this and
    the hexadecimal format. It is now also possible to search host
    reservations by an identifier using the textual format.
    (Gitlab #639)

* 188 [bug] slawek

    Fixed the UI overlapping when subnet utilization is greater than 100%.
    It improves the usability of the dashboard until the utilization
    problems are finally resolved.
    (Gitlab #560)

* 187 [func] marcin

    Config reviews are scheduled automatically after getting updated
    host reservations via the host_cmds hooks library.
    (Gitlab #680)

* 186 [func] slawek

    Implemented the support for Kea and Kea CA configuration
    with comments. Stork (Server or Agent) can parse the JSONs
    with C-Style single-line and block comments, and single-line
    comments started with hash.
    (Gitlab #264)

* 185 [build] marcin

    Upgraded go-pg package from version 9 to version 10.
    (Gitlab #678)

* 184 [func] marcin

    Implemented new Kea configuration checkers. The first one
    verifies if there are any empty shared networks or networks
    with a single subnet. The second one verifies if there are
    subnets containing no pools and no host reservations. The
    third one verifies if there are subnets containing only out
    of the pool addresses and delegated prefixes, in which case
    the out-of-pool host reservation mode can be enabled.
    (Gitlab #672)

* 183 [build] marcin

    Stork now uses golang 1.17 and protoc-gen-go 1.26.
    (Gitlab #652)

* 182 [bug] marcin

    Fixed a bug in the host reservation database model. The bug caused
    issues with updating a host reservation when it contained no IP
    addresses.
    (Gitlab #677)

* 181 [bug] marcin

    Corrected an issue in the database migrations which caused the
    Stork 1.0.0 server to fail to start when the database contained
    host reservations.
    (Gitlab #676)

Stork 1.0.0 released on 2021-12-08.

* 180 [doc] tomek

    Prometheus and Grafana alerting mechanisms are described briefly.
    (Gitlab #600)

* 179 [build] slawek

    Renamed Stork Agent configuration variable STORK_AGENT_ADDRESS
    to STORK_AGENT_HOST. This change requires modifications in the
    existing agent.env files.
    (Gitlab #641)

* 178 [doc] slawek

    Extended the comments in the files with environment variables.
    Unified them with the man pages.
    (Gitlab #632)

* 177 [bug] slawek

    Stork calculates properly the subnet, shared network,
    and global utilizations. Fixed the problem with showing
    more used addresses than available.
    (Gitlab #560)

* 176 [doc] slawek

    Added the troubleshooting section in the documentation.
    It contains some hints on how to resolve the agent-related
    issues.
    (Gitlab #475)

* 175 [doc] slawek

    Renamed STORK_ENABLE_METRICS server environment variable
    to STORK_SERVER_ENABLE_METRICS.
    (Gitlab #621)

* 174 [bug] slawek

    Ensured that the agent registration over IPv6 works correctly
    excluding link-local scope.
    (Gitlab #447)

* 173 [build] marcin

    Upgraded UI to use Angular and Primeng 13.
    (Gitlab #606)

* 172 [bug] marcin

    Fixed a bug in the Stork server, which caused stale subnets,
    hosts, and shared networks after reconfiguring a monitored
    Kea server.
    (Gitlab #473)

* 171 [func] slawek

    Dump machine configuration to file. After pressing the button
    in the UI, all data related to a specific machine (database entries,
    configurations, logs) are packed into a single tarball. Next,
    they can be easily shared with technical support (e.g. as an
    email attachment).
    (Gitlab #43)

* 170 [func] marcin

    Kea configuration review can now be requested from the Kea
    daemon tab in the UI. In addition, the server automatically
    re-reviews the configurations whenever new configuration
    checkers are available in the new Stork releases.
    (Gitlab #609)

* 169 [func] marcin

    Server database connection can be protected with TLS.
    (Gitlab #403)

Stork 0.22.0 released on 2021-11-05.

* 168 [func] slawek

    The Stork Agent support for the Basic Authentication introduced
    in Kea 1.9.0. User can define the credentials used to
    establish connection with Kea CA.
    (Gitlab #347)

* 167 [func] slawek

    The Stork Server has now the ability to export metrics to
    Prometheus. It reports the machine states and pool utilization.
    (Gitlab #576)

* 166 [bug] slawek

    Fixed the problem with too many log messages about
    updating a machine state. Stork doesn't report that
    a machine was updated anymore.
    (Gitlab #595)

* 165 [func] slawek

    The Stork Agent reports the metrics to Prometheus with the
    subnet prefix instead of sequential ID if the subnet_cmds
    is installed.
    (Gitlab #574)

* 164 [func] marcin

    Implemented Kea configuration review mechanism. It runs checks
    on the Kea server configurations, and displays found issues in
    the Kea daemon tabs.
    (Gitlab #461)

* 163 [func] marcin

    Implemented host_cmds_presence configuration checker. It
    verifies if the host_cmds hooks library is loaded when hosts
    backends are used.
    (Gitlab #601)

* 162 [build] andrei

    Rebuilt CI image to upgrade openssl and renew certificate
    following the LetsEncrypt root certificate expiration on
    the 30th of September 2021. The CI image now also has the psql
    client preinstalled.
    (Gitlab #596)

Stork 0.21.0 released on 2021-10-06.

* 161 [func] slawek

    The Stork Agent now supports communication with Kea over TLS.
    It automatically detects if the Kea Control Agent is configured
    to use TLS.
    (Gitlab #527)

* 160 [build] slawek

    Fix failed pipeline issues - bump CentOS version and related
    packages, change some unit tests to avoid crashes in CI
    environment.
    (Gitlab #552)

* 159 [bug] slawek

    Eliminated memory leaks from the Stork Web UI.
    (Gitlab #105)

Stork 0.20.0 released on 2021-09-08.

* 158 [build] marcin

    Corrected issues with the nginx-server.conf, an example Nginx
    configuration file providing the reverse proxy setup for Stork.
    The proxy now correctly forwards calls to download the Stork
    Agent installation script. The updated configuration also
    allows for accurately determining the Stork server URL while
    generating the downloaded script.
    (Gitlab #557)

* 157 [func] godfryd, slawek

    Added cert-import command to stork-tool. This allows
    importing CA key and cert, and server key and cert.
    (Gitlab #570)

* 156 [build] marcin
    Running unit tests no longer requires specifying a password
    explicitly. Renamed database connection specific environment
    variables in the stork-tool.
    (Gitlab #555)

* 155 [func] slawek

    Added resolving the include statement in Kea configuration when
    an agent is detecting applications.
    (Gitlab #564)

* 154 [bug] slawek

    Prevent agent re-registration after its restart.
    (Gitlab #528 and #558)

* 153 [bug] slawek

    Corrected an issue with fetching Stork events from the
    databases running on PostgreSQL 10. Also, the Stork
    server requires PostgreSQL version 10 or later.
    (Gitlab #571)

* 152 [build] marcin
    Resolved an issue with building Stork packages in Docker on
    the MacOS.
    (Gitlab #490)

* 151 [func] slawek

    Obfuscate sensitive Kea configuration parts displayed using
    JSON viewer.
    (Gitlab #561)

Stork 0.19.0 released on 2021-08-11.

* 150 [bug] godfryd

    Fixed reading STORK_AGENT_PROMETHEUS_KEA_EXPORTER_ADDRESS
    and STORK_AGENT_PROMETHEUS_BIND9_EXPORTER_ADDRESS environment
    variables in Stork Agent.
    (Gitlab #559)

* 149 [func] godfryd

    Upgraded Angular and PrimeNG to version 12.x.
    (Gitlab #405)

Stork 0.18.0 released on 2021-06-02.

* 148 [func] godfryd

    Added including Grafana dashboard JSON files to RPM
    and deb packages.
    (Gitlab #544)

* 147 [func] godfryd

    Added system tests for collecting statistics from
    various versions of Kea app.
    (Gitlab #439)

* 146 [func] marcin

    Added new tab displaying selected host reservation's details.
    The tab includes the information about the allocated leases
    for the reservation, e.g. if the reservation is in use by the
    client owning the reservation or there is a conflict (the lease
    is allocated to a different client). It also indicates whether
    the matching lease is declined, expired or there are no matching
    leases.
    (Gitlab #530)

* 145 [func] godfryd

    Refactored stork-db-migrate tool to stork-tool and added
    commands for certificates management.
    (Gitlab #515)

* 144 [func] slawek

    Added Kea daemon configuration preview in JSON format.
    (Gitlab #531)

* 143 [bug] ymartin-ovh

    Fixed honoring the listen-only flags in Stork Agent.
    (Gitlab #536)

* 142 [func] marcin

    Updated Stork demo to expose new features: leases search,
    Kea database backends and files locations.
    (Gitlab #542)

* 141 [func] godfryd

    Fixed and improved detecting various versions of BIND 9.
    (Gitlab #474)

Stork 0.17.0 released on 2021-05-07.

* 140 [func] godfryd

    Added displaying number of unauthorized machines on machines
    page in select button.
    (Gitlab #492)

* 139 [func] marcin

    New information presented in the Kea tab includes
    locations of a lease file, forensic logging file and the
    information about database backends used by the particular
    Kea instance.
    (Gitlab #299)

* 138 [func] marcin

    Implemented declined leases search.
    (Gitlab #510)

* 137 [func] godfryd

    Added system tests for new agent registration.
    (Gitlab #507)

Stork 0.16.0 released on 2021-04-07.

* 136 [func] marcin

    Implemented Leases Search.
    (Gitlab #509)

* 135 [func] godfryd

    Added Grafana dashboard for DHCPv6. Enabled generating
    DHCPv6 traffic in Stork Simulator. Adjusted Stork demo
    to handle DHCPv6 traffic.
    (Gitlab #176)

* 134 [bug] godfryd

    Fixed getting host address for listening in agent.
    (Gitlab #504)

Stork 0.15.0 released on 2021-03-03.

* 133 [doc] andrei

    Spell checks
    (Gitlab #497)

* 132 [doc] sgoldlust

    Updates to the Stork ARM.
    (Gitlab #476)

* 131 [ui] tomek

    Added tooltips for the Grafana links on the dashboard and
    subnets view.
    (Gitlab #470)

* 130 [func] marcin

    Added a dialog box in the UI to rename apps.
    (Gitlab #477)

* 129 [doc] godfryd, marcin

    Documented secure communication channel between the Stork Server
    and the agents in the ARM. The new agent installation and
    registration methods were described.
    (Gitlab #486)

* 128 [func] godfryd, marcin

    Updated Stork demo setup to use new machines registration methods.
    Machines automatically request registration using the agent token
    method. Their registration can be approved in the machines view.
    (Gitlab #485)

* 127 [func] godfryd, tomek, marcin

    Secured agent-server channel part 3. Implemented agent deployment
    using script downloaded from the server. The script installs
    deb/rpm packages with stork agent. Then the script registers
    current machine in the server performing key and certs
    exchange. Enabled TLS to gRPC traffic between agent and server
    using certs that are set up during agent registration. Added
    instruction on machines page how to install an agent. Added UI for
    presenting and regenerating server token.
    (Gitlab #483)

* 126 [func] godfryd

    This is the second part of secured agent-server channel
    implementation. Added code for registering a machine in the server
    and performing key and certs exchange but it is not used fully
    yet. Added server-token and agent-token based agent
    authorizations. Added REST API for presenting and regenerating
    server token, but it is not used in UI yet. Updated content of
    reference agent.env agent config file.
    (Gitlab #481)

* 125 [func] marcin

    Assign friendly names to the apps monitored in Stork. The apps'
    names are auto-generated using the following scheme:
    [app-type]@[machine-address]%[app-unique-id], e.g.,
    kea@machine1.example.org%123. The [app-unique-id] is not appended
    to the name of the first first app of the given type on the
    particular machine. Thus, the name can be kea@machine1.example.org.
    The auto-generated apps' names are presented in the Web UI instead
    of the previously used app ID. The names are not yet editable by a
    user.
    (Gitlab #458)

* 124 [func] godfryd

    The first part of secured agent-server channel implementation.
    Added generating root CA and server keys and certs,
    and server token generation during server startup.
    (Gitlab #479)

* 123 [bug] marcin

    Corrected an issue with refreshing the events list on the page
    displaying the machine information. Previously, when switched
    to a different tab, the events list could remain stale.
    (Gitlab #463)

* 122 [func] godfryd

    Migrated command line processing in agent from jessevdk/go-flags
    to urfave/cli/v2. Thanks to this it is possible to define commands
    in command line. Previously only switches were possible in command
    line. This is a preparation for new agent command: register.
    (Gitlab #468)

Stork 0.14.0 released on 2020-12-09.

* 121 [func] marcin

    Events received over SSE and presented on various Stork pages are
    now filtered and only the events appropriate for the current view
    are shown. Prior to this change all events were always shown.
    (Gitlab #429)

* 120 [func] marcin

    When Stork server pulls updated Kea configurations it detects those
    configurations that did not change since last update using a fast
    hashing algorithm. In case when there was no configuration change
    for a daemon, Stork skips processing subnets and/or hosts within
    this configuration. This improves efficiency of the configuration
    pull and update. In addition, when configuration change is detected,
    an event is displayed informing about such change in the web UI.
    (Gitlab #460)

* 119 [doc] tomek

    Prometheus and Grafana integration is now documented. Also, updated
    requirements section pointing out that stat_cmds hook is needed for
    Stork to show Kea statistics correctly.
    (Gitlab #433, #451)

* 118 [bug] marcin

    Prevent an issue whereby Stork server would attempt to fetch updated
    machine state while the request to add this machine is still being
    processed. This used to cause data conflict errors in the logs and
    network congestion.
    (Gitlab #446)

* 117 [build] marcin

    Upgraded Go from 1.13.5 to 1.15.5 and golangcilint from 1.21.0 to
    1.33.0.
    (Gitlab #457)

* 116 [perf] marcin

    Improved performance of connecting large Kea installation with many
    subnets to Stork. Adding subnets to the database is now much more
    efficient as it avoids extensive subnet lookups. Instead it uses
    indexing techniques.
    (Gitlab #421)

Stork 0.13.0 released on 2020-11-06.

* 115 [func] marcin

    Improved presentation of the HA server scopes. Added a help
    tip describing expected HA scopes in various cases.
    (Gitlab #387)

* 114 [bug] godfryd

    The links on the dashboard to subnets and shared networks have been
    adjusted so they take into account DHCP version. This way subnets and
    shared network pages automatically set filtering by protocol version
    based on parameters provided in URL.
    (Gitlab #389)

* 113 [bug] godfryd

    Fixed handling renamed statistics from Kea. In Kea 1.8 some
    of the statistics have been renamed, e.g. total-addresses
    to total-addresses. Now Stork supports both of the cases.
    (Gitlab #413)

* 112 [bug] godfryd

    Fixed handling situation when IP address of Kea Control Agent has
    changed. Till now Stork was not able to detect this and was still
    communicating to the old address. Now it checks if address has
    changed and updates it in the database.
    (Gitlab #409)

* 111 [bug] marcin

    Corrected presentation of the HA state in the dashboard and
    the HA status panel in cases when HA is enabled for a server
    but the HA state information was not fetched yet. In such
    cases a spinner icon and the 'fetching...' text is now
    presented.
    (Gitlab #277)

* 110 [bug] marcin

    The rake build_agent task now supports building the agent
    using wget versions older than 1.19. Prior to this change,
    the agent build was failing on Debian 9.
    (Gitlab #423)

* 109 [doc] tomek

    Updated Prerequisites section. We now have a single list of
    supported systems.
    (Gitlab #431)

* 108 [test] tomek, marcin

    Corrected and extended existing boilerplate WebUI unit tests.
    (Gitlab #164)

* 107 [bug] godfryd

    Fixed problem of adding Kea with 4500 subnets. Now messages
    with Kea configuration sent from Stork Agent to Stork Server
    are compressed so it is possible to sent huge configurations.
    Added new Kea instance to Stork demo with 7000 subnets.
    (Gitlab #398)

* 106 [doc] godfryd

    Added documentation for Stork system tests. The documentation
    describes how to setup environment for running test tests,
    how to run them and how to develop them.
    (Gitlab #427)

Stork 0.12.0 released on 2020-10-14.

* 105 [func] godfryd

    Added a new page with events table that allows filtering and
    paging events. Improved event tables on dashboard, machines and
    applications pages. Enabling and disabling monitoring now
    generates events.
    (Gitlab #380)

* 104 [bug] matthijs

    Stork was unable to parse inet_spec if there were multiple addresses
    in the 'allow' clause.  Also fix the same bug for 'keys'.
    (Gitlab #411)

* 103 [func] godfryd

    Introduced breadcrumb that shows current location in Stork
    web application.
    (Gitlab #337)

* 102 [func] tomek

    The stork-db-migrate tool can now migrate up and down to specific
    schema versions. The SQL tracing now works and can be used to
    export SQL schema to external file.
    (Gitlab #366)

Stork 0.11.0 released on 2020-09-04.

* 101 [func] godfryd

    Merged Stork DHCP Traffic Simulator and Stork DNS Traffic
    Simulator into one web application called Stork Environment
    Simulator. Added there capabilities for adding all present
    machines in demo setup and ability to stop and start Stork Agents,
    Kea and BIND 9 daemons. This allows simulation of communication
    issues between applications, Stork Agents and Stork Server.
    (Gitlab #380)

* 101 [func] marcin

    Restrict log viewer's access to the remote files. The log viewer
    can only access log files belonging to the monitored application.
    (Gitlab #348)

* 100 [func] godfryd

    Improved user experience of updating machine address/port.
    Improved visual aspects. Added refreshing state from the machine
    after changing the address.
    (Gitlab #283)

* 99 [func] godfryd

    The DHCP dashboard now is presenting only monitored daemons.
    The daemons that have monitoring switched off are not visible
    in the dashboard.
    (Gitlab #365)

* 98 [bug] marcin

    Corrected an issue causing false errors about broken communication
    with the monitored Kea application after the application was
    brought back online.
    (Gitlab #384)

* 97 [bug] godfryd

    Improved layout of various tables that they are displayed correctly
    on smaller screens. Fixed address of the machine that is displayed
    in the tables (previous it was always showing 127.0.0.1).
    (Gitlab #295)

* 96 [doc] matthijs

    Add documentation on monitoring the BIND 9 application.
    (Gitlab #382)

* 95 [func] godfryd

    Fixed presenting an application status on a machine tab
    with BIND 9 application. Previously it was always red/inactive.
    Now it is presented the same way as it is for Kea app: status
    per each daemon of an app.
    (Gitlab #379)

* 94 [bug] marcin

    Fixed an issue whereby the user was unable to login to Stork
    when database password contained upper case letters. In addition,
    passwords with spaces and quotes are now also supported.
    (Gitlab #361)

* 93 [func] marcin

    Login credentials are passed in the message body rather than as
    query parameters. In addition, the user information is obfuscated
    when db tracing is enabled.
    (Gitlab #375)

Stork 0.10.0 released on 2020-08-13.

* 92 [func] godfryd

    Improved presenting application status on machines page. Now,
    instead of summary app status, there are presented statuses for
    each daemon of given application.
    (Gitlab #297, #282)

* 91 [doc] tomek

    Update man pages and installation instructions.
    (Gitlab #202, #266, #307)

* 90 [ui] tomek

    Clarified machines page, added tooltips. Updated color scheme
    to improve readability of wide tables.
    (Gitlab #112, #293)

* 90 [bug] marcin

    Fixed an issue with refreshing log displayed within the log viewer.
    The issue was triggered by the periodic updates of the information
    about monitored apps. As a result of the updates the log file
    identifiers were changing which resulted in an error message
    informing that the viewed file no longer exists.
    (Gitlab #364)

* 89 [func] godfryd

    Changed md5 to blowfish as algorithm in hash function used to store
    password in PostgreSQL database.
    (Gitlab #356)

* 88 [bug] godfryd

    Fixed upgrading RPM agent and server packages. There was a problem
    of re-adding stork-agent and stork-server users that already exist
    in case of upgrade.
    (Gitlab #334)

* 87 [doc] marcin

    Described Kea log viewer in the ARM.
    (Gitlab #349)

* 86 [func] tmark

    Added tool tip to RPS columns on DHCP dashboard.
    (Gitlab #363)

* 85 [bug] marcin

    Fixed regression in the log viewer functionality which removed links
    to the log files on the Kea app pages. In addition, improved
    error message presentation on the log viewer pages.
    (Gitlab #359)

* 84 [func] godfryd

    Added stop/start monitoring button to better control which services
    are monitored. Communication failures now generate events that are
    recorded in the events system. Machine view now shows events.
    (Gitlab #324, #339)

* 83 [func] tmark

    Added RPS (Response Per Second) statistics to DHCP Dashboard
    (Gitlab #252)

* 82 [func] marcin

    Viewing the tail of the remote log files is enabled in the UI.
    (Gitlab #344)

* 81 [func] matthijs

    Add more query details to BIND 9 exporter and Grafana dashboard:
    queries by duration, which transport protocol is used, packet sizes.
    (Gitlab #63)

* 80 [func] marcin

    List of loggers used by Kea server is fetched and displayed in the
    Kea application tab.
    (Gitlab #342)

* 79 [ui] vicky, tomek, marcin

    Added explicit link to DHCP dashboard.
    (Gitlab #280)

* 78 [bug] godfryd

    Fixed crashes when empty requests were sent to ReST API endpoints
    for users and machines.
    (Gitlab #310, #311, #312)

Stork 0.9.0 released on 2020-07-01.

* 77 [bug] matthijs

    BIND 9 process collector would not be created if named process was
    started after Stork Agent.
    (Gitlab #325)

* 76 [func] marcin

    Pool utilization in the Stork dashboard is shown with a progress bar.
    (Gitlab #235)

* 75 [bug] matthijs

    Bind exporter did not unregister all Prometheus collectors on
    shutdown.
    (Gitlab #326)

* 74 [bug] marcin

    Fixed a security problem whereby an unlogged user had access to some
    restricted pages. If the unlogged user tries to access a restricted
    page, the user is redirected to the login page. If the user tries
    to access a page without proper privileges, the HTTP 403 page is
    displayed.
    (Gitlab #119)

* 73 [func] marcin

    Monitor communication issues between Stork and the applications.
    If there is a communication problem with any app it is highlighted
    via appropriate icon and a text that describes the problem. The
    server logs were adjusted to indicate if the communication issue
    is new or has been occurring for a longer period of time.
    (Gitlab #305)

* 72 [func] tomek

    Implemented version reporting in agent and server.
    (Gitlab #265)

Stork 0.8.0 released on 2020-06-10.

* 71 [bug] godfryd

    Prevent Stork Agent crashes encountered when unknown statistics
    was returned by Kea.
    (Gitlab #316)

* 70 [func] matthijs

    Implemented Bind exporter and embedded it in Stork Agent.
    It is based on bind_exporter:
    https://github.com/prometheus-community/bind_exporter
    (Gitlab #218)

* 69 [func] godfryd

    Implemented basic events mechanism. The events pertaining to
    machines, apps, daemons, subnets and other entities are displayed
    on the dashboard page. The server-sent events (SSE) mechanism is
    used by the browser to refresh the list of events.
    (Gitlab #275)

* 68 [func] marcin

    Display last failure detected by High Availability for a daemon.
    (Gitlab #308)

* 67 [func] marcin

    Hostname reservations are now fetched from Kea servers and displayed
    in the UI. It is also possible to filter hosts by hostname
    reservations.
    (Gitlab #303)

* 66 [bug] marcin

    Corrected a bug which caused presenting duplicated subnets when
    the subnets where filtered by text. This issue occurred when
    multiple pools belonging to a subnet were matched by the
    filtering text.
    (Gitlab #245)

* 65 [func] marcin

    Extended High Availability information is displayed for Kea
    versions 1.7.8 and later.
    (Gitlab #276)

* 64 [func] godfryd

    Changed the syntax for search expressions (`is:<flags>` and
    `not:<flag>`). E.g. `is:global` should be used instead of just
    `global`.
    (Gitlab #267)

* 63 [func] tmark

    Added --listen-prometheus-only and --listen-stork-only command line
    flags to stork-agent.
    (Gitlab #213)

Stork 0.7.0 released on 2020-05-08.

* 62 [func] marcin

    Global host reservations in Kea are shown in the UI.
    (Gitlab #263)

* 61 [func] godfryd

    Implemented global search. It allows for looking across different
    entity types.
    (Gitlab #256)

* 60 [func] marcin

    HA state is presented in the dashboard.
    (Gitlab #251)

* 59 [func] marcin

    The list of hosts now includes a tag indicating if the host
    has been specified in the Kea configuration file or a host
    database. In addition, a bug has been fixed which caused some
    hosts to be associated with more then one Kea app, even when
    only one of them actually had them configured.
    (Gitlab #246)

* 58 [func] godfryd

    Improved presenting Kea daemons on Kea app page. There have
    been added links to subnet, shared network and host reservations
    pages with filtering set to given app id.
    (Gitlab #241)

* 57 [bug] marcin

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
    Added preconfigured Prometheus & Grafana containers to
    docker-compose. Added Kea and BIND 9 Prometheus exporters.

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
    status monitoring with Kea. Added new sections describing
    applications management.
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

    Kea High Availability status is presented on the Kea application
    page.
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
    installation instructions regarding how to build and run Stork
    natively.
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

    Enabled sign-in/sign-out mechanism with HTTP sessions based on
    cookies. The default admin account has been created with default
    credentials.
    (Gitlab #22)

* 7 [func] godfryd

    Added initial implementation of the page which allows for adding new
    machines and listing them. The missing part of this implementation is
    the actual storage of the machines in the database. In addition, the
    agent has been extended to return a state of the machine.
    (Gitlab #23)

* 6 [func] godfryd

    Added initial implementation of Stork Agent. Implemented basic
    communication between Stork Agent and Stork Server using gRPC
    (Server initiates connection to Agent).
    (Gitlab #26)

* 5 [func] marcin

    Added stork-db-migrate tool to be used for migrating the database
    schema between versions and returning the current schema version
    number. Also, added basic schema with SQL tables holding system
    users and session information.
    (Gitlab #20)

* 4 [doc] tomek

    Added several text files: AUTHORS (lists project authors and
    contributors), ChangeLog.md (contains all new user visible changes)
    and CONTRIBUTING.md (Contributor's guide, explains how to get your
    patches accepted in Stork project in a seamless and easy way.
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
* [func]  New feature.  In some cases this may be a backward incompatible
          change, which would require a bump of major version.
* [sec]   Security hole fix. This is no different than a general bug
          fix except that it will be handled as confidential and will cause
          security patch releases.
* [perf]  Performance related change.
* [ui]    User Interface change.

Header syntax:

[Leading asterisk] [Entry number] [Category] [Incompatibility mark (optional)] [Author]

The header components are delimited by a single space.

The backward incompatible or operational change is indicating by the
exclamation mark (!).
