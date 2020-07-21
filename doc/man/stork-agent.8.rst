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

Stork does not use explicit configuration file. Instead, its behavior can be controlled with
command line switches and/or variables. The Stork Agent takes the following command line switches.
Equivalent environment variables are listed in square brackets, where applicable.

``--listen-stork-only``
   listen for commands from the Stork Server only, but not for Prometheus requests.
   [$STORK_AGENT_LISTEN_STORK_ONLY]

``--listen-prometheus-only``
   listen for Prometheus requests only, but not for commands from the Stork Server.
   [$STORK_AGENT_LISTEN_PROMETHEUS_ONLY]

``-v`` or ``--version``
   show software version.

Stork Server flags:

``--host=``
   Specifies the IP or hostname to listen on. [$STORK_AGENT_ADDRESS]

``--port=``
   Specifies the TCP port to listen on for connections. (default: 8080) [$STORK_AGENT_PORT]

Prometheus Kea Exporter flags:

``--prometheus-kea-exporter-host=``
   the IP or hostname to listen on for incoming Prometheus connection (default: 0.0.0.0)
   [$STORK_AGENT_PROMETHEUS_KEA_EXPORTER_ADDRESS]

``--prometheus-kea-exporter-port=``
   the port to listen on for incoming Prometheus connection (default: 9547)
   [$STORK_AGENT_PROMETHEUS_KEA_EXPORTER_PORT]

``--prometheus-kea-exporter-interval=``
   specifies how often the agent collects stats from Kea, in seconds (default: 10)
   [$STORK_AGENT_PROMETHEUS_KEA_EXPORTER_INTERVAL]

Prometheus BIND 9 Exporter flags:

``--prometheus-bind9-exporter-host=``
   the IP or hostname to listen on for incoming Prometheus connection (default: 0.0.0.0)
   [$STORK_AGENT_PROMETHEUS_BIND9_EXPORTER_ADDRESS]

``--prometheus-bind9-exporter-port=``
   the port to listen on for incoming Prometheus connection (default: 9119)
   [$STORK_AGENT_PROMETHEUS_BIND9_EXPORTER_PORT]

``--prometheus-bind9-exporter-interval=``
   specifies how often the agent collects stats from BIND 9, in seconds (default: 10)
   [$STORK_AGENT_PROMETHEUS_BIND9_EXPORTER_INTERVAL]

``-h`` or ``--help``
   Displays list of available parameters.


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
