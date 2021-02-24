.. _installation:

************
Installation
************

Stork can be installed from pre-built packages or from sources. The following sections describe both methods. Unless there's a
good reason to compile from sources, installing from native DEB or RPM packages is easier and faster.

.. _supported_systems:

Supported Systems
=================

Currently Stork is being tested on the following systems:

- Ubuntu 18.04 and 20.04
- Fedora 31 and 32
- CentOS 7
- MacOS 10.15*

Note that MacOS is not and will not be officially supported. Many developers in our team use Macs, so we're trying to keep Stork
buildable on this platform.

Stork server and agents are written in Go language. The server uses PostgreSQL database. In principle, the software could be run
on any POSIX system that has Go compiler and PostgreSQL. It is likely the software can be built on many other modern systems, but
for the time being our testing capabilities are modest. If your favourite OS is not on this list, please do try running Stork
and report your findings.

Installation Prerequisites
==========================

The ``Stork Agent`` does not require any specific dependencies to run. It can be run immediately after installation.

Stork uses the `status-get` command to communicate with Kea, and therefore will only work with a version of Kea that supports
`status-get`, which was introduced in Kea 1.7.3 and backported to 1.6.3.

Stork requires the premium ``Host Commands (host_cmds)`` hook library to be loaded by the Kea instance to retrieve host
reservations stored in an external database. Stork does work without the Host Commands hook library, but is not able to display
host reservations. Stork can retrieve host reservations stored locally in the Kea configuration without any additional hook
libraries.

Stork requires the open source ``Stat Commands (stat_cmds)`` hook library to be loaded by the Kea instance to retrieve lease
statistics. Stork does work without the Stat Commands hook library, but will not be able to show pool utilization and other
statistics.

For the ``Stork Server``, a PostgreSQL database (https://www.postgresql.org/) version 11 or later is required. It may work with
PostgreSQL 10, but this was not tested. The general installation procedure for PostgreSQL is OS-specific and is not included
here. However, please keep in mind that Stork uses pgcrypto extensions, which are often come in a separate package. For
example, you need postgresql-crypto package on Fedora and postgresql12-contrib on RHEL and CentOS.

These instructions prepare a database for use with the ``Stork
Server``, with the `stork` database user and `stork` password.  Next,
a database called `stork` is created and the `pgcrypto` extension is
enabled in the database.

First, connect to PostgreSQL using `psql` and the `postgres`
administration user. Depending on your system configuration, this may require
switching to `postgres` user, using `su postgres` command first.

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

   Make sure the actual password is stronger than 'stork' which is trivial to guess.
   Using default passwords is a security risk. Stork puts no restrictions on the
   characters used in the database passwords nor on their length. In particular,
   it accepts passwords containing spaces, quotes, double quotes and other
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

Debian/Ubuntu
~~~~~~~~~~~~~

The first step for both Debian and Ubuntu is:

.. code-block:: console

   $ curl -1sLf 'https://dl.cloudsmith.io/public/isc/stork/cfg/setup/bash.deb.sh' | sudo bash

Next, install the package with ``Stork Server``:

.. code-block:: console

   $ sudo apt install isc-stork-server


CentOS/RHEL/Fedora
~~~~~~~~~~~~~~~~~~

The first step for RPM-based distributions is:

.. code-block:: console

   $ curl -1sLf 'https://dl.cloudsmith.io/public/isc/stork/cfg/setup/bash.rpm.sh' | sudo bash

Next, install the package with ``Stork Server``:

.. code-block:: console

   $ sudo dnf install isc-stork-server

If ``dnf`` is not available, ``yum`` can be used in similar fashion.

Setup
~~~~~

These steps are the same for both Debian-based and RPM-based
distributions that use `SystemD`.

After installing ``Stork Server`` from the package, the basic settings
must be configured. They are stored in ``/etc/stork/server.env``.

These are the required settings to connect with the database:

* STORK_DATABASE_HOST - the address of a PostgreSQL database; default is `localhost`
* STORK_DATABASE_PORT - the port of a PostgreSQL database; default is `5432`
* STORK_DATABASE_NAME - the name of a database; default is `stork`
* STORK_DATABASE_USER_NAME - the username for connecting to the database; default is `stork`
* STORK_DATABASE_PASSWORD - the password for the username connecting to the database

With those settings in place, the ``Stork Server`` service can be
enabled and started:

.. code-block:: console

   $ sudo systemctl enable isc-stork-server
   $ sudo systemctl start isc-stork-server

To check the status:

.. code-block:: console

   $ sudo systemctl status isc-stork-server

By default, the ``Stork Server`` web service is exposed on port 8080,
so it can be visited in a web browser at http://localhost:8080.

It is possible to put ``Stork Server`` behind an HTTP reverse proxy
using `Nginx` or `Apache`. In the ``Stork Server`` package an example
configuration file is provided for `Nginx`, in
`/usr/share/stork/examples/nginx-stork.conf`.

Securing Connections Between Stork Server and Stork Agents
----------------------------------------------------------

Connections between the server and agents are always secured using
standard cryptography solutions, i.e., PKI and TLS.

The keys and certificates are automatically generated during the
server startup to:

* register new agents (prepare agents keys and certificates) and,
* establish safe, encrypted connections where both ends are authenticated.

During agent installation, a registration procedure is conveyed where
an agent generates its private key and CSR certificate, and then it is
signed by the server and returned to the agent. The agent is using
this signed certificate later for authentication and connection
encryption.

There are two ways of registering an agent:

#. using agent's token,
#. using server's token.

They are described in the following chapters.

Installing the Stork Agent
--------------------------

These steps are the same for both Debian-based and RPM-based
distributions that use `SystemD`.

After installing ``Stork Agent`` from the package, a user needs to
specify the necessary settings in ``/etc/stork/agent.env``.

General settings:

* STORK_AGENT_ADDRESS - the IP address of the network interface which ``Stork Agent``
  should use for listening for ``Stork Server`` incoming connections;
  default is `0.0.0.0` (i.e. listen on all interfaces)
* STORK_AGENT_PORT - the port that should be used for listening; default is `8080`
* STORK_AGENT_LISTEN_STORK_ONLY - enable Stork functionality only,
  i.e. disable Prometheus exporters; default is false
* STORK_AGENT_LISTEN_PROMETHEUS_ONLY - enable Prometheus exporters
  only, i.e. disable Stork functionality; default is false

Settings specific to Prometheus exporters:

* STORK_AGENT_PROMETHEUS_KEA_EXPORTER_ADDRESS - the IP or hostname to
  listen on for incoming Prometheus connection; default is `0.0.0.0`
* STORK_AGENT_PROMETHEUS_KEA_EXPORTER_PORT - the port to listen on for
  incoming Prometheus connection; default is `9547`
* STORK_AGENT_PROMETHEUS_KEA_EXPORTER_INTERVAL - specifies how often
  the agent collects stats from Kea, in seconds; default is `10`
* STORK_AGENT_PROMETHEUS_BIND9_EXPORTER_ADDRESS - the IP or hostname
  to listen on for incoming Prometheus connection; default is `0.0.0.0`
* STORK_AGENT_PROMETHEUS_BIND9_EXPORTER_PORT - the port to listen on
  for incoming Prometheus connection; default is `9119`
* STORK_AGENT_PROMETHEUS_BIND9_EXPORTER_INTERVAL - specifies how often
  the agent collects stats from BIND 9, in seconds; default is `10`

The next setting must be used only if an agent is automatically registered in
Stork server using agent token:

* STORK_AGENT_SERVER_URL - URL of Stork server

Registration using Agent Token
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

This method requires that after installing ``Stork Agent`` from
packages, a user will set two parameters:
``STORK_AGENT_SERVER_URL`` and ``STORK_AGENT_ADDRESS``.
``STORK_AGENT_SERVER_URL`` should point to URL of ``Stork Server``,
e.g.: ``http://stork-server.example.com:8080``. The other one,
``STORK_AGENT_ADDRESS``, should indicate an address and a port of an
agent, e.g.: ``stork-agent.example.com:8080``.

At that moment, a user should start the agent service:

.. code-block:: console

   $ sudo systemctl enable isc-stork-agent
   $ sudo systemctl start isc-stork-agent

To check the status:

.. code-block:: console

   $ sudo systemctl status isc-stork-agent

When the agent starts, it first generates its private key, a CSR
certificate (using the address specified with the
``STORK_AGENT_ADDRESS``), and an agent token. Then, it tries to
connect to the Stork Server using its URL specified with the
``STORK_SERVER_URL``. Finally, it starts the registration
procedure. It sends the CSR and the token to the server.  The server
stores the token, signs the CSR, and returns the CSR to the agent. The
agent will use this certificate to authenticate itself to the
server. The registration procedure finishes on the agent's side.


Still, the agent is not authorized and is not fully functional from
the server perspective. Now a user needs to visit a machines page in
``Stork Server`` web UI (menu ``Services -> Machines``). After
switching to unauthorized machines in UI, the just started agent
should be visible. The user needs to invoke an authorize function to
make the agent fully working and visible in ``Stork
Server``. Switching back to authorized machines should show the full
state of the machine.

The agent is not yet authorized from the server's perspective. To
authorize the agent, a user must visit a machines page in Stork
Server's web UI (menu ``Services -> Machines``) and switch to the list
of unauthorized agents. The newly registered agent should be on that
list. Compare if agent token stored in
``/var/lib/stork-agent/tokens/agent-token.txt`` is the same as the one
displayed in web UI. If they match then click on the ``Action`` button
and select ``Authorize`` menu item. Switching back to authorized
machines should show the full state of the machine.


Registration using Server Token
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

At first, the user needs to visit the machines page on the ``Stork
Server`` web UI (menu ``Services -> Machines``). Clicking ``How to
Install Agent to New Machine`` button will reveal a dialog box. It
presents several things:

#. a list of commands for installing an agent package,
#. a ``server token``,
#. a button for regenerating the ``server token``.

The presented commands should be executed on the agent's
machine. Invoked ``stork-install-agent.sh`` will prompt for several
things:

1. for ``server token``:

.. code-block:: text

   >>>> Please, provide server access token (optional):

If server token is skipped with Enter the registration using agent
token will be performed, otherwise it will still be server token
based one.

The ``server token`` should be copied from web UI and pasted to the
terminal.

2. The next question will be for `agent address`:

.. code-block:: text

   >>>> Please, provide address (IP or name/FQDN) of current host with Stork Agent (it will be used to connect from Stork Server) [...]:

3. The following step is to specify the `agent port`:

.. code-block:: text

   >>>> Please, provide port that Stork Agent will use to listen on [8080]:

And that's it. The script execution should end with a message:

.. code-block:: text

   machine ping over TLS: OK
   registration completed successfully

Now the agent should be authorized in the server and should be visible on
the machines page in the web UI.

Registration Methods Summary
~~~~~~~~~~~~~~~~~~~~~~~~~~~~

The server token way requires manual installation of the agent
and providing server token. In effect the agent is immediatelly
registered in the Stork server. If server token leaks then
it should be regenerated on the machines page in ``How to
Install Agent to New Machine`` dialog box. In does not break
earlier registered agents. It only impacts new registrations that
should use new server token.

The agent token way requires preconfiguring only server URL
on agent side, no server token is needed. This allows e.g. to prepare
a container image with an agent offline and deploy it later.
Then it required manual agent authorization in Stork web UI.
The identity of agent should be confirmed by comparing agent token
stored in ``/var/lib/stork-agent/tokens/agent-token.txt`` with
the agent token presented in web UI.


Agent Setup Summary
~~~~~~~~~~~~~~~~~~~

After successful agent setup, the agent periodically tries to detect installed
Kea DHCP or BIND 9 services on the system. If it finds them, they are
reported to the ``Stork Server`` when it connects to the agent.

Further configuration and usage of the ``Stork Server`` and the
``Stork Agent`` are described in the :ref:`usage` chapter.


Upgrading
---------

An upgrade procedure looks the same as installation procedure.

At first, install new packages on the server. Installation scripts in
Deb/RPM package will perform the required database and other migrations.

The next step is agent upgrade. The steps are the same: download the
agent installation script to the agent machine and invoke it. This time
registration will be skipped as the machine is already registered.

.. _installation_sources:

Installing from Sources
=======================

Compilation Prerequisites
-------------------------

Usually it's more convenient to install Stork using native packages. See :ref:`supported_systems` and :ref:`install-pkgs` for
details regarding supported systems. However, you can build the sources on your own.

The dependencies needed to be installed to build ``Stork`` sources are:

 - Rake
 - Java Runtime Environment (only if building natively, not using Docker)
 - Docker (only if running in containers, this is needed to build the demo)

Other dependencies are installed automatically in a local directory by Rake tasks. This does not
require root priviledges. If you intend to run the demo environment, you need Docker and don't need
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
migration is triggered automatically upon server startup.  This is
only useful if for some reason it is desirable to set up the database
but not yet run the server. In most cases this step can be skipped.

.. code-block:: console

    $ rake build_migrations
    $ backend/cmd/stork-db-migrate/stork-db-migrate init
    $ backend/cmd/stork-db-migrate/stork-db-migrate up

The up and down command has an optional `-t` parameter that specifies desired
schema version. This is only useful when debugging database migrations.

.. code-block:: console

    $ # migrate up version 25
    $ backend/cmd/stork-db-migrate/stork-db-migrate up -t 25
    $ # migrate down back to version 17
    $ backend/cmd/stork-db-migrate/stork-db-migrate down -t 17

Note the server requires the latest database version to run, will always
run the migration on its own and will refuse to start if migration fails
for whatever reason. The migration tool is mostly useful for debugging
problems with migration or migrating the database without actually running
the service. For complete reference, see manual page here:
:ref:`man-stork-db-migrate`.

To debug migrations, another useful feature is SQL tracing using the `--db-trace-queries` parameter.
It takes either "all" (trace all SQL operations, including migrations and run-time) or "run" (just
run-time operations, skip migrations). If specified without paraemter, "all" is assumed. With it enabled,
`stork-db-migrate` will print out all its SQL queries on stderr. For example, you can use these commands
to generate an SQL script that will update your schema. Note that for some migrations, the steps are
dependent on the contents of your database, so this will not be an universal Stork schema. This parameter
is also supported by the Stork server.

.. code-block:: console

   $ backend/cmd/stork-db-migrate/stork-db-migrate down -t 0
   $ backend/cmd/stork-db-migrate/stork-db-migrate up --db-trace-queries 2> stork-schema.txt


Integration with Prometheus and Grafana
=======================================

Stork can optionally be integrated with `Prometheus <https://prometheus.io/>`_, an open source monitoring and alerting toolkit
and `Grafana <https://grafana.com/>`_, an easy-to-view analytics platform for querying, visualization and altering. Grafana
requires external data storage. Prometheus is currently the only environment supported by both Stork and Grafana. It is possible
to use Prometheus only without Grafana, but using Grafana requires Prometheus.

Prometheus Integration
----------------------

Stork agent by default makes the
BIND 9 and Kea statistics available in a format understandable by Prometheus (works as a Prometheus exporter, in Prometheus
nomenclature). If Prometheus server is available, it can be configured to monitor Stork Agents. To enable Stork Agent
monitoring, you need to edit ``prometheus.yml`` (typically stored in /etc/prometheus/, but this may vary depending on your
installation) and add the following entries there:

.. code-block:: yaml

  # statistics from Kea
  - job_name: 'kea'
    static_configs:
      - targets: ['agent-kea.example.org:9547', 'agent-kea6.example.org:9547', ... ]

  # statistics from bind9
  - job_name: 'bind9'
    static_configs:
      - targets: ['agent-bind9.example.org:9119', 'another-bind9.example.org:9119', ... ]

By default, Stork agent exports BIND 9 data on TCP port 9119 and Kea data on TCP port 9547. This can be configured using command
line parameters (or the Prometheus export can be disabled altogether). For details, see the stork-agent manual page.

After restarting, the Prometheus web interface can be used to inspect whether statistics are exported properly. BIND 9
statistics use ``bind_`` prefix (e.g. bind_incoming_queries_tcp), while Kea statistics use ``kea_`` prefix (e.g.
kea_dhcp4_addresses_assigned_total).

Grafana Integration
-------------------

Stork provides several Grafana templates that can easily be imported. Those are available in the ``grafana/`` directory of the
Stork source codes. Currently the available templates are `bind9-resolver.json` and `kea-dhcp4.json`. More are expected in the
future. Grafana integration requires three steps.

1. Prometheus has to be added as a data source. This can be done in several ways, including UI interface and editing Grafana
configuration files. For details, see Grafana documentation about Prometheus integration; here we simply indicate the easiest
method. Using the Grafana UI interface, select Configuration, select Data Sources, click "Add data source", and choose
Prometheus, then specify necessary parameters to connect to your Prometheus instance. In test environments, the only really
necessary parameter is URL, but most production deployments also want authentication.

2. Import existing dashboard. In the Grafana UI click Dashboards, then Manage, then Import and select one of the templates, e.g.
`kea-dhcp4.json`. Make sure to select your Prometheus data source that you added in the previous step. Once imported, the
dashboard can be tweaked as needed.

3. Once Grafana is configured, go to Stork UI interface, log in as super-admin, click Settings in the Configuration menu and
then fill URLs to Grafana and Prometheus that point to your installations. Once this is done, Stork will be able to show links
for subnets leading to specific subnets. More integrations like this are expected in the future.

Alternatively, a Prometheus data source can be added by editing `datasource.yaml` (typically stored in `/etc/grafana`,
but this may vary depending on your installation) and adding entries similar to this one:

.. code-block:: yaml

   datasources:
   - name: Stork-Prometheus instance
     type: prometheus
     access: proxy
     url: http://prometheus.example.org:9090
     isDefault: true
     editable: false

Also, the Grafana dashboard files can be copied to `/var/lib/grafana/dashboards/` (again, this may vary depending on your
installation).

Example dashboards with some live data can be seen in the `Stork screenshots gallery
<https://gitlab.isc.org/isc-projects/stork/-/wikis/Screenshots#grafana>`_ .
