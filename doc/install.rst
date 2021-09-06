.. _installation:

************
Installation
************

Stork can be installed from pre-built packages or from sources. The following sections describe both methods. Unless there's a
good reason to compile from sources, installing from native deb or RPM packages is easier and faster.

.. _supported_systems:

Supported Systems
=================

Stork is tested on the following systems:

- Ubuntu 18.04 and 20.04
- Fedora 31 and 32
- CentOS 7
- MacOS 10.15*

Note that MacOS is not and will not be officially supported. Many developers on ISC's team use Macs, so the goal is to keep Stork
buildable on this platform.

The Stork server and agents are written in the Go language; the server uses a PostgreSQL database. In principle, the software can be run
on any POSIX system that has a Go compiler and PostgreSQL. It is likely the software can also be built on other modern systems, but
for the time being ISC's testing capabilities are modest. We encourage users to try running Stork on other OSes not on this list
and report their findings to ISC.

Installation Prerequisites
==========================

The ``Stork Agent`` does not require any specific dependencies to run. It can be run immediately after installation.

Stork uses the `status-get` command to communicate with Kea, and therefore only works with a version of Kea that supports
`status-get`, which was introduced in Kea 1.7.3 and backported to 1.6.3.

Stork requires the premium ``Host Commands (host_cmds)`` hooks library to be loaded by the Kea instance to retrieve host
reservations stored in an external database. Stork does work without the Host Commands hooks library, but will not be able to display
host reservations. Stork can retrieve host reservations stored locally in the Kea configuration without any additional hooks
libraries.

Stork requires the open source ``Stat Commands (stat_cmds)`` hooks library to be loaded by the Kea instance to retrieve lease
statistics. Stork does work without the Stat Commands hooks library, but will not be able to show pool utilization and other
statistics.

Stork uses Go implementation for handling TLS connections, certificates and keys. The secrets are stored in the PostgreSQL
database, in the `secret` table.

For the ``Stork Server``, a PostgreSQL database (https://www.postgresql.org/) version 10
or later is required. Stork will attempt to run with older versions but may not work
correctly. The general installation procedure for PostgreSQL is OS-specific and is not included
here. However, please note that Stork uses pgcrypto extensions, which often come in a separate package. For
example, a postgresql-crypto package is required on Fedora and postgresql12-contrib is needed on RHEL and CentOS.

These instructions prepare a database for use with the ``Stork
Server``, with the `stork` database user and `stork` password.  Next,
a database called `stork` is created and the `pgcrypto` extension is
enabled in the database.

First, connect to PostgreSQL using `psql` and the `postgres`
administration user. Depending on the system's configuration, this may require
switching to the user `postgres` first, using the `su postgres` command.

.. code-block:: console

    $ psql postgres
    psql (11.5)
    Type "help" for help.
    postgres=#

Then, prepare the database:

.. code-block:: psql

    postgres=# CREATE USER stork WITH PASSWORD 'stork';
    CREATE ROLE
    postgres=# CREATE DATABASE stork;
    CREATE DATABASE
    postgres=# GRANT ALL PRIVILEGES ON DATABASE stork TO stork;
    GRANT
    postgres=# \c stork
    You are now connected to database "stork" as user "thomson".
    stork=# create extension pgcrypto;
    CREATE EXTENSION

.. note::

   Make sure the actual password is stronger than 'stork', which is trivial to guess.
   Using default passwords is a security risk. Stork puts no restrictions on the
   characters used in the database passwords nor on their length. In particular,
   it accepts passwords containing spaces, quotes, double quotes, and other
   special characters.

.. _install-pkgs:

Installing from Packages
========================

Stork packages are stored in repositories located on the Cloudsmith
service: https://cloudsmith.io/~isc/repos/stork/packages/. Both
Debian/Ubuntu and RPM packages may be found there.

Detailed instructions for setting up the operating system to use this
repository are available under the `Set Me Up` button on the
Cloudsmith repository page.

It is possible to install both ``Stork Agent`` and ``Stork Server`` on
the same machine. It is useful in small deployments with a single
monitored machine to avoid setting up a dedicated system for the Stork
Server. In those cases, however, an operator must consider the possible
impact of the Stork Server service on other services running on the same
machine.


Installing the Stork Server
---------------------------

.. _install-server-deb:

Installing on Debian/Ubuntu
~~~~~~~~~~~~~~~~~~~~~~~~~~~

The first step for both Debian and Ubuntu is:

.. code-block:: console

   $ curl -1sLf 'https://dl.cloudsmith.io/public/isc/stork/cfg/setup/bash.deb.sh' | sudo bash

Next, install the ``Stork Server`` package:

.. code-block:: console

   $ sudo apt install isc-stork-server

.. _install-server-rpm:

Installing on CentOS/RHEL/Fedora
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

The first step for RPM-based distributions is:

.. code-block:: console

   $ curl -1sLf 'https://dl.cloudsmith.io/public/isc/stork/cfg/setup/bash.rpm.sh' | sudo bash

Next, install the ``Stork Server`` package:

.. code-block:: console

   $ sudo dnf install isc-stork-server

If ``dnf`` is not available, ``yum`` can be used instead:

.. code-block:: console

   $ sudo yum install isc-stork-server

Setup
~~~~~

The following steps are common for Debian-based and RPM-based distributions
using `systemd`.

Configure ``Stork Server`` settings in ``/etc/stork/server.env``. The following
settings are required for the database connection:

* STORK_DATABASE_HOST - the address of a PostgreSQL database; default is `localhost`
* STORK_DATABASE_PORT - the port of a PostgreSQL database; default is `5432`
* STORK_DATABASE_NAME - the name of a database; default is `stork`
* STORK_DATABASE_USER_NAME - the username for connecting to the database; default is `stork`
* STORK_DATABASE_PASSWORD - the password for the username connecting to the database

.. note::

   All of the database connection settings have default values, but we strongly
   recommend protecting the database with a non-default and hard-to-guess password
   in the production environment. The `STORK_DATABASE_PASSWORD` setting must be
   adjusted accordingly.

The remaining settings pertain to the server's REST API configuration:

* STORK_REST_HOST - IP address on which the server listens
* STORK_REST_PORT - port number on which the server listens; default is `8080`
* STORK_REST_TLS_CERTIFICATE - a file with a certificate to use for secure connections
* STORK_REST_TLS_PRIVATE_KEY - a file with a private key to use for secure connections
* STORK_REST_TLS_CA_CERTIFICATE - a certificate authority file used for mutual TLS authentication
* STORK_REST_STATIC_FILES_DIR - a directory with static files served in the UI

With the settings in place, the ``Stork Server`` service can now be enabled and
started:

.. code-block:: console

   $ sudo systemctl enable isc-stork-server
   $ sudo systemctl start isc-stork-server

To check the status:

.. code-block:: console

   $ sudo systemctl status isc-stork-server


.. note::

   By default, the ``Stork Server`` web service is exposed on port 8080 and
   can be tested using web browser at http://localhost:8080. To use a different IP address or port,
   please set the `STORK_REST_HOST` and `STORK_REST_PORT` variables in the ``/etc/stork/stork.env``
   file.

The ``Stork Server`` can be configured to run behind an HTTP reverse proxy
using `Nginx` or `Apache`. The ``Stork Server`` package contains an example
configuration file for `Nginx`, in `/usr/share/stork/examples/nginx-stork.conf`.

Installing the Stork Agent
--------------------------

There are two ways to install packaged ``Stork Agent`` on a monitored machine.
The first method is to use the Cloudsmith repository like in the case of the
``Stork Server`` installation. The second method is to use an installation
script provided by the ``Stork Server`` which downloads the agent packages
embedded in the server package. The second installation method is supported
since the Stork 0.15.0 release. The preferred installation method depends on
the selected agent registration type. Supported registration methods are
described in the :ref:`secure-server-agent`.

Agent Configuration Settings
~~~~~~~~~~~~~~~~~~~~~~~~~~~~

The following are the ``Stork Agent`` configuration settings available in the
``/etc/stork/agent.env`` after installing the package.

The general settings:

* STORK_AGENT_ADDRESS - the IP address of the network interface which ``Stork Agent``
  should use to receive the connections from the server;  default is `0.0.0.0`
  (i.e. listen on all interfaces)
* STORK_AGENT_PORT - the port number the agent should use to receive the
  connections from the server;  default is `8080`
* STORK_AGENT_LISTEN_STORK_ONLY - enable Stork functionality only,
  i.e. disable Prometheus exporters; default is false
* STORK_AGENT_LISTEN_PROMETHEUS_ONLY - enable Prometheus exporters
  only, i.e. disable Stork functionality; default is false

The following settings are specific to the Prometheus exporters:

* STORK_AGENT_PROMETHEUS_KEA_EXPORTER_ADDRESS - the IP address or hostname the
  agent should use to receive the connections from Prometheus fetching Kea
  statistics; default is `0.0.0.0`
* STORK_AGENT_PROMETHEUS_KEA_EXPORTER_PORT - the port the agent should use to
  receive the connections from Prometheus fetching Kea statistics; default is
  `9547`
* STORK_AGENT_PROMETHEUS_KEA_EXPORTER_INTERVAL - specifies how often
  the agent collects stats from Kea, in seconds; default is `10`
* STORK_AGENT_PROMETHEUS_BIND9_EXPORTER_ADDRESS - the IP address or hostname the
  agent should use to receive the connections from Prometheus fetching BIND9
  statistics; default is `0.0.0.0`
  to listen on for incoming Prometheus connection; default is `0.0.0.0`
* STORK_AGENT_PROMETHEUS_BIND9_EXPORTER_PORT - the port the agent should use to
  receive the connections from Prometheus fetching BIND9 statistics; default is
  `9119`
* STORK_AGENT_PROMETHEUS_BIND9_EXPORTER_INTERVAL - specifies how often
  the agent collects stats from BIND9, in seconds; default is `10`

The last setting is used only when ``Stork Agents`` register in the ``Stork Server``
using agent token:

* STORK_AGENT_SERVER_URL - Stork Server URL used by the agent to send REST
  commands to the server during agent registration

.. _secure-server-agent:

Securing Connections Between Stork Server and Stork Agents
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

Connections between the server and the agents are secured using
standard cryptography solutions, i.e. PKI and TLS.

The server generates the required keys and certificates during its first startup.
They are used to establish safe, encrypted connections between the server
and the agents with authentication of both ends of these connections.
The agents use the keys and certificates generated by the server to
create agent-side keys and certificates during the agents' registration
procedure described in the next sections. The private key and CSR
certificate generated by an agent and signed by the server are used for
authentication and connection encryption.

An agent can be registered in the server using one of the two supported
methods:

#. using agent token,
#. using server token.

In the first case, an agent generates a token and passes it to the server
requesting registration. The server associates the token with the particular
agent. A Stork super admin must approve the registration request in the web UI,
ensuring that the token displayed in the UI matches the agent's token in the
logs. The ``Stork Agent`` is typically installed from the Cloudsmith repository
when this registration method is used.

In the second registration method, a server generates a token common for all
new registrations. The super admin must copy the token from the UI and paste
it into the agent's terminal during the interactive agent registration procedure.
This registration method does not require any additional approval of the agent's
registration request in the web UI. If the pasted server token is correct,
the agent should be authorized in the UI when the interactive registration
completes. The ``Stork Agent`` is typically installed using a script that
downloads the agent packages embedded in the server when this registration
method is used.

The applicability of the two methods is described in
:ref:`registration-methods-summary`.

The installation and registration process using both methods are described
in the subsequent sections.

.. _register-agent-token-cloudsmith:

Installation from Cloudsmith and Registration with an Agent Token
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

This section describes installing an agent from the Cloudsmith repository and
performing the agent's registration using an agent token.

The ``Stork Agent`` installation steps are similar to the ``Stork Server``
installation steps described in :ref:`install-server-deb` and
:ref:`install-server-rpm`. Use one of the following commands depending on
your Linux distribution:

.. code-block:: console

   $ sudo apt install isc-stork-agent

.. code-block:: console

   $ sudo dnf install isc-stork-agent

in place of the commands installing the server.

Next, specify the required settings in the ``/etc/stork/agent.env`` file.
The ``STORK_SERVER_URL`` should be the URL on which the server receives the
REST connections, e.g. ``http://stork-server.example.org:8080``. The
``STORK_AGENT_ADDRESS`` should point to the agent's address (or name), e.g.
``stork-agent.example.org``. Finally, a non-default agent port can be
specified with the ``STORK_AGENT_PORT``.

.. note::

   Even though the examples provided in this documentation use the ``http``
   scheme, we highly recommend using secure protocols in the production
   environments. We use ``http`` in the examples because it usually
   makes it easier to start testing the software and eliminate all issues
   unrelated to the use of ``https`` before it is enabled.

Start the agent service:

.. code-block:: console

   $ sudo systemctl enable isc-stork-agent
   $ sudo systemctl start isc-stork-agent

To check the status:

.. code-block:: console

   $ sudo systemctl status isc-stork-agent

You should expect the following log messages when the agent successfully
sends the registration request to the server:

.. code-block:: text

    machine registered
    stored agent signed cert and CA cert
    registration completed successfully

A server administrator must approve the registration request via the
web UI before the machine can be monitored. Visit the ``Services -> Machines``
page. Click the ``Unauthorized`` button located above the list of machines
on the right side. This list contains all machines pending registration approval.
Before authorizing the machine, ensure that the agent token displayed on this
list is the same as the agent token in the agent's logs or the
``/var/lib/stork-agent/tokens/agent-token.txt`` file. If they match,
click on the ``Action`` button and select ``Authorize``. The machine
should now be visible on the list of authorized machines.

.. _register-server-token-script:

Installation with a Script and Registration with a Server Token
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

This section describes installing an agent using a script and packages
downloaded from the ``Stork Server`` and performing the agent's
registration using a server token.

Open Stork in the web browser and log in as a user from the super admin group.
Select ``Services`` and then ``Machines`` from the menu. Click on the
``How to Install Agent on New Machine`` button to display the agent
installation instructions. Copy-paste the commands from the displayed
window into the terminal on the machine where the agent is installed.
These commands are also provided here for convenience:

.. code-block:: console

   $ wget http://stork.example.org:8080/stork-install-agent.sh
   $ chmod a+x stork-install-agent.sh
   $ sudo ./stork-install-agent.sh

Please note that this document provides an example URL of the ``Stork Server``
and it must be replaced with a server URL used in the particular deployment.

The script downloads an OS specific agent package from the ``Stork Server``
(deb or RPM), installs the package, and starts the agent's registration procedure.

In the agent machine's terminal, a prompt for a server token is presented:

.. code-block:: text

    >>>> Server access token (optional):

The server token is available for a super admin user after clicking on the
``How to Install Agent on New Machine`` button in the ``Services -> Machines``.
Copy the server token from the dialog box and paste it in the prompt
displayed on the agent machine.

The following prompt appears next:

.. code-block:: text

    >>>> IP address or FQDN of the host with Stork Agent (the Stork Server will use it to connect to the Stork Agent):

Specify an IP address or FQDN which the server should use to reach out to an
agent via the secure gRPC channel.

When asked for the port:

.. code-block:: text

   >>>> Port number that Stork Agent will use to listen on [8080]:

specify the port number for the gRPC connections, or hit Enter if the
default port 8080 matches your settings.

If the registration is successful, the following messages are displayed:

.. code-block:: text

   machine ping over TLS: OK
   registration completed successfully


Unlike the :ref:`register-agent-token-cloudsmith`, this registration method
does not require approval via the web UI. The machine should be
already listed among the authorized machines.

.. _register-agent-token-script:

Installation with a Script and Registration with an Agent Token
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

This section describes installing an agent using a script and packages downloaded from
the ``Stork Server`` and performing the agent's registration using an agent token. It
is an interactive procedure alternative to the procedure described in
:ref:`register-agent-token-cloudsmith`.

Start the interactive registration procedure following the steps in
the :ref:`register-server-token-script`.

In the agent machine's terminal, a prompt for a server token is presented:

.. code-block:: text

    >>>> Server access token (optional):

Because this registration method does not use the server token, do not type anything
in this prompt. Hit Enter to move on.

The following prompt appears next:

.. code-block:: text

    >>>> IP address or FQDN of the host with Stork Agent (the Stork Server will use it to connect to the Stork Agent):

Specify an IP address or FQDN which the server should use to reach out to an
agent via the secure gRPC channel.

When asked for the port:

.. code-block:: text

   >>>> Port number that Stork Agent will use to listen on [8080]:

specify the port number for the gRPC connections, or hit Enter if the
default port 8080 matches your settings.

You should expect the following log messages when the agent successfully
sends the registration request to the server:

.. code-block:: text

    machine registered
    stored agent signed cert and CA cert
    registration completed successfully

Similar to :ref:`register-agent-token-cloudsmith`, the agent's registration
request must be approved in the UI to start monitoring the newly registered
machine.

.. _register-server-token-cloudsmith:

Installation from Cloudsmith and Registration with a Server Token
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

This section describes installing an agent from the Cloudsmith repository and
performing the agent's registration using a server token. It is an alternative to
the procedure described in :ref:`register-server-token-script`.

The ``Stork Agent`` installation steps are similar to the ``Stork Server``
installation steps described in :ref:`install-server-deb` and
:ref:`install-server-rpm`. Use one of the following commands depending on
your Linux distribution:

.. code-block:: console

   $ sudo apt install isc-stork-agent

.. code-block:: console

   $ sudo dnf install isc-stork-agent

in place of the commands installing the server.

Start the agent service:

.. code-block:: console

   $ sudo systemctl enable isc-stork-agent
   $ sudo systemctl start isc-stork-agent

To check the status:

.. code-block:: console

   $ sudo systemctl status isc-stork-agent

Start the interactive registration procedure with the following command:

.. code-block:: console

   $ su stork-agent -s /bin/sh -c 'stork-agent register -u http://stork.example.org'

where the last parameter should be the appropriate Stork server's URL.

Follow the same registration steps as described in the :ref:`register-server-token-script`.

.. _registration-methods-summary:

Registration Methods Summary
~~~~~~~~~~~~~~~~~~~~~~~~~~~~

Stork supports two different agents' registration methods described above.
Both methods can be used interchangeably, and it is often a matter of
preference which one the administrator selects. However, it is worth
mentioning that the agent token registration may be more suitable in
some situations. This method requires a server URL, agent address
(or name), and agent port as registration settings. If they are known
upfront, it is possible to prepare a system (or container) image with
the agent offline. After starting the image, the agent will send the
registration request to the server and await authorization in the web UI.

The agent registration with the server token is always manual. It
requires copying the token from the web UI, logging into the agent,
and pasting the token. Therefore, the registration using the server
token is not appropriate when it is impossible or awkward to access
the machine's terminal, e.g. in Docker. On the other hand, the
registration using the server token is more straightforward because
it does not require unauthorized agents' approval via the web UI.

If the server token leaks, it poses a risk that rogue agents register.
In that case, the administrator should regenerate the token to prevent
the uncontrolled registration of new agents. Regeneration of the token
does not affect already registered agents. The new token must be used
for the new registrations.

The server token can be regenerated in the ``How to Install Agent on New Machine``
dialog box available after entering the ``Services -> Machines`` page.


Agent Setup Summary
~~~~~~~~~~~~~~~~~~~

After successful agent setup, the agent periodically tries to detect installed
Kea DHCP or BIND 9 services on the system. If it finds them, they are
reported to the ``Stork Server`` when it connects to the agent.

Further configuration and usage of the ``Stork Server`` and the
``Stork Agent`` are described in the :ref:`usage` chapter.


Inspecting Keys and Certificates
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

Stork Server maintains TLS keys and certificates internally for securing
communication between ``Stork Server`` and ``Stork Agents``. They can be inspected
and exported using ``Stork Tool``, e.g:

.. code-block:: console

    $ stork-tool cert-export --db-url postgresql://user:pass@localhost/dbname -f srvcert -o srv-cert.pem

The certificates can be inspected using openssl (e.g. ``openssl x509 -noout -text -in srv-cert.pem``).
Similarly, the secret keys can be inspected in similar fashion (e.g. ``openssl ec -noout -text -in cakey``)

For more details check ``stork-tool`` manual: :ref:`man-stork-tool`. There are five secrets that can be
exported or imported: Certificate Authority secret key (``cakey``), Certificate Authority certificate (``cacert``),
Stork server private key (``srvkey``), Stork server certificate (``srvcert``) and a server token (``srvtkn``).

Using External Keys and Certificates
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

It is possible to use external TLS keys and certificates. They can be imported
to ``Stork Server`` using ``stork-tool``:

.. code-block:: console

    $ stork-tool cert-import --db-url postgresql://user:pass@localhost/dbname -f srvcert -i srv-cert.pem

Both CA key and CA certificate have to be changed at the same time as
CA certificate depends on CA key. If they are changed then server key
and certificate also need to be changed.

The capability to use external certificates and key is considered experimental.

For more details check ``stork-tool`` manual: :ref:`man-stork-tool`.

Upgrading
---------

Due to the new security model introduced with TLS in Stork 0.15.0
release, upgrades from versions 0.14.0 and earlier require registering
the agents from scratch.

Server upgrade procedure looks the same as the installation procedure.

First, install the new packages on the server. Installation scripts in
deb/RPM package will perform the required database and other migrations.

.. _installation_sources:

Installing From Sources
=======================

Compilation Prerequisites
-------------------------

Usually, it is more convenient to install Stork using native packages. See :ref:`supported_systems` and :ref:`install-pkgs` for
details regarding supported systems. However, the sources can also be built separately.

The dependencies that need to be installed to build ``Stork`` sources are:

 - Rake
 - Java Runtime Environment (only if building natively, not using Docker)
 - Docker (only if running in containers; this is needed to build the demo)

Other dependencies are installed automatically in a local directory by Rake tasks. This does not
require root privileges. If the demo environment will be run, Docker is needed but not
Java (Docker will install Java within a container).

For details about the environment, please see the Stork wiki at
https://gitlab.isc.org/isc-projects/stork/-/wikis/Install .

Download Sources
----------------

The Stork sources are available on the ISC GitLab instance:
https://gitlab.isc.org/isc-projects/stork.

To get the latest sources invoke:

.. code-block:: console

   $ git clone https://gitlab.isc.org/isc-projects/stork

Building
--------

There are several components of ``Stork``:

- ``Stork Agent`` - this is the binary `stork-agent`, written in Go
- ``Stork Server`` - this is comprised of two parts:
  - `backend service` - written in Go
  - `frontend` - an `Angular` application written in Typescript

All components can be built using the following command:

.. code-block:: console

   $ rake build_all

The agent component is installed using this command:

.. code-block:: console

   $ rake install_agent

and the server component with this command:

.. code-block:: console

   $ rake install_server

By default, all components are installed to the `root` folder in the
current directory; however, this is not useful for installation in a
production environment. It can be customized via the ``DESTDIR``
variable, e.g.:

.. code-block:: console

   $ sudo rake install_server DESTDIR=/usr

Database Migration Tool (optional)
==================================

Optional step: to initialize the database directly, the migrations
tool must be built and used to initialize and upgrade the database to the
latest schema. However, this is completely optional, as the database
migration is triggered automatically upon server startup. This is
only useful if for some reason it is desirable to set up the database
but not yet run the server. In most cases this step can be skipped.

.. code-block:: console

    $ rake build_tool
    $ backend/cmd/stork-tool/stork-tool db-init
    $ backend/cmd/stork-tool/stork-tool db-up

The up and down commands have an optional `-t` parameter that specifies the desired
schema version. This is only useful when debugging database migrations.

.. code-block:: console

    $ # migrate up version 25
    $ backend/cmd/stork-tool/stork-tool db-up -t 25
    $ # migrate down back to version 17
    $ backend/cmd/stork-tool/stork-tool db-down -t 17

Note that the server requires the latest database version to run, always
runs the migration on its own, and will refuse to start if the migration fails
for any reason. The migration tool is mostly useful for debugging
problems with migration or migrating the database without actually running
the service. For complete reference, see the manual page here:
:ref:`man-stork-tool`.

To debug migrations, another useful feature is SQL tracing using the `--db-trace-queries` parameter.
It takes either "all" (trace all SQL operations, including migrations and run-time) or "run" (just
trace run-time operations, skip migrations). If specified without any parameters, "all" is assumed. With it enabled,
`stork-tool` prints out all its SQL queries on stderr. For example, these commands can be used
to generate an SQL script that updates the schema. Note that for some migrations, the steps are
dependent on the contents of the database, so this is not a universal Stork schema. This parameter
is also supported by the ``Stork Server``.

.. code-block:: console

   $ backend/cmd/stork-tool/stork-tool db-down -t 0
   $ backend/cmd/stork-tool/stork-tool db-up --db-trace-queries 2> stork-schema.txt


Integration With Prometheus and Grafana
=======================================

Stork can optionally be integrated with `Prometheus <https://prometheus.io/>`_, an open source monitoring and alerting toolkit,
and `Grafana <https://grafana.com/>`_, an easy-to-view analytics platform for querying, visualization, and alerting. Grafana
requires external data storage. Prometheus is currently the only environment supported by both Stork and Grafana. It is possible
to use Prometheus without Grafana, but using Grafana requires Prometheus.

Prometheus Integration
----------------------

The Stork agent, by default, makes the
Kea (and eventually, BIND 9) statistics available in a format understandable by Prometheus (it works as a Prometheus exporter, in Prometheus
nomenclature). If Prometheus server is available, it can be configured to monitor Stork agents. To enable Stork agent
monitoring, the ``prometheus.yml`` (which is typically stored in /etc/prometheus/, but this may vary depending on the
installation) must be edited to add the following entries there:

.. code-block:: yaml

  # statistics from Kea
  - job_name: 'kea'
    static_configs:
      - targets: ['agent-kea.example.org:9547', 'agent-kea6.example.org:9547', ... ]

  # statistics from bind9
  - job_name: 'bind9'
    static_configs:
      - targets: ['agent-bind9.example.org:9119', 'another-bind9.example.org:9119', ... ]

By default, the Stork agent exports Kea data on TCP port 9547 (and BIND 9 data on TCP port 9119). This can be configured using
command-line parameters, or the Prometheus export can be disabled altogether. For details, see the stork-agent manual page
at :ref:`man-stork-agent`.

After restarting, the Prometheus web interface can be used to inspect whether statistics are exported properly. Kea statistics use the ``kea_`` prefix (e.g. kea_dhcp4_addresses_assigned_total); BIND 9
statistics will eventually use the ``bind_`` prefix (e.g. bind_incoming_queries_tcp).

Grafana Integration
-------------------

Stork provides several Grafana templates that can easily be imported. Those are available in the ``grafana/`` directory of the
Stork source code. The currently available templates are `bind9-resolver.json` and `kea-dhcp4.json`. Grafana integration requires three steps:

1. Prometheus must be added as a data source. This can be done in several ways, including via the user interface to edit the Grafana
configuration files. This is the easiest method; for details, see the Grafana documentation about Prometheus integration.
Using the Grafana user interface, select Configuration, select Data Sources, click "Add data source," and choose
Prometheus, and then specify the necessary parameters to connect to the Prometheus instance. In test environments, the only really
necessary parameter is the URL, but authentication is also desirable in most production deployments.

2. Import the existing dashboard. In the Grafana UI, click Dashboards, then Manage, then Import, and select one of the templates, e.g.
`kea-dhcp4.json`. Make sure to select the Prometheus data source added in the previous step. Once imported, the
dashboard can be tweaked as needed.

3. Once Grafana is configured, go to the Stork user interface, log in as super-admin, click Settings in the Configuration menu, and
then add the URLs to Grafana and Prometheus that point to the installations. Once this is done, Stork will be able to show links
for subnets leading to specific subnets.

Alternatively, a Prometheus data source can be added by editing `datasource.yaml` (typically stored in `/etc/grafana`,
but this may vary depending on the installation) and adding entries similar to this one:

.. code-block:: yaml

   datasources:
   - name: Stork-Prometheus instance
     type: prometheus
     access: proxy
     url: http://prometheus.example.org:9090
     isDefault: true
     editable: false

Also, the Grafana dashboard files can be copied to `/var/lib/grafana/dashboards/` (again, this may vary depending on the
installation).

Example dashboards with some live data can be seen in the `Stork screenshots gallery
<https://gitlab.isc.org/isc-projects/stork/-/wikis/Screenshots#grafana>`_ .
