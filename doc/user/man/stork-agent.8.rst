..
   Copyright (C) 2019-2024 Internet Systems Consortium, Inc. ("ISC")

   This Source Code Form is subject to the terms of the Mozilla Public
   License, v. 2.0. If a copy of the MPL was not distributed with this
   file, You can obtain one at http://mozilla.org/MPL/2.0/.

   See the COPYRIGHT file distributed with this work for additional
   information regarding copyright ownership.

.. _man-stork-agent:

``stork-agent`` - Stork Agent to Monitor BIND 9 and Kea services
----------------------------------------------------------------

Synopsis
~~~~~~~~

:program:`stork-agent` [**--listen-stork-only**] [**--listen-prometheus-only**] [**-v**] [**--host=**] [**--port=**] [**--skip-tls-cert-verification=**] [**--prometheus-kea-exporter-address=**] [**--prometheus-kea-exporter-port=**] [**--prometheus-kea-exporter-interval=**] [**-h**]

:program:`stork-agent` register [**--server-url=**] [**--server-token**] [**--agent-host=**] [**--agent-port=**]

Description
~~~~~~~~~~~

The ``stork-agent`` is a small tool that operates on systems
that are running BIND 9 or Kea services. The Stork server connects to
the Stork agent and uses it to monitor services remotely.

Arguments
~~~~~~~~~

Stork does not use an explicit configuration file. Instead, its behavior can be controlled with
command-line switches and/or variables. The Stork agent takes the following command-line switches.
Equivalent environment variables are listed in square brackets, where applicable.

``--listen-stork-only``
   Instructs ``stork-agent`` to listen for commands from the Stork server, but not for Prometheus requests. ``[$STORK_AGENT_LISTEN_STORK_ONLY]``

``--listen-prometheus-only``
   Instructs ``stork-agent`` to listen for Prometheus requests, but not for commands from the Stork server. ``[$STORK_AGENT_LISTEN_PROMETHEUS_ONLY]``

``--hook-directory``
   The path to the hook directory. ``[$STORK_AGENT_HOOK_DIRECTORY]``

``--env-file``
   Environment file location; applicable only if the use-env-file is provided. The default is ``/etc/stork/agent.env``.

``--use-env-file``
   Read the environment variables from the environment file. The default is ``false``.

``-v|--version``
   Returns the software version.

``-h`` or ``--help``
   Returns the list of available parameters.

Stork server flags:

``--host=``
   Specifies the IP address or hostname to listen on for incoming Stork server connections. ``[$STORK_AGENT_HOST]``

``--port=``
   Specifies the TCP port to listen on for incoming Stork server connections. The default is 8080. ``[$STORK_AGENT_PORT]``

``--skip-tls-cert-verification=``
   Indicates that TLS certificate verification should be skipped when the Stork agent makes HTTP calls over TLS. The default is ``false``. ``[$STORK_AGENT_SKIP_TLS_CERT_VERIFICATION]``

Prometheus Kea Exporter flags:

``--prometheus-kea-exporter-address=``
   Specifies the IP address or hostname on which the agent exports Kea statistics to Prometheus. The default is 0.0.0.0. ``[$STORK_AGENT_PROMETHEUS_KEA_EXPORTER_ADDRESS]``

``--prometheus-kea-exporter-port=``
   Specifies the port on which the agent exports Kea statistics to Prometheus. The default is 9547. ``[$STORK_AGENT_PROMETHEUS_KEA_EXPORTER_PORT]``

``--prometheus-kea-exporter-interval=``
   Specifies how often the agent collects statistics from Kea, in seconds. The default is 10. ``[$STORK_AGENT_PROMETHEUS_KEA_EXPORTER_INTERVAL]``

``--prometheus-kea-exporter-per-subnet-stats=``
   Enable or disable collecting per subnet stats from Kea. The default is true. ``[$STORK_AGENT_PROMETHEUS_KEA_EXPORTER_PER_SUBNET_STATS]``

Prometheus BIND 9 Exporter flags:

``--prometheus-bind9-exporter-address=``
   Specifies the IP address or hostname on which the agent exports BIND 9 statistics to Prometheus. The default is 0.0.0.0. ``[$STORK_AGENT_PROMETHEUS_BIND9_EXPORTER_ADDRESS]``

``--prometheus-bind9-exporter-port=``
   Specifies the port on which the agent exports BIND 9 statistics to Prometheus. The default is 9119. ``[$STORK_AGENT_PROMETHEUS_BIND9_EXPORTER_PORT]``

Stork logs at INFO level by default. Other levels can be configured using the
``STORK_LOG_LEVEL`` variable. Allowed values are: DEBUG, INFO, WARN, ERROR.

To control the logging colorization, Stork supports the ``CLICOLOR`` and
``CLICOLOR_FORCE`` standard UNIX environment variables. Use ``CLICOLOR_FORCE`` to
enforce enabling or disabling the ANSI colors usage. Set ``CLICOLOR`` to ``0`` or
``false`` to disable colorization even if the TTY is attached.

The highest priority always have the command line flags. The parameters from the
environment file take precedence over the environment variables if the
``--use-env-file`` flag is used.

Registration
~~~~~~~~~~~~

The ``register`` command runs the agent registration using a specified server token and exits.
After the successful registration, run the agent normally. The ``register`` command takes the
following arguments:

``-u|--server-url=``
   Specifies a URL of the Stork Server receiving the registration request. ``[$STORK_AGENT_SERVER_URL]``

``-t|--server-token=``
   Specifies the access token used by the Stork server to allow registration of the Stork agents. ``[$STORK_AGENT_SERVER_TOKEN]``

``-a|--agent-host=``
   Specifies an IP address or DNS name the host where the agent is running. E.g.: localhost or 10.11.12.13. ``[$STORK_AGENT_HOST]``

``-p|--agent-port=``
   Specifies the port on which the agent listens for incoming connections. The default is 8080. ``[$STORK_AGENT_PORT]``

``-n|--non-interactive``
   Disables the interactive mode. The default is false. ``[$STORK_AGENT_NON_INTERACTIVE]``

``-v|--version``
   Returns the software version.

``-h|--help``
   Returns register command's parameters.

``-h|--help``
   Returns register command's parameters.

Mailing Lists and Support
~~~~~~~~~~~~~~~~~~~~~~~~~

There are public mailing lists available for the Stork project. **stork-users**
(stork-users at lists.isc.org) is intended for Stork users. **stork-dev**
(stork-dev at lists.isc.org) is intended for Stork developers, prospective
contributors, and other advanced users. The lists are available at
https://www.isc.org/mailinglists. The community provides best-effort support
on both of those lists.

History
~~~~~~~

``stork-agent`` was first coded in November 2019 by Michal Nowikowski.

See Also
~~~~~~~~

:manpage:`stork-server(8)`
