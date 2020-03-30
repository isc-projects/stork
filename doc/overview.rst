.. _overview:

********
Overview
********

Goals
=====

The goals of the Stork project are as follows:

- provide monitoring and insight into `ISC Kea DHCP` and `ISC BIND 9`
  operations
- to provide alerting mechanisms that indicate failures, fault
  conditions and other unwanted events in `ISC Kea DHCP` and
  `ISC BIND 9` services
- to permit easier troubleshooting of these services


Architecture
============

Stork is comprised of two components: ``Stork Server`` and ``Stork Agent``.

``Stork Agent`` is installed along with `Kea DHCP` or `BIND 9` and is
able to interact directly with those services. There may be many
agents deployed in a network, one per each machine.

``Stork Server`` is installed on a stand-alone machine. It connects to
any indicated agents and indirectly (via those agents) interacts with
the `Kea DHCP` and `BIND 9` services. It provides an integrated,
centralized front end for interacting with these services. There is
only one ``Stork Server`` deployed in a network.

.. figure:: static/arch.png
   :align: center
