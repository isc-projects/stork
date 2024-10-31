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
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

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
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

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
~~~~~~~~~~~~~~~~~

The Stork server provides a way to browse the logs of the Kea DHCP and BIND 9
services.

The logs are fetched directly from the filesystem, so the logs can be read
even if the Kea or BIND 9 services are down.

The Stork server can read only the data logged into a file. It cannot read
the logs from the syslog or standard output. The Stork agent must have the
necessary permissions to access the log files.

Viewing the DHCP data
~~~~~~~~~~~~~~~~~~~~~

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
~~~~~~~~~~~~~~~~~~~~~~~

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
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

The server provides a way to analyze the Kea DHCP configuration and suggest
tweaks and improvements. This solution allows to detect potential issues,
performance bottlenecks, and fields for optimization. It proposes also the
hooks that can be loaded to enable more Stork features.

Introduced in Stork 0.22.0.

Searching for leases
~~~~~~~~~~~~~~~~~~~~

The Stork server provides a search engine to find the DHCP leases. The user
can search for the leases by the IP address, MAC address, hostname, DUID, or
client identifier. They can also search for all declined leases.

This feature requires the ``lease_cmds`` hook loaded in Kea.

Stork server also displays a list of the leases related to a particular host
reservation.

Introduced in Stork 0.16.0.

Monitoring the BIND 9 service
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

The Stork server has a limited capabilities to monitor the BIND 9 service.
It can display the status of the BIND 9 service, the version of the BIND 9
daemon, and the details of the configured control and statistics channels.
The UI displays also the RNDC keys if set and the basic statistics.

The BIND 9 instance must be configured with the control channel to enable the
monitoring. Additionally, the Stork agent must have the necessary permissions
to access the ``named`` daemon configuration and to execute the RNDC commands.

The statistics channel must be configured to enable the statistics export to Prometheus.

Introduced in Stork 0.3.0.
