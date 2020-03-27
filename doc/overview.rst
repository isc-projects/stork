.. _overview:

********
Overview
********

Goals
=====

The goals of Stork project are as follows:

- make easier to observe what is happening to `ISC Kea` and `ISC BIND 9` services
- provide alerting mechanisms that quicky indicate failures in `ISC Kea` and `ISC BIND 9` services
- make easier troubleshooting of these services


Architecture
============

Stork comprises of two components: ``Stork Server`` and ``Stork Agent``.

``Stork Agent`` is installed along `Kea` or `BIND 9`. This way the agent is able
to interact directly with `Kea` and `BIND 9` services.

``Stork Server`` is installed on stand alone machine. It connects to indicated agents
and indirectly (via agents) interacts with `Kea` and `BIND 9` services. This way it provides
integrated, centralized place for interacting with these services.

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
   |    Kea DHCP      |         /                   \             |    BIND9 DNS     |
   |                  |        |                     \            |                  |
   +------------------+        /                      -\          +------------------+
                              |                         \
                              /                 +------------------+
               +------------------+             |                  |
               |                  |             |   Stork Agent    |
               |   Stork Agent    |             |                  |
               |                  |             |    BIND9 DNS     |
               |    Kea DHCP      |             |                  |
               |                  |             +------------------+
               +------------------+
