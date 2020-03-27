.. _overview:

********
Overview
********

Goals
=====

The goals of the Stork project are as follows:

- to make it easier for administrators to observe the operation of `ISC Kea DHCP` and `ISC BIND 9` services
- to provide alerting mechanisms that quickly indicate failures in `ISC Kea DHCP` and `ISC BIND 9` services
- to permit easier troubleshooting of these services


Architecture
============

Stork is comprised of two components: ``Stork Server`` and ``Stork Agent``.

``Stork Agent`` is installed along with `Kea DHCP` or `BIND 9` and is able
to interact directly with those services.

``Stork Server`` is installed on a stand-alone machine. It connects to any indicated agents
and indirectly (via those agents) interacts with the `Kea DHCP` and `BIND 9` services. It provides
an integrated, centralized front end for interacting with these services.

Architecture diagram::

                                +----------------+
                                |                |
                                |  Stork Server  |
                                |                |
                                +----------------+
                               ---/ /    \     \
                           ---/    |      -\    --\
                      ----/        /        \      --\
                     /            |          \        ---\
   +------------------+           /           -\          --\     +------------------+
   |                  |          |              \            --\  |                  |
   |   Stork Agent    |          /               \              --|   Stork Agent    |
   |                  |         |                 -\              |                  |
   |    Kea DHCP      |         /                   \             |    BIND 9 DNS    |
   |                  |        |                     \            |                  |
   +------------------+        /                      -\          +------------------+
                              |                         \
                              /                 +------------------+
               +------------------+             |                  |
               |                  |             |   Stork Agent    |
               |   Stork Agent    |             |                  |
               |                  |             |    BIND 9 DNS    |
               |    Kea DHCP      |             |                  |
               |                  |             +------------------+
               +------------------+
