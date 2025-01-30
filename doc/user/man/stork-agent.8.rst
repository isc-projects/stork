..
   Copyright (C) 2019-2025 Internet Systems Consortium, Inc. ("ISC")

   This Source Code Form is subject to the terms of the Mozilla Public
   License, v. 2.0. If a copy of the MPL was not distributed with this
   file, You can obtain one at http://mozilla.org/MPL/2.0/.

   See the COPYRIGHT file distributed with this work for additional
   information regarding copyright ownership.

.. _man-stork-agent:

``stork-agent`` - The Stork Agent to Monitor BIND 9 and Kea Services
--------------------------------------------------------------------

Synopsis
~~~~~~~~

:program:`stork-agent` [**--listen-stork-only**] [**--listen-prometheus-only**] [**-v**] [**--host=**] [**--port=**] [**--skip-tls-cert-verification=**] [**--prometheus-kea-exporter-address=**] [**--prometheus-kea-exporter-port=**] [**--prometheus-kea-exporter-interval=**] [**-h**]

:program:`stork-agent` register [**--server-url=**] [**--server-token**] [**--agent-host=**] [**--agent-port=**] [**--non-interactive**]

Description
~~~~~~~~~~~

The ``stork-agent`` is a tool that operates on systems that are running BIND 9
or Kea services. The Stork server typically connects to the Stork agent and uses it to
monitor services remotely, but can also act as a stand-alone statistics exporter to
Prometheus.

Arguments
~~~~~~~~~



The Stork agent's behavior can be controlled with command-line switches and/or
environment variables. The environment variables can be set before running
the agent, or they can be loaded from a file using the ``--use-env-file``
and ``--env-file`` flags. ``stork-agent`` takes the following arguments
(equivalent environment variables are listed in square brackets,
where applicable)

``--listen-stork-only``
   Instructs ``stork-agent`` to listen for commands from the Stork server, but not for Prometheus requests. ``[$STORK_AGENT_LISTEN_STORK_ONLY]``

``--listen-prometheus-only``
   Instructs ``stork-agent`` to listen for Prometheus requests, but not for commands from the Stork server. ``[$STORK_AGENT_LISTEN_PROMETHEUS_ONLY]``

``--hook-directory``
   The path to the hook directory. ``[$STORK_AGENT_HOOK_DIRECTORY]``

``--bind9-path``
   The path to the BIND 9 configuration file. Does not need to be specified, unless the location is uncommon. ``[$STORK_AGENT_BIND9_PATH]``

``--env-file``
   The environment file location; applicable only if the ``use-env-file`` is provided. The default is ``/etc/stork/agent.env``.

``--use-env-file``
   Instructs ``stork-agent`` to read the environment variables from the environment file. The default is ``false``.

``-v|--version``
   Returns the software version.

``-h|--help``
   Returns the list of available parameters.

Stork server flags:

``--server-url=``
   Specifies the URL of the Stork server receiving the registration request. Optional; can be skipped to suppress automatic registration. ``[$STORK_AGENT_SERVER_URL]``

``--host=``
   Specifies the IP address or hostname to listen on for incoming Stork server connections. ``[$STORK_AGENT_HOST]``

``--port=``
   Specifies the TCP port to listen on for incoming Stork server connections. The default is 8080. ``[$STORK_AGENT_PORT]``

``--skip-tls-cert-verification=``
   Indicates that TLS certificate verification should be skipped when the Stork agent makes HTTP calls over TLS. The default is ``false``. ``[$STORK_AGENT_SKIP_TLS_CERT_VERIFICATION]``

Prometheus Kea Exporter flags:

``--prometheus-kea-exporter-address=``
   Specifies the IP address or hostname on which the Stork agent exports Kea statistics to Prometheus. The default is 0.0.0.0. ``[$STORK_AGENT_PROMETHEUS_KEA_EXPORTER_ADDRESS]``

``--prometheus-kea-exporter-port=``
   Specifies the port on which the Stork agent exports Kea statistics to Prometheus. The default is 9547. ``[$STORK_AGENT_PROMETHEUS_KEA_EXPORTER_PORT]``

``--prometheus-kea-exporter-interval=``
   Specifies how often the Stork agent collects statistics from Kea, in seconds. The default is 10. ``[$STORK_AGENT_PROMETHEUS_KEA_EXPORTER_INTERVAL]``

``--prometheus-kea-exporter-per-subnet-stats=``
   Enables or disables collecting per-subnet stats from Kea. The default is true. ``[$STORK_AGENT_PROMETHEUS_KEA_EXPORTER_PER_SUBNET_STATS]``

Prometheus BIND 9 Exporter flags:

``--prometheus-bind9-exporter-address=``
   Specifies the IP address or hostname on which the Stork agent exports BIND 9 statistics to Prometheus. The default is 0.0.0.0. ``[$STORK_AGENT_PROMETHEUS_BIND9_EXPORTER_ADDRESS]``

``--prometheus-bind9-exporter-port=``
   Specifies the port on which the Stork agent exports BIND 9 statistics to Prometheus. The default is 9119. ``[$STORK_AGENT_PROMETHEUS_BIND9_EXPORTER_PORT]``

Stork logs at INFO level by default. Other levels can be configured using the
``STORK_LOG_LEVEL`` variable. Allowed values are: DEBUG, INFO, WARN, ERROR.

To control the logging colorization, Stork supports the ``CLICOLOR`` and
``CLICOLOR_FORCE`` standard UNIX environment variables. Use ``CLICOLOR_FORCE`` to
enforce enabling or disabling ANSI colors usage. Set ``CLICOLOR`` to ``0`` or
``false`` to disable colorization even if the TTY is attached.

Stork evaluates and prioritizes the settings it receives based on where they are applied.
Command-line flags have the highest priority; next are parameters from the
environment file, if the ``--use-env-file`` flag is used. The lowest priority is given
to environment variables.

Examples
~~~~~~~~

To start the Stork agent and register it automatically with the Stork server, run the following command:

.. code-block:: bash

   $ stork-agent --server-url=http://stork-server.example.com:8080 --host=stork-agent.example.com --port=8080

If the Stork agent is already registered with the Stork server, it can be started with this command:

.. code-block:: bash

   $ stork-agent --host=stork-agent.example.com --port=8080

By default, the Stork agent receives server requests and exports metrics to Prometheus. To only listen to the
Stork server, run the following command:

.. code-block:: bash

   $ stork-agent (...) --listen-stork-only

To only listen to Prometheus requests, run the following command:

.. code-block:: bash

   $ stork-agent (...) --listen-prometheus-only

If performance issues are observed with exporting Kea statistics to Prometheus, the interval between
statistics collection can be increased, or collection of per-subnet stats can be disabled. For example:

.. code-block:: bash

   $ stork-agent (...) --prometheus-kea-exporter-interval=30 --prometheus-kea-exporter-per-subnet-stats=false

By default, the Stork agent reads arguments only from the command line. To read arguments from the environment
file, run the following command:

.. code-block:: bash

   $ stork-agent --use-env-file

The default environment file location is ``/etc/stork/agent.env``. To specify a different location, run the following
command:

.. code-block:: bash

   $ stork-agent --use-env-file --env-file=/path/to/agent.env

Registration
~~~~~~~~~~~~

The ``register`` command runs the agent registration using a specified server token and exits.
After successful registration, run the agent normally. The ``register`` command takes the
following arguments:

``-u|--server-url=``
   Specifies the URL of the Stork server receiving the registration request. ``[$STORK_AGENT_SERVER_URL]``

``-t|--server-token=``
   Specifies the access token used by the Stork server to allow registration of the Stork agents. ``[$STORK_AGENT_SERVER_TOKEN]``

``-a|--agent-host=``
   Specifies an IP address or DNS name of the host where the agent is running, e.g. localhost or 10.11.12.13. ``[$STORK_AGENT_HOST]``

``-p|--agent-port=``
   Specifies the port on which the agent listens for incoming connections. The default is 8080. ``[$STORK_AGENT_PORT]``

``-n|--non-interactive``
   Disables the interactive mode. The default is false. ``[$STORK_AGENT_NON_INTERACTIVE]``

To register the Stork agent in interactive mode, run the following command:

.. code-block:: bash

   $ stork-agent register
   >>> Enter the URL of the Stork server: 
   >>> Enter the Stork server access token (optional):
   >>> Enter IP address or FQDN of the host with Stork agent (for the Stork server connection) [hostname]: 
   >>> Enter port number that Stork Agent will listen on [8080]: 

To register the Stork agent with the server token, providing all the necessary information through CLI arguments, run the following command:

.. code-block:: bash

   $ stork-agent register --server-url=http://stork-server.example.com:8080 --server-token=1234567890 --agent-host=stork-agent.example.com --agent-port=8080

To register the Stork agent without the server token, using the environment variables, run the following commands:

.. code-block:: bash

   $ export STORK_AGENT_SERVER_URL=http://stork-server.example.com:8080
   $ export STORK_AGENT_HOST=stork-agent.example.com
   $ export STORK_AGENT_PORT=8080
   $ stork-agent register

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
