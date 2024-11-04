.. _overview:

********
Overview
********

Goals
=====

The goals of the ISC Stork project are:

- To provide monitoring and insight into Kea DHCP operations.
- To provide alerting mechanisms that indicate failures, fault
  conditions, and other unwanted events in Kea DHCP services.
- To permit easier troubleshooting of these services.
- To allow remote configuration of the Kea DHCP servers.

Although Stork currently only offers monitoring, insight, alerts
and configuration for Kea DHCP, we plan to add similar capabilities
for BIND 9 in future versions.

Please refer to the :ref:`glossary` for specific wording used further
in this documentation and in the Stork UI.

Architecture
============

Stork is composed of two components: the Stork server (``stork-server``)
and the Stork agent (``stork-agent``).

The Stork server is installed on a stand-alone machine. It connects to
any agents, typically running on other machines, and indirectly (via those agents)
interacts with the Kea DHCP and BIND 9 apps. It provides an integrated,
centralized front end for these apps. Only one Stork server is deployed
in a network.

The Stork agent is installed along with Kea DHCP and/or BIND 9 and
interacts directly with those apps. There may be many
agents deployed in a network, one per machine. The following figure shows
connections between Stork components, Kea and BIND 9. It also shows different
kind of databases in a typical deployment.

.. figure:: ./static/arch.png
   :align: center
   :alt: Connections between Stork components, Kea and BIND 9


The presented example includes 3 physical machines, each running a Stork agent
instance, and Kea and/or BIND 9 apps. The leftmost machine includes a Kea
server connected to a database. It is typically one of the database systems
natively supported by Kea (MySQL or PostgreSQL). Kea uses the database
to store three types of information:

- DHCP leases (this storage is often referred to as lease database or lease backend),
- DHCP host reservations (this storage is referred to as host database or host backend),
- Kea configuration information (configuration backend).

For more information regarding the supported database backends please consult
`Kea ARM <https://kea.readthedocs.io/en/latest/arm/admin.html#kea-database-administration>`_.

Note that Stork server does not directly communicate with the Kea databases.
The lease, host and configuration information is pulled from the Kea instances
using the Kea control channel. Kea may pull necessary information from its database
to form a response. Depending on the configuration, Kea may use all database backends
or only a subset of them. It may also lack the database completely. If it uses
the database backends, they may be combined in the same database instance
or they may be separate instances. The rightmost machine on the figure above
is an example of the Kea server running without a database. In this case it
stores allocated DHCP leases in a CSV file (often called Memfile backend).

Stork server is connected to its own PostgreSQL database. It has a different
schema than Kea database and stores the information required for the Stork
server operation. This database is typically installed on the same physical
machine as the Stork server but may also be remote.

.. note::

  Unlike Kea, Stork server has no concept of replaceable database backends.
  It is integrated only with PostgreSQL. In particular, using MySQL as a
  Stork server database is not supported.

Stork server pulls the configuration information from the respective
Kea servers when they are first connected to the Stork server via agents,
saves pulled information in its local database and exposes to
the end users via the REST API. It continues to pull Kea configurations
periodically and updates the local database when it finds any changes. It
also pulls the current configuration from the Kea servers before applying
any configuration updates, to minimize a risk of conflicts with any
updates applied directly to the Kea servers (outside of Stork).

.. note::

  The future goal is to make Kea servers fully configurable from Stork. It
  already supports configuring the most frequently changing parameters
  (e.g., host reservations, subnets, shared networks and selected global parameters).
  However, some configuration capabilities are still unavailable. It implies that the
  administrators may sometimes need to apply configuration updates directly to the
  Kea servers, and these servers are the source of the configuration truth to
  Stork which periodically pulls this information. Nevertheless, we highly recommend
  applying configuration updates via Stork interface, whenever possible. Stork
  provides locking mechanisms preventing multiple end users from concurrently
  modifying configuration of the same Kea server. Direct configuration updates
  bypass this mechanism resulting in a risk of configuration conflicts.


Stork uses ``config-set`` and ``config-write`` Kea commands to save changes related
to global parameters and options, subnets and shared networks. For this to work, Kea
needs to have write access to its configuration. This is a security decision made
by a Kea administrator. Some deployments might choose to restrict write access.
In such cases, Stork will not be able to push configuration changes to Kea.

The host reservations management mechanism does not modify configuration on
disk. It stores host reservations in the database instead. Therefore the note above
does not apply to hosts management.

Preprocessing the Kea and BIND 9 statistics for the Prometheus server
=====================================================================

The BIND 9 and Kea DHCP servers provide statistics in their own custom formats.
The Stork agent preprocesses these statistics and converts them into a format
understood by the Prometheus server. The agents acts as a Prometheus exporter
and waits for the Prometheus server to scrape the statistics.

To fetch the statistics, Kea DHCP daemon must be configured to load the
``stats_cmds`` hook. The hook is responsible for sharing the statistics through
the Kea REST API. Optionally, the ``subnets_cmds`` hook can be loaded to
provide additional labels for the metrics exported to Prometheus.

The BIND 9 daemon must have a properly configured statistics channel to enable
this feature.

The Stork agent exports only a subset of the available statistics. The user
can limit the exported statistics in the agent configuration file.

Introduced in Stork 0.5.0 (Kea) and Stork 0.6.0 (BIND 9).

Monitoring status of services
=============================

The Stork server monitors continuously the status of the Kea DHCP daemons,
Kea Control Agent, Kea DHCP-DDNS and BIND 9 services and provides a dashboard
to show the current state.

The status is monitored on two levels. The first level is the status of the
machine where Kea or BIND 9 is running. The user can see if the connection to
the agent is established, and additional information about the machine, such as
the operating system, CPU and memory usage.
The second level is the status of the Kea DHCP and BIND 9 daemons. The user can
inspect if the processes are running, and if they are not, the user can see the
reason for the failure.

The Stork server keeps the events log, which contains history of the status
changes of the Kea and BIND 9 services.

Browsing the logs
=================

The Stork server provides a way to browse the logs of the Kea DHCP and BIND 9
services.

The logs are fetched directly from the filesystem, so the logs can be read
even if the Kea or BIND 9 services are down.

The Stork server can read only the data logged into a file. It cannot read
the logs from the syslog or standard output. The Stork agent must have the
necessary permissions to access the log files.

Viewing the DHCP data
=====================

The Stork server has a extensive capabilities to display the DHCP state and configuration. It
aggregates the data from all connected Kea servers and presents it in a
comprehensive form. It allows the user to browse all details of all networks in
a single place even if they are spread across multiple Kea servers.

The Stork server has dedicated pages for viewing the following data:

- Viewing subnets

  The user can see all subnets defined in the Kea servers. The user can view
  the subnet details, such as the subnet ID, subnet prefix, related DHCP
  options, and subnet pools.

  The user can also see the statistics of the subnet usage. They are presented
  only if the ``stats_cmds`` hook is loaded in a particular Kea server.

  If the particular subnet is specified in multiple Kea servers, it is
  displayed only once, with a list of server names where it is defined.

  Introduced in Stork 0.4.0.

- Viewing shared networks

  The user can see all shared networks defined in the Kea servers. The user
  can view the shared network details, such as the shared network ID, and shared
  network name. The server displays the list of subnets belonging to the shared
  network. The user can see the overall utilization of the shared network and
  the utilization of the subnets belonging to the shared network.

  The utilization data and other statistics are presented only if the
  ``stats_cmds`` hook is loaded in a particular Kea server.

  Introduced in Stork 0.5.0.

- Viewing host reservations

  The user can see all host reservations defined in the Kea servers. The user
  can view the host reservation details, such as host identifiers, DHCP options,
  and reserved hostname and IP addresses.

  The server can fetch the host reservations from the host database if the
  ``host_cmds`` hook is loaded in Kea.

  Introduced in Stork 0.6.0.

- Viewing global parameters and DHCP options

  The user can see the global parameters and DHCP options defined in the Kea
  servers.

  Introduced in Stork 1.18.0.

- Viewing the High-Availability status

  The user can see the status of the High-Availability configured across the
  Kea servers. The UI presents the detailed information about each HA peer.
  In case of a failure, the user can observe the reason for the failure and
  how the non-failed server is handling the situation.

  The Stork server gracefully supports the hub-and-spoke Kea feature.

  Introduced in Stork 0.3.0.

- Viewing the DHCP daemon details

  The user can see the details of the Kea DHCP daemons. The UI displays the
  daemon version, the database backends, the loaded hooks, and the whole
  configuration in a JSON format.

  Introduced in Stork 0.3.0.

Managing the DHCP configuration
===============================

The Stork server is capable of modifying the Kea DHCP configuration. It is
altered through calling the Kea hooks or by editing the JSON configuration on
the Stork server side and sending it back to the Kea server.

The following operations are supported:

- Adding, editing, and deleting subnets

  The user can add, edit, and delete subnets in the Kea servers. The user can
  change the subnet details, such as the subnet prefix, related DHCP options,
  and subnet pools.

  The ``subnet_cmds`` hook must be loaded in Kea to support this feature.

  Introduced in Stork 1.13.0.

- Adding, editing, and deleting shared networks

  The user can add, edit, and delete shared networks in the Kea servers. The
  user can change the shared network details, such as the shared network name, 
  the list of subnets belonging to the shared network and the DHCP options.

  The ``subnets_cmds`` hook must be loaded in Kea to support this feature.

  Introduced in Stork 1.18.0.

- Adding, editing, and deleting host reservations

  The user can add, edit, and delete host reservations in the Kea servers. The
  user can change the host reservation details, such as host identifiers, DHCP
  options, and reserved hostname and IP addresses.

  The ``host_cmds`` hook must be loaded in Kea to support this feature.

  Introduced in Stork 1.3.0.

- Editing global parameters and DHCP options

  The user can edit the global parameters and DHCP options in the Kea servers.

  Introduced in Stork 1.19.0.

Reviewing the Kea configuration
===============================

The server provides a way to analyze the Kea DHCP configuration and suggest
tweaks and improvements. This solution allows to detect potential issues,
performance bottlenecks, and fields for optimization. It proposes also the
hooks that can be loaded to enable more Stork features.

Introduced in Stork 0.22.0.

Searching for leases
====================

The Stork server provides a search engine to find the DHCP leases. The user
can search for the leases by the IP address, MAC address, hostname, DUID, or
client identifier. They can also search for all declined leases.

This feature requires the ``lease_cmds`` hook loaded in Kea.

Stork server also displays a list of the leases related to a particular host
reservation.

Introduced in Stork 0.16.0.

Monitoring the BIND 9 service
=============================

The Stork server has a limited capabilities to monitor the BIND 9 service.
It can display the status of the BIND 9 service, the version of the BIND 9
daemon, and the details of the configured control and statistics channels.
The UI displays also the RNDC keys if set and the basic statistics.

The BIND 9 instance must be configured with the control channel to enable the
monitoring. Additionally, the Stork agent must have the necessary permissions
to access the ``named`` daemon configuration and to execute the RNDC commands.

The statistics channel must be configured to enable the statistics export to Prometheus.

Introduced in Stork 0.3.0.

Security design
===============

Stork has been designed with security in mind. The following section describes
the security design and the security features implemented in Stork.

The Stork environment is composed from several services, i.e., Stork server, Stork agent(s), Kea Control Agent, Kea
DHCP daemons, Kea D2 daemon, BIND 9 daemon, PostgreSQL database, Prometheus. Each service has its own security
considerations.

There is a diagram of all Stork components and services that it interacts with:

.. figure:: ./static/ecosystem-protocols.drawio.png
   :align: center
   :alt: Stork security diagram

   Connections and protocols between Stork components and services

The Stork server is the central component of the Stork environment. It serves the Web UI and REST API over the HTTP
protocol (connections no. 1, 4, and 8 on the diagram). The administrator may secure it by providing a trusted
SSL/TLS certificate. It is recommended especially when the Stork server is exposed to the public network.
The Stork server may share some statistics with the Prometheus monitoring system. It is strongly recommended to limit
access to the metrics endpoint to the Prometheus server only. Stork server has no a built-in mechanism to do it but it
may be achieved by using a reverse proxy like Nginx or Apache. See the :ref:`server-setup` section for more details.

The Stork server requires a PostgreSQL database to store its data. The connection to the database is may be established
over the local socket or over the HTTP protocol (connection no. 10 on the diagram). The first option is more secure,
as it does not expose the database traffic to the network but it requires the database to be installed on the same
machine as the Stork server. The second option allows the database to be installed on a different machine, but it is
recommended to secure the connection with SSL/TLS. The Stork server supports a mutual TLS authentication with the
database that should ensure the highest level of security. In any case, Stork server should use a dedicated database
user with the minimum required permissions and no one else should have access to the database. The database should be
regularly backed up. See the :ref:`securing-the-database-connection` for more details.

The Stork agent resides on the same machine as the Kea and BIND 9 daemons and it is permitted to access their
configuration files, logs, and use their APIs. Additionally, it can list the processes running on the machine and read
their details. Therefore, it is recommended to run the Stork agent as a dedicated user with the minimum required
permissions.
The Stork server communicates with the Stork agents over the GRPC protocol (connection no. 5 on the diagram). The Stork
has a built-in solution for securing the communication on this channel using the Transport Layer Security (TLS)
protocol. It is a mutual TLS authentication that ensures that the server and the agent are who they claim to be.
It is self-managed and does not require any additional configuration. The server acts as a Certificate Authority (CA)
and generates the root certificate and the private key. They are stored in the server's database. The server generates
a certificate and a private key for each agent during the agent registration process. The agent uses the certificate and
the private key to authenticate itself to the server. The server doesn't trust the agent's certificate by default. The
server administrator must approve the agent registration request in the Stork web UI. The server administrator must
compare the token displayed in the UI with the token displayed in the agent's logs. If the tokens match, the
administrator can approve the registration request. It is a one-time operation that protect against the
man-in-the-middle attacks.
This mechanism can be by-passed by using an additional server token for the agent registration. The server token is a
secret available only to the administrator on the server UI. It may be provide to the agent during the agent registration
process. The agents registered with this token are automatically approved by the server.
The server token is a secret and must be protected. It is recommended to use it only in the secure environments. If it
is compromised, the administrator can revoke it in the server UI. See the :ref:`secure-server-agent` for more details.

Stork agent is responsible for exchange the data between the Stork server and the Kea (connection no. 11. on the
diagram) and BIND 9 (connections no. 7 and 9 on the diagram) daemons. The agent and the daemons are running on the same
machine, so the communication is local. However, it still can be secured.

Kea Control Agent supports Basic Auth to authenticate the clients of its REST API - the control channel used by the
Stork agent. This solution may be enabled to protect the Kea CA from unauthorized access. If it is enabled, the Stork
agent must be configured with the username and password to authenticate itself to the Kea CA. It is recommended to limit
the access to this file only to the Stork agent user. Kea Control Agent may be configured to serve the REST API over the
HTTPS protocol. Is is strongly recommended to enable it if the Basic Auth is configured or if the Kea CA listens on the
non-localhost interfaces. Additionally, the Kea CA may be configured to require the client certificate to authenticate
the clients. The Stork agent supports the mutual TLS authentication partially. If it recognizes the Kea CA requires the
client certificate, it attaches its GRPC client certificate (the certificate that was obtained during the agent
registration) to the request. This certificate doesn't pass the client certificate verification by the Kea CA. It means
that the Kea CA must be configured to not verify the client certificate.

Connection to BIND 9 utilizes two protocols: RNDC (control channel, connection no. 9 on the diagram) and HTTP (
statistics channel, connection no. 7 on the diagram). The RNDC protocol may be secured by using the RNDC keys. It is
especially recommended if the BIND 9 daemon listens on the non-localhost interfaces. The Stork agent retries the RNDC
key from the BIND 9 configuration file. The agent must have the necessary permissions to read this file and use the
``rndc`` and ``named-checkconf`` commands.
The statistics channel is served over the HTTP protocol and may be secured by the SSL/TLS certificate.

The Stork agent may acts as a Prometheus exporter for the Kea and BIND 9 statistics. The Prometheus server scrapes the
metrics from the agent over the HTTP protocol (connection no. 6 on the diagram). This connection is unsecure and doesn't
support TLS. The metrics channel is expected to not be exposed to the public network. It is recommended to configure the
firewall to limit the access to the metrics endpoint to the Prometheus server only.

The Stork server supports hooks that may be loaded to provide new authentication methods. If the authentication methods
uses a dedicated authentication service, it is recommended to secure the connection to this service with the SSL/TLS
certificate if the service and hook supports it. Especially, the LDAP hook may be configured to use the SSL/TLS (LDAPS)
protocol.
