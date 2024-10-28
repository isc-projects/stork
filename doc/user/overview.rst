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

Features
========

Stork agent provides the following features:

- Forwarding commands from the Stork server to the Kea DHCP and BIND 9 applications
- Passing the Kea statistics to the Prometheus server

  Requires the ``stat_cmds`` hook loaded in Kea. Optionally, the ``subnets_cmds`` hook
  can be loaded to provide additional labels.

- Passing the BIND 9 statistics to the Prometheus server

Stork server provides the following features:

- Monitoring a status of Kea DHCP, Kea Control Agent, and Kea DHCP-DDNS services
- Monitoring a status of BIND 9 services.
- Monitoring a status of the machine where Kea or BIND 9 is running
- Viewing an overview of the machine where Kea or BIND 9 is running (resource usage, operating system details)
- Browsing Kea logs
- Adding, editing, and deleting subnets

  Requires the ``subnet_cmds`` hook loaded in Kea.

- Viewing shared networks
- Adding, editing, and deleting shared networks

  Requires the ``subnets_cmds`` hook loaded in Kea.

- Viewing host reservations

  Browsing host reservations from the host database requires the ``host_cmds`` hook loaded in Kea.

- Viewing global parameters
- Viewing global DHCP options
- Viewing the High-Availability status

  Requires the HA hook loaded and configured in Kea.

- Viewing database backends, loaded hooks and other Kea configuration parameters
- Viewing a whole Kea configuration in a JSON format
- Reviewing the Kea configuration and suggesting tweaks and improvements

- Searching for leases

  Requires the ``lease_cmds`` hook loaded in Kea.

- Viewing leases related to a particular host reservation

  Requires the ``lease_cmds`` and ``host_cmds`` hooks loaded in Kea.

- Viewing statistics of the subnet and shared network usage

  Requires the ``stat_cmds`` hook loaded in Kea.

- Viewing a basic information about the BIND 9 instances
- Viewing the RNDC keys
- Viewing statistics of the BIND 9 utilization
- Viewing the maintenance events 
