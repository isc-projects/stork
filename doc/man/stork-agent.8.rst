..
   Copyright (C) 2019-2020 Internet Systems Consortium, Inc. ("ISC")

   This Source Code Form is subject to the terms of the Mozilla Public
   License, v. 2.0. If a copy of the MPL was not distributed with this
   file, You can obtain one at http://mozilla.org/MPL/2.0/.

   See the COPYRIGHT file distributed with this work for additional
   information regarding copyright ownership.


stork-agent - Stork agent that monitors BIND 9 and Kea services
---------------------------------------------------------------

Synopsis
~~~~~~~~

:program:`stork-agent` [**--host**] [**--port**]

Description
~~~~~~~~~~~

The ``stork-agent`` is a small tool that is being run on the systems
that are running BIND 9 and Kea services. Stork server connects to
the stork agent and uses it to monitor services remotely.

Arguments
~~~~~~~~~

The Stork Agent takes the following arguments:

``-h`` or ``--help``
   Displays list of available parameters.

``-v`` or ``--version``
   Returns stork-agent version.

``--host=hostname``
   Specifies the IP to listen on. Can be controlled with $STORK_AGENT_ADDRESS environment
   variable. The default value is ``::``.

``--port=1234``
   Specifies the TCP port to listen on for connections. The default is 8080. Can be controlled
   with $STORK_AGENT_PORT environment variable.

``--listen-stork-only``
   Instructs the agent to listen for commands from the Stork Server but not for Prometheus requests.
   Can also be set with the $STORK_AGENT_LISTEN_STORK_ONLY environment variable.

``--listen-prometheus-only``
   Instructs the agent to listen for Prometheus requests but not for commands from the Stork Server.
   Can also be set with the $STORK_AGENT_LISTEN_PROMETHEUS_ONLY environment variable.

``--prometheus-kea-exporter-host``
   Instructs the agent to open Prometheus Kea exporter socket on specified address.

``--prometheus-kea-exporter-port``
   Instructs the agent to open Prometheus Kea exporter socket on specified port.

``--prometheus-kea-exporter-interval``
   Instruct the agent to how frequently the statistics should be pulled from Kea.

``--prometheus-bind9-exporter-host``
   Instructs the agent to open Prometheus BIND 9 exporter socket on specified address.

``--prometheus-bind9-exporter-port``
   Instructs the agent to open Prometheus BIND 9 exporter socket on specified port.

``--prometheus-bind9-exporter-interval``
   Instruct the agent to how frequently the statistics should be pulled from BIND 9.

Configuration
~~~~~~~~~~~~~

Stork agent uses two environment variables to control its behavior:

- STORK_AGENT_ADDRESS - if defined, governs which IP address to listen on

- STORK_AGENT_PORT - if defined, it controls which port to listen on. The
  default is 8080.


Mailing List and Support
~~~~~~~~~~~~~~~~~~~~~~~~~

There is a public mailing list available for the Stork project. **stork-dev**
(stork-dev at lists.isc.org) is intended for BIND 9 and Kea developers,
prospective contributors, and other advanced users. The list is available at
https://lists.isc.org. The community provides best-effort support
on both of those lists.

Once stork will become more mature, ISC will be providing professional support
for Stork services.

History
~~~~~~~

The ``stork-agent`` was first coded in November 2019 by Michal Nowikowski.

See Also
~~~~~~~~

:manpage:`stork-server(8)`
