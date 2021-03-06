.. _overview:

********
Overview
********

Goals
=====

The goals of the Stork project are as follows:

- to provide monitoring and insight into `ISC Kea DHCP`
  operations, and to eventually add them for `ISC BIND 9`
- to provide alerting mechanisms that indicate failures, fault
  conditions, and other unwanted events in `ISC Kea DHCP`, and
  eventually `ISC BIND 9`, services
- to permit easier troubleshooting of these services

Architecture
============

Stork is comprised of two components: the ``Stork Server`` and the ``Stork Agent``.

The ``Stork Agent`` is installed along with `Kea DHCP` or `BIND 9` and
interacts directly with those services. There may be many
agents deployed in a network, one per machine.

The ``Stork Server`` is installed on a stand-alone machine. It connects to
any indicated agents and indirectly (via those agents) interacts with
the `Kea DHCP` and `BIND 9` services. It provides an integrated,
centralized front end for interacting with these services.
Only one ``Stork Server`` is deployed in a network.

.. figure:: static/arch.png
   :align: center
