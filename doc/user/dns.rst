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
and BIND 9 will not appear on the list of detected apps.

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
and PowerDNS will not be shown on the list of detected apps.

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

Any errors occuring during the zone fetch can be inspected by clicking the
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

The remaining filtering boxes allow for filtering the zones by ``App ID``,
``Serial``, ``Class``, and ``App Type``.


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

