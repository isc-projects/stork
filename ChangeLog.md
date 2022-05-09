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
    of the statistics have been renamed, e.g. total-addreses
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
    is new or has been occuring for a longer period of time.
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

    Implementated Bind exporter and embedded it in Stork Agent.
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
* [func]  new feature.  In some cases this may be a backward incompatible
	      change, which would require a bump of major version.
* [sec]   Security hole fix. This is no different than a general bug
          fix except that it will be handled as confidential and will cause
 	      security patch releases.
* [perf]  Performance related change.
* [ui]    User Interface change.

*: Backward incompatible or operational change.
