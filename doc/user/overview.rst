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

Although Stork currently only offers monitoring, insight, and alerts for
Kea DHCP, we plan to add similar capabilities for BIND 9 in future
versions.

Architecture
============

Stork is comprised of two components: the Stork server (``stork-server``) and the Stork agent (``stork-agent``).

The Stork server is installed on a stand-alone machine. It connects to
any indicated agents and indirectly (via those agents) interacts with
the Kea DHCP services. It provides an integrated,
centralized front end for these services.
Only one Stork server is deployed in a network.

The Stork agent is installed along with Kea DHCP and
interacts directly with those services. There may be many
agents deployed in a network, one per machine.

.. figure:: static/arch.png
   :align: center
