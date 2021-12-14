.. _overview:

********
Overview
********

Goals
=====

The goals of the ISC Stork project are as follows:

- to provide monitoring and insight into Kea DHCP operations
- to provide alerting mechanisms that indicate failures, fault
  conditions, and other unwanted events in Kea DHCP services
- to permit easier troubleshooting of these services

Although Stork currently only offers monitoring, insight, and alerts for
Kea DHCP, we plan to add similar capabilities for BIND 9 in future
versions.

Architecture
============

Stork is comprised of two components: the ``Stork Server`` and the ``Stork Agent``.

The ``Stork Agent`` is installed along with Kea DHCP and
interacts directly with those services. There may be many
agents deployed in a network, one per machine.

The ``Stork Server`` is installed on a stand-alone machine. It connects to
any indicated agents and indirectly (via those agents) interacts with
the Kea DHCP services. It provides an integrated,
centralized front end for these services.
Only one ``Stork Server`` is deployed in a network.

.. figure:: static/arch.png
   :align: center
