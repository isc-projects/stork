.. _dns:

***
DNS
***

DNS Servers Integration with Stork
==================================

Stork can monitor the following DNS servers:

- `BIND 9 <https://www.isc.org/bind/>`_
- `PowerDNS <https://www.powerdns.com/>`_

Stork agent interacts with these servers using certain APIs. To use these APIs, the
agent must parse DNS servers' configuration files to retrieve the configurations of
these APIs, credentials, etc. It implies certain requirements on the DNS servers'
configurations. These requirements are described in the respective sections below.

BIND 9
~~~~~~

Detection
---------

Stork agent begins detecting the BIND 9 server by parsing the process command line.
If the ``named`` process is started with the ``-c`` parameter, the agent uses the
path specified in the parameter as the configuration file location. If ``named`` was
started without this parameter, the agent will use the config file location specified
in the ``STORK_AGENT_BIND9_CONFIG`` environment variable, if set.

If the config file is not found using the methods described above, the agent will try
to determine its location by executing and parsing the output of the ``named -V`` command,
which contains the information about the BIND 9 build. Finally, it will fallback to
the typical config file locations in the following order:

- ``/etc/bind/``
- ``/etc/opt/isc/isc-bind/``
- ``/etc/opt/isc/scls/isc-bind/``
- ``/usr/local/etc/namedb/``

If the config file is not found using the methods described above, the agent will report an error,
and BIND 9 will not appear on the list of detected daemons.

.. note::
    The ``STORK_AGENT_BIND9_CONFIG`` environment variable setting has no effect if
    the ``named`` process was started with the ``-c`` parameter. The explicit
    command line parameter takes precedence over the user setting because the parameter
    indicates the config file location that the ``named`` process is actually using.

Access Point Settings
---------------------

Stork agent requires access to the BIND 9 configuration file to retrieve the
information about the control API and statistics endpoints, as well as the
security keys accepted by these APIs. It also retrieves the server's IP address
and security credentials to perform AXFR zone transfers. The agent uses
AXFR to get the zone contents (RRs).

The following is an example setting of ``controls`` block that the agent will
try to determine the ``rndc`` endpoint and credentials to use:

.. code-block:: text

    controls {
        inet * port 9053 allow { localhost; } keys { "rndc-key"; };
    };


where ``rndc-key`` is the name of the key allowed to authenticate the ``rndc``
command. The ``keys`` clause is optional. If used, the relevant key must
be specified in the config file. For example:


.. code-block:: text

    key "rndc-key" {
        algorithm hmac-sha256;
        secret "iCQvHPqq43AvFK/xRHaKrUiq4GPaFyBpvt/GwKSvKwM=";
    };

If no ``port`` is specified in the ``controls`` block the agent will assume the default
port 953. If no ``controls`` block is found the agent will assume the ``rndc`` endpoint is
``127.0.0.1:953``.

The statistics channel is used by the Stork agent for two purposes. First,
for fetching and exporting DNS server statistics from BIND 9 to
`Prometheus <https://prometheus.io>`_. Second, it is used for fetching
a list of configured views and zones. The following is the example statistics
channel setting expected by the agent:

.. code-block:: text

    statistics-channels {
        inet 127.0.0.1 port 8053 allow { 127.0.0.1; };
    };

According to these settings, the agent will try to get the statistics from the
`http://127.0.0.1:8053/json/v1` endpoints including:

- `http://127.0.0.1:8053/json/v1/server`
- `http://127.0.0.1:8053/json/v1/traffic`
- `http://127.0.0.1:8053/json/v1/zones`

The agent will assume the default port 80 if no port is specified. The ``statistics-channels``
block is mandatory to enable exporting statistics to `Prometheus <https://prometheus.io>`_
and for the :ref:`zone_viewer`.

Zone Transfer Settings
----------------------

Stork agent uses zone transfer (AXFR) to get the zone contents (RRs) when a
user clicks ``Show Zone`` button in the zone viewer. The agent extracts the
following configuration information from the BIND 9 configuration file to
perform the zone transfer for a selected zone:

- DNS server address and port using the ``listen-on`` or ``listen-on-v6`` options.
- TSIG key name, algorithm and secret using the ``allow-transfer`` and ``match-clients`` options.

Using the TSIG key is optional when the desired zone is defined in the default
view (i.e., when the zone is specified globally, rather than in a custom view).
It is mandatory when the desired zone is in a non-default view because the DNS server
determines the view where the zone belongs based on the TSIG key. Without the TSIG
key the request is ambiguous because the zone with the given name may belong to
multiple views.

.. note::

    Please refer to the `Understanding views in BIND 9, with examples <https://kb.isc.org/docs/aa-00851>`_
    article for more details about views in BIND 9.


The algorithm by which the Stork agent determines appropriate TSIG key is complex and
requires some explanation.

The ``allow-transfer`` statement in BIND 9 configuration controls who can perform
zone transfer for a given zone or view. This setting can be specified in the zone scope,
view scope or as a global option. The match list can contain IP addresses, keys, ACLs
or keywords (e.g., ``any``, ``none``). The zone-level setting overrides the view-level
setting, which overrides the global setting. From the Stork agent's perspective the most
important information extracted from the ``allow-transfer`` statements is whether the
zone transfer is allowed (is not ``none``), and if they contain any references to the
TSIG keys to be used in the zone transfer.

The ``match-clients`` statement can be defined in a view scope. The server uses this
statement to match the DNS clients with a given view. It can contain IP addresses,
keys or ACLs. This statement is another source of information for the Stork agent about
the TSIG keys to be used for the zone transfer. Any keys specified in this statement
will take precedence over the keys specified in the ``allow-transfer`` statement.

It is important to note that since the Stork agent runs on the same machine as the DNS
server, the source IP addresses used by the agent cannot be used by the DNS server
for matching the AXFR requests with the views. It imposes a requirement on the BIND 9
configuration to rather use keys as view discriminators in the ``match-clients``
and/or ``allow-transfer`` statements. For example:

.. code-block:: text

    key "trusted-key" {
        algorithm hmac-sha256;
        secret "VO6xA4Tc1PWYaqMuPaf6wfkITb+c9/mkzlEaWJavejU=";
    };

    key "guest-key" {
        algorithm hmac-sha256;
        secret "6L8DwXFboA7FDQJQP051hjFV/n9B3IR/SwDLX7y5czE=";
    };

    acl trusted { !key guest-key; key trusted-key; localhost; };
    acl guest   { !key trusted-key; key guest-key; localhost; };

    view "trusted" {
        match-clients { trusted; };
        zone "bind9.example.com" {
            type master;
            file "/etc/bind/db.bind9.example.com.trusted";
        };
    };

    view "guest" {
        zone "bind9.example.com" {
            type master;
            file "/etc/bind/db.bind9.example.com.guest";
            allow-transfer { guest; };
        };
    };

This configuration snippet defines two views: ``trusted`` and ``guest``. Both
views contain a zone name. The ``trusted`` view is associated with the ``trusted-key``
key via ACL ``trusted``. The ``guest`` view is associated with the ``guest-key``
via the ACL ``guest``, and the ``allow-transfer`` statement, instead of ``match-clients``.
This configuration carries enough information for the Stork agent to perform
successful zone transfer for the ``bind9.example.com`` zone in any of the views.
The agent will pick the correct TSIG key to let the DNS server determine the desired view.

When the DNS server is not configured to use custom views, the configuration can
be much simpler:

.. code-block:: text

    zone "bind9.example.com" {
        type master;
        allow-transfer { any; };
        file "/etc/bind/db.bind9.example.com";
    };

This zone is defined globally and uses ``allow-transfer`` statement to allow anybody
to perform zone transfer. It requires no TSIG keys. If the reference to a TSIG key
is attached to the zone via ``allow-transfer`` statement, the agent will use this
key to perform the zone transfer.

See `match-clients <https://bind9.readthedocs.io/en/stable/reference.html#namedconf-statement-match-clients>`_
and `allow-transfer <https://bind9.readthedocs.io/en/stable/reference.html#namedconf-statement-allow-transfer>`_
sections of the BIND 9 reference manual for more details.

Zone Transfer Monitoring Settings
---------------------------------

The :ref:`zone-transfers-monitoring` section documents the usage of the zone transfer monitoring
dashboard to track the transfers between all BIND servers in the network administrated with Stork.
In this section, it is described how BIND servers should be configured to make use of this feature.

Stork agent takes advantage of BIND 9 logging to detect zone transfers on that BIND instance.
It can parse and follow both the logs emitted directly to a file or to the ``systemd`` journal.

Stork can infer the log files to track from the BIND configuration when BIND is detected. Such
a configuration can look similar to the following:

.. code-block:: text

    logging {
        channel default_log {
            file "/var/log/bind/default.log" versions 3 size 20m;
            print-time yes;
            print-category yes;
            print-severity yes;
            severity info;
        };
        channel xfer-in {
            file "/var/log/bind/xfer-in" versions 3 size 20m;
            print-time yes;
            print-severity yes;
            severity info;
        };
        channel xfer-out {
            file "/var/log/bind/xfer-out" versions 3 size 20m;
            print-time yes;
            print-severity yes;
            severity info;
        };
        category xfer-out { xfer-out; };
        category xfer-in { xfer-in; };
    };


It creates dedicated channels for the zone transfer logs. Stork agent will track the
logs from these channels but not the default channel. Even though, separation of the logs
is not strictly required, it is recommended in the installations with large logs volumes
to avoid the performance degradation. If zone transfer monitoring logs are mixed with the
other logs, Stork agent will have to work harder to filter those that are relevant.

The ``severity`` of the log messages in the zone transfer channels should be set to ``info``
at least. It can be set to ``debug``. Logging at higher levels (e.g., ``warning`` or ``notice``)
will effectively silence any logs useful for zone transfers detection. As a result, the zone
transfers from the particular server will not appear in the dashboard.

Stork agent should be able to extract the relevant logging settings in most cases. Still, the
administrators can use two Stork agent's command line switches
(i.e., ``--xfr-in-tracking-path`` and ``--xfr-out-tracking-path``) and the corresponding
environment variables (i.e., ``STORK_AGENT_XFR_IN_TRACKING_PATH`` and
``STORK_AGENT_XFR_OUT_TRACKING_PATH``) to specify the exact locations of the log files to track.
Both can point to the same file if both incoming and outgoing zone transfers are logged
to it.

Stork agent must be run with the ``--enable-xfr-tracking`` switch or the ``STORK_AGENT_ENABLE_XFR_TRACKING``
environment variable set to ``true`` to enable zone transfer monitoring.

For example:

.. code-block:: text

    $ stork-agent --enable-xfr-tracking --xfr-in-tracking-path /var/log/named/xfer-in --xfr-out-tracking-path /var/log/named/xfer-out


If BIND server is running as a ``systemd`` service, the agent must be configured to track the
``systemd`` logs instead. In this case, the agent must be started with the ``--xfr-tracking-systemd-unit``
switch or the ``STORK_AGENT_XFR_TRACKING_SYSTEMD_UNIT`` environment variable set to the BIND service name
(typically ``named.service`` or simply ``named``).

For example:

.. code-block:: text

    $ stork-agent --enable-xfr-tracking --xfr-tracking-systemd-unit named.service

In that case, the Stork agent runs the ``journalctl`` command to track the logs from ``systemd`` service.

.. note::
    When Stork agent is parsing log files to track zone transfers, the administrator must ensure that the
    Stork agent's user has read permissions to the BIND log directory and the log files. This is typically
    achieved by assigning the ``stork-agent`` user to a ``named`` or ``bind`` group. When Stork agent is parsing
    ``systemd`` logs to track zone transfers, the ``stork-agent`` user must be added to the administrator
    group (e.g., ``wheel``, ``adm``) or ``systemd-journal`` group.



PowerDNS
~~~~~~~~

Detection
---------

Stork agent begins detecting the PowerDNS server by parsing the process command line.
If the ``pdns_server`` process is started with the ``--config-dir`` parameter, the agent
uses the path specified in the parameter as the configuration file location. The default
configuration file name is ``pdns.conf``, but the server can be started with the
``--config-name`` parameter described in the `PowerDNS documentation <https://doc.powerdns.com/authoritative/guides/virtual-instances.html#running-virtual-instances>`_.
Stork agent uses the custom file name resulting from using this parameter, if it
is found in the server's command line.

If ``pdns_server`` was started without the ``--config-dir`` parameter, the agent will use
the config file location specified in the ``STORK_AGENT_POWEDNS_CONFIG`` environment variable,
if set.

If the config file is not found using the methods described above, the agent will try
to find the config file in the typical locations in the following order:

- ``/etc/powerdns/``
- ``/etc/pdns/``
- ``/usr/local/etc/``
- ``/opt/homebrew/etc/powerdns/``

If the config file is not found using the methods described above, the agent will report an error,
and PowerDNS will not be shown on the list of detected daemons.

.. note::
    The ``STORK_AGENT_POWERDNS_CONFIG`` environment variable setting has no effect if
    the ``pdns_server`` process was started with the ``--config-dir`` parameter. The explicit
    command line parameter takes precedence over the user setting because the parameter
    indicates the config file location that the ``pdns_server`` process is actually using.


Access Point Settings
---------------------

Stork agent requires access to the PowerDNS configuration file to retrieve the
information about the control API (webserver) endpoint, as well as the
security key accepted by this API. The agent uses the control API to get the
general server information, a list of zones, and zone contents (RRs).

The webserver must be enabled for the Stork agent to detect monitor the
PowerDNS server. The following is a simple configuration snippet containing
the settings expected by the agent:

.. code-block:: text

    # The API must be explicitly enabled.
    api=yes
    api-key=changeme
    webserver=yes

    # The webserver-address and webserver-port settings are optional.
    # If not specified, the agent will use the default values of 127.0.0.1:8081.
    webserver-address=0.0.0.0
    webserver-port=8085

.. note::

    Please specify a random, strong API key in the ``api-key`` setting. Do not use
    ``changeme`` nor other easy to guess value in production.

Zone Transfer Settings
----------------------

Stork agent uses zone transfer (AXFR) to get the zone contents (RRs) when a
user clicks ``Show Zone`` button in the zone viewer.

.. note::

    DNS views introduced in the PowerDNS 5.0.0 version are not supported by Stork yet.
    For that reason, the agent is not using TSIG keys for the zone transfer.

The agent merely checks if the ``allow-axfr-ips`` setting allows for the zone
transfer from the local host, and if the ``disable-axfr`` is not set to
``true``. It also extracts the DNS server port from the ``local-port`` setting,
if specified. The following is a simple configuration snippet that explicitly
enables zone transfer by the agent:

.. code-block:: text

    allow-axfr-ips=127.0.0.1,::1
    disable-axfr=no
    local-port=53

In fact, all of these settings are optional because they are set to their
default values above.

.. _zone_viewer:

Zone Viewer
===========

Listing Zones
~~~~~~~~~~~~~

Zone viewer lists the zones gathered from all monitored DNS servers and allows
for filtering them and browsing their contents (RRs). The Stork agents local to
the monitored DNS servers are responsible for gathering the list of zones using
APIs provided by these servers, and getting the zone contents using zone transfer.
While getting the list of zones occur automatically once the agent starts, getting
the RRs is not immediate, and is only initiated by the Stork server when the user
clicks the ``Show Zone`` button in the zone viewer.

In order to list the zones gathered by the agent, navigate to the ``DNS --> Zones``.
The list of zones is initially empty. Stork server does not gather the zones
automatically for performance reasons. To see the zones on the list, click the
``Fetch Zones`` button. The server will contact all connected Stork agents
running on the same machines as the DNS servers to fetch the zones that the
agents had gathered. This operation may take significant amount of time (sometimes
minutes) depending on the number of zones.

The zones are cached in the Stork server database, so browsing the list of fetched
zones is fast. The zones are not refreshed automatically. To see the updated list of
zones, click the ``Fetch Zones`` button again.

Any errors occurring during the zone fetch can be inspected by clicking the
``Fetch Status`` button. The status view also includes the following information:

- **Zone Configs Count**: the number of different zone configurations in the server (if the same zone name appears in multiple views, it is counted multiple times).
- **Distinct Zones**: the number of different zones in the server (if the same zone name appears in multiple views, it is counted only once).
- **Builtin Zones**: the number of distinct builtin zones in the server. Builtin zones are special zones automatically generated by BIND 9.

The number of builtin zones for each BIND 9 server is around hundred. It is often
convenient to filter out the builtin zones from the list to only browse
those that are configured by the user. Click the ``Toggle builtin zones`` to
exclude or include the builtin zones on the list.

The listed zone types can be selected using the ``Zone Type`` dropdown.
A ``master`` zone type is an alias for the ``primary`` zone type, and a
``slave`` zone type is an alias for the ``secondary`` zone type.
``master`` and ``slave`` types are not listed in the dropdown.
Selecting ``primary`` or ``secondary`` will include ``master`` and ``slave``
zones besides ``primary`` or ``secondary`` accordingly.

``RPZ`` is a special type of zone (response policy zone) which configures
the DNS server to apply a set of rules to the DNS queries. The ``RPZ``
filtering box provides three options:

- ``include``: include RPZ along with other zones,
- ``exclude``: exclude RPZ from the list, and only show non-RPZ zones,
- ``only``: return only RPZ.

The remaining filtering boxes allow for filtering the zones by ``DaemonID``,
``Serial``, ``Class``, and ``Daemon Name``.


Viewing Zone Contents
~~~~~~~~~~~~~~~~~~~~~

The details of the selected zone are shown in a tab when the zone name is
clicked on the list. If the selected zone's name is found on multiple DNS
servers/views, the ``DNS Views Associated with the Zone`` has multiple
rows, each row displaying the details for the given DNS server/view, and
the ``Show Zone`` button.

The ``Show Zone`` button is only enabled if the zone type is ``primary`` or
``secondary`` because zone transfer is only supported for these zone types.

When the ``Show Zone`` button is clicked, the server contacts appropriate
Stork agent to attempt the zone transfer unless the zone contents have
been already transferred and are cached in the Stork server database.
Caching reduces the burden on the DNS servers and respective agents to
run zone transfer for each ``Show Zone`` button click. However, it implies
that the zone contents may get outdated. To enforce the zone transfer, and
get the latest snapshot of the zone contents, click the ``Refresh from DNS``
button. Check ``Cached from DNS server on`` timestamp to see the age of the
presented zone contents.

.. _zone-transfers-monitoring:

Zone Transfers Monitoring
=========================

In the networks with interconnected primary and secondary DNS servers, it is possible
to monitor the zone transfers between them. Zone transfers are used by the DNS servers
to synchronize zone contents. A zone configured or updated on a primary DNS server is
propagated to the secondary DNS servers connected to the primary. The secondary DNS servers
can have other secondary DNS servers connected to them. Moreover, a secondary DNS server
for a particular zone can be a primary DNS server for another zone. With this level of
granularity, the administrators can sometimes create very complex hierarchical topologies
of the DNS services which increases the risk of the failures, and difficulties in troubleshooting.

To help with monitoring and troubleshooting, Stork provides the zone transfer monitoring
dashboard.


Zone Transfers Monitoring Dashboard
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

Navigate to the ``DNS --> Zone Transfers`` to open the dashboard.

The dashboard displays zone transfers sorted from the one that started most recently
(i.e., by the ``Started At`` timestamp). This is the timestamp read from the DNS
server logs marking the beginning of the zone transfer. In other words, this is the
exact moment when the zone transfer was initiated by the DNS server.

When expanding a zone transfer row, there are also other timestamps displayed in the
``Transfer Details`` section. The ``Created At`` timestamp marks the moment when the
zone transfer was first detected and recorded by Stork. The ``Completed At`` timestamp
indicates when the zone transfer was completed (either successfully or unsuccessfully).

Zone transfers can have one of the following statuses:

- ``started`` - the zone transfer was initiated and it is still in progress
- ``completed`` - the zone transfer was completed successfully
- ``message`` - the zone transfer neither completed successfully nor failed, but a log message pertaining to this zone transfer was logged after it was started
- ``failed`` - the zone transfer failed as indicated by the zone transfer status log message
- ``up-to-date`` - the zone transfer was initiated but the zone was already up to date, as indicated by the received SOA record, so the transfer is discontinued

Stork marks the zone transfer as ``failed`` when it comes across BIND log message similar to this:

.. code-block:: text

    23-Feb-2026 10:41:27.147 0x7ffffb63b000: transfer of 'bind9.example.org/IN' from 172.24.0.53#53: Transfer status: connection refused


It clearly indicates the transfer failure. However, it is possible that BIND logs some more subtle messages indirectly indicating
the transfer failure. In that case, the zone transfer status may be set to ``message``, and the administrator should inspect
the ``Log Message`` field to evaluate the transfer result. Analyzing BIND logs directly may also be helpful.

The ``Primary`` and the ``Secondary`` columns hold the IP addresses or names of the machines hosting the DNS servers participating
in the transfers. The ``Primary`` is the DNS server performing an outgoing transfer, and the ``Secondary`` is the DNS server
performing an incoming transfer. These columns may contain either an IP address (without the link) or a link to the Stork agent
monitoring the given DNS server. It is an IP address when the DNS server participating the transfer is outside of the network
monitored by Stork or the DNS server is within the network but is not monitored by Stork.

For example, if the local DNS server mirrors the root zone:

.. code-block:: text

    zone "." {
        type mirror;
        allow-transfer { any; };
        primaries { 192.5.5.241; };
    };


the ``Primary`` column will contain the IP address ``192.5.5.241`` (without the link), and the ``Secondary`` column will contain the link
to the Stork agent where the local DNS server transferring from the F-root server is running.

The ``Statistics`` field contains the statistics for the zone transfer reported by BIND in the logs.


Local Zone Transfers
~~~~~~~~~~~~~~~~~~~~

Stork agent uses `AXFR (authoritative transfer) <https://www.rfc-editor.org/info/rfc5936>`_
to fetch zone contents from a local DNS server. If a user clicks the ``Show Zone`` button and the
zone contents are not yet cached in the Stork server database, the zone transfer is initiated
between the agent and the local DNS server. These transfers are captured by the zone transfer
monitoring but they are not shown in the dashboard by default. They are less interesting from
the administrator's perspective and presenting them could clutter the dashboard.

To see the local zone transfers in the dashboard, check the ``Include local transfers``
checkbox above the transfers list. They are marked with the ``local`` tag in the
``Primary`` and the ``Secondary`` columns, so it is easy to distinguish them from
the transfers between the DNS servers.


Removing Old Zone Transfers
~~~~~~~~~~~~~~~~~~~~~~~~~~~

To prevent continued growth of the database size, Stork periodically removes the old
zone transfers. By default, zone transfers started more than 24 hours ago are removed.
This setting can be controlled on the Settings page in the ``Zone Transfers`` section.
Set the new value for the ``Maximum age of the zone transfers to keep`` in seconds.
Note that this value must be greater or equal 60 seconds. In order to disable the removal
of old zone transfers, uncheck the ``Enable zone transfer pruning`` checkbox.
