..
   Copyright (C) 2019-2021 Internet Systems Consortium, Inc. ("ISC")

   This Source Code Form is subject to the terms of the Mozilla Public
   License, v. 2.0. If a copy of the MPL was not distributed with this
   file, You can obtain one at http://mozilla.org/MPL/2.0/.

   See the COPYRIGHT file distributed with this work for additional
   information regarding copyright ownership.

.. _man-stork-agent:

stork-agent - Stork agent that monitors BIND 9 and Kea services
---------------------------------------------------------------

Synopsis
~~~~~~~~

:program:`stork-agent` [**--host**] [**--port**]

Description
~~~~~~~~~~~

The ``stork-agent`` is a small tool that operates on systems
that are running BIND 9 and Kea services. The Stork server connects to
the Stork agent and uses it to monitor services remotely.

Arguments
~~~~~~~~~

Stork does not use an explicit configuration file. Instead, its behavior can be controlled with
command-line switches and/or variables. The Stork agent takes the following command-line switches.
Equivalent environment variables are listed in square brackets, where applicable.

``--listen-stork-only``
   listen for commands from the Stork server only, but not for Prometheus requests.
   [$STORK_AGENT_LISTEN_STORK_ONLY]

``--listen-prometheus-only``
   listen for Prometheus requests only, but not for commands from the Stork server.
   [$STORK_AGENT_LISTEN_PROMETHEUS_ONLY]

``-v`` or ``--version``
   show software version.

``Stork Server`` flags:

``--host=``
   the IP or hostname to listen on for incoming Stork server connections. [$STORK_AGENT_ADDRESS]

``--port=``
   the TCP port to listen on for incoming Stork server connections. (default: 8080) [$STORK_AGENT_PORT]

``--skip-tls-cert-verification=``
   skip TLS certificate verification when the Stork Agent connects to Kea over TLS and Kea uses self-signed certificates. (default: false) [$STORK_AGENT_SKIP_TLS_CERT_VERIFICATION]

``Prometheus Kea Exporter`` flags:

``--prometheus-kea-exporter-address=``
   the IP or hostname on which the agent exports Kea statistics to Prometheus. (default: 0.0.0.0)
   [$STORK_AGENT_PROMETHEUS_KEA_EXPORTER_ADDRESS]

``--prometheus-kea-exporter-port=``
   the port to listen on for incoming Prometheus connections. (default: 9547)
   [$STORK_AGENT_PROMETHEUS_KEA_EXPORTER_PORT]

``--prometheus-kea-exporter-interval=``
   how often the agent collects stats from Kea, in seconds. (default: 10)
   [$STORK_AGENT_PROMETHEUS_KEA_EXPORTER_INTERVAL]

``Prometheus BIND 9 Exporter`` flags:

``--prometheus-bind9-exporter-address=``
   the IP or hostname on which the agent exports BIND9 statistics to Prometheus. (default: 0.0.0.0)
   [$STORK_AGENT_PROMETHEUS_BIND9_EXPORTER_ADDRESS]

``--prometheus-bind9-exporter-port=``
   the port to listen on for incoming Prometheus connections. (default: 9119)
   [$STORK_AGENT_PROMETHEUS_BIND9_EXPORTER_PORT]

``--prometheus-bind9-exporter-interval=``
   how often the agent collects stats from BIND 9, in seconds. (default: 10)
   [$STORK_AGENT_PROMETHEUS_BIND9_EXPORTER_INTERVAL]

``-h`` or ``--help``
   the list of available parameters.


Mailing Lists and Support
~~~~~~~~~~~~~~~~~~~~~~~~~

There are public mailing lists available for the Stork project. **stork-users**
(stork-users at lists.isc.org) is intended for Stork users. **stork-dev**
(stork-dev at lists.isc.org) is intended for Stork developers, prospective
contributors, and other advanced users. The lists are available at
https://lists.isc.org. The community provides best-effort support
on both of those lists.

Once Stork becomes more mature, ISC will provide professional support
for Stork services.

History
~~~~~~~

The ``stork-agent`` was first coded in November 2019 by Michal Nowikowski.

See Also
~~~~~~~~

:manpage:`stork-server(8)`
