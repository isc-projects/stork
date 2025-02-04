.. _installation:

************
Installation
************

Stork can be installed from pre-built packages or from sources; the following sections describe both methods. Unless there is a
good reason to compile from sources, installing from native deb or RPM packages is easier and faster.

.. _compatible_systems:

Compatible Systems
==================

``stork-server`` and ``stork-agent`` are written in the Go language; the server uses a PostgreSQL database. In
principle, the software can be run on any POSIX system that has a Go compiler and PostgreSQL. It is likely the software
can also be built on other modern systems, but ISC's testing capabilities are modest. We encourage users to try running
Stork on other OSes not on this list and report their findings to ISC.

Our main effort goes into ensuring compatibility with live Ubuntu and RHEL long-time-support (LTS) versions.

The table below describes the checks performed on each particular operating
system.

.. csv-table:: Operating-system compatibility
   :file: compatible-systems.csv
   :header-rows: 1

Legend:

- ``X`` - the check is regularly performed on our CI system
- ``D`` - the operating system is used to develop Stork, so some checks are performed as a side effect
- ``U`` - the check is not covered by our CI system, but some users have reported successful passes

\* MacOS is not and will not be officially supported. However, many developers on ISC's team use Macs, so the intention
is to keep Stork buildable on this platform.

\*\* BSD-like systems are not officially supported, but some users have reported successful installations on these
systems.


Installation Prerequisites
==========================

The Stork agent does not require any specific dependencies to run. It can be run immediately after installation.

Stork uses the ``status-get`` command to communicate with Kea.

Stork requires the premium Host Commands (``host_cmds``) hook library to be loaded by the Kea instances to retrieve host
reservations stored in an external database. Stork works without the Host Commands hook library, but is not able to display
host reservations. Stork can retrieve host reservations stored locally in the Kea configuration without any additional hook
libraries; however, managing (adding, updating, deleting) host reservations with Stork requires the ``host_cmds`` hooks to be loaded on all
Kea instances where these host reservations belong.

Stork requires the premium Subnet Commands (``subnet_cmds``) hook library to be loaded by the Kea instances
to manage the subnets and shared networks. Stork can fetch and present subnets and shared networks without this
hook library; however, adding, updating, or deleting a subnet or shared network requires the ``subnet_cmds``
hook library to be loaded on all Kea instances where this subnet or shared network belongs.

Stork requires the open source Statistics Commands (``stat_cmds``) hook library to be loaded by the Kea instance to retrieve lease
statistics. Stork works without the Stat Commands hook library, but is not able to show pool utilization and other
statistics.

Stork uses the Go implementation to handle TLS connections, certificates, and keys. The secrets are stored in the PostgreSQL
database, in the ``secret`` table.

For the Stork server, a PostgreSQL database (https://www.postgresql.org/) version 10
or later is required. Stork attempts to run with older versions, but may not work
correctly. The general installation procedure for PostgreSQL is OS-specific and is not included
here; however, please note that Stork uses pgcrypto extensions, which often come in a separate package. For
example, a postgresql-crypto package is required on Fedora and postgresql12-contrib is needed on RHEL and CentOS.

.. _stork-tool:

The Stork Tool
==============

The Stork tool is a program installed with the Stork server, providing commands
to set up the server's database and manage TLS certificates. Using this tool is optional
because the server runs the database migrations and creates suitable certificates at
startup on its own. However,  the tool provides useful commands for inspecting
the current database schema version and downgrading to a previous version.
In addition, the :ref:`setup-server-database` section describes how the Stork tool can be
conveniently used to create a new database and its credentials without the need to run
SQL commands directly using the ``psql`` program.

The :ref:`inspecting-keys-and-certificates` section describes how to use the Stork tool for TLS
certificate management. The :ref:`configuring-deployment-specific-views` section describes how to set up a custom
welcome message on the login page.

Other sections describe different methods for installing the Stork server from packages;
see: :ref:`install-server-deb` and :ref:`install-server-rpm`. The ``stork-tool`` program
is installed from packages along with the server, but the tool can also be
built from sources:

.. code-block:: console

    $ rake build:tool

Please refer to the manual page for usage details: :ref:`man-stork-tool`.

.. _setup-server-database:

Preparing the Stork Server Database
===================================

Before running ``stork-server``, a PostgreSQL database and the user with suitable privileges
must be created. Using ``stork-tool`` is the most convenient way to set up the database.

The following command creates a new database ``stork`` and a user ``stork`` with all privileges
in this database. It also installs the ``pgcrypto`` extension required by the Stork server:

.. code-block:: console

    $ stork-tool db-create --db-name stork --db-user stork
    created database and user for the server with the following credentials  database_name=stork password=L82B+kJEOyhDoMnZf9qPAGyKjH5Qo/Xb user=stork

By default, ``stork-tool`` connects to the database as user ``postgres``, a default admin role
in many PostgreSQL installations. If an installation uses a different administrator name, it can
be specified with the ``--db-maintenance-user`` option. For example:

.. code-block:: console

    $ stork-tool db-create --db-maintenance-user thomson --db-name stork --db-user stork
    created database and user for the server with the following credentials  database_name=stork password=L82B+kJEOyhDoMnZf9qPAGyKjH5Qo/Xb user=stork

Similarly, a ``postgres`` database often exists in a PostgreSQL installation.
However, a different maintenance database can be selected with the ``--db-maintenance-name``
option.

``stork-tool`` generates a random password to the created database. This password needs
to be copied into the server environment file or used in the ``stork-server`` command line,
to configure the server to use this password while connecting to the database. Use the
``--db-password`` option with the ``db-create`` command to create a user with the specified
password.


It is also possible to create the database manually (i.e., using the ``psql`` tool).

First, connect to PostgreSQL using ``psql`` and the ``postgres``
administration user. Depending on the system's configuration, it may require
switching to the user ``postgres`` first, using the ``su postgres`` command.

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
    You are now connected to database "stork" as user "postgres".
    postgres=# GRANT ALL PRIVILEGES ON SCHEMA public TO stork;
    GRANT
    stork=# create extension pgcrypto;
    CREATE EXTENSION

.. note::

   Make sure the actual password is stronger than "stork", which is trivial to guess.
   Using default passwords is a security risk. Stork puts no restrictions on the
   characters used in the database passwords, nor on their length. In particular,
   it accepts passwords containing spaces, quotes, double quotes, and other
   special characters. Please also consider using the ``stork-tool`` to generate
   a random password.

To generate a random password run:

.. code-block:: console

    $ stork-tool db-password-gen
    generated new database password               password=1qWVzmLKy/j40/FVsvjM2ylcFdaFfNxh


The newly created database is not ready for use until necessary database migrations
are executed. The migrations create tables, indexes, triggers, and functions required
by the Stork server. As mentioned above, the server can automatically run the
migrations at startup, bringing the database schema up to the latest version. However,
to run the migrations before starting the server, the
``stork-tool`` command can be used:

.. code-block:: console

    $ stork-tool db-init
    $ stork-tool db-up

The ``up`` and ``down`` commands have an optional ``-t`` parameter that specifies the
desired schema version. It is useful when debugging database migrations or downgrading to
an earlier Stork version.

.. code-block:: console

    $ # migrate up to version 25
    $ stork-tool db-up -t 25
    $ # migrate down back to version 17
    $ stork-tool db-down -t 17

The server requires the latest database version to run, always
runs the migration on its own, and refuses to start if the migration fails
for any reason. The migration tool is mostly useful for debugging
problems with migration, or for migrating the database without actually running
the service. For the complete manual page, please see
:ref:`man-stork-tool`.

To debug migrations, another useful feature is SQL tracing using the ``--db-trace-queries`` parameter.
The options are either ``all`` (trace all SQL operations, including migrations and runtime) or ``run`` (only
trace runtime operations and skip migrations). If specified without any parameters, ``all`` is assumed. With it enabled,
``stork-tool`` prints out all its SQL queries on stderr. For example, these commands can be used
to generate an SQL script that updates the schema. Note that for some migrations, the steps are
dependent on the contents of the database, so this is not a universal Stork schema. This parameter
is also supported by the Stork server.

.. code-block:: console

   $ stork-tool db-down -t 0
   $ stork-tool db-up --db-trace-queries 2> stork-schema.txt

Tuning Database Timeouts
------------------------

In rare cases, reading or writing to the database may hang. This can be caused by a temporary network issue, or by
misconfiguration of the proxy server switching the connection between different database instances. Setting appropriate
timeout values can mitigate such issues.

PostgreSQL exposes two distinct connection options to configure the read and write timeouts. Stork's corresponding
configuration parameters specify the timeouts: ``--db-read-timeout`` (the
``STORK_DATABASE_READ_TIMEOUT`` environment variable) and ``--db-write-timeout`` (the ``STORK_DATABASE_WRITE_TIMEOUT``).
A non-zero timeout value must be specified with the unit, e.g., ``500ms``` for 500 milliseconds, ``1s`` for one second,
or ``5m`` for five minutes.

The default value is ``0``, which disables the timeout. In this case, Stork waits indefinitely for the completion of the
read and write database operations.

.. note::

   Some operations may take a long time to complete (e.g., pulling the hosts), especially when the monitored Kea
   configurations are large. The timeouts should be adjusted accordingly.

.. warning::

   We do not recommend specifying short timeouts when database connection problems are rare and/or hangs related
   to infinite reads or writes have not been observed. Specifying too-short timeouts may lead to premature database read
   or write errors for long-lasting operations.

   Note that the flags ``--rest-read-timeout`` and ``--rest-write-timeout`` can control how long the
   REST API operation lasts. They are suitable for securing the Stork API against denial-of-service attacks
   that involve sending massive, long-processing requests to the web service to exhaust its resources.

.. _install-pkgs:

Installing From Packages
========================

Stork packages are stored in repositories located on the Cloudsmith
service: https://cloudsmith.io/~isc/repos/stork/packages/. DEB (Debian/Ubuntu
family), RPM (RedHat family), and APK (Alpine) packages may be found there.

Detailed instructions for setting up the operating system to use this
repository are available under the ``Set Me Up`` button on the
Cloudsmith repository page.

A few command-line tools are required for setting up the repository:

- ``bash`` to execute the ``setup.*.sh`` scripts that use bash-specific features
  like ``==`` inside the ``test`` expression, ``local`` variables, or the ``function``
  keyword in function declarations.
- ``curl`` to fetch the ``setup.*.sh`` script itself, but also for actions
  carried out by the script like fetching GPG keys, checking if URLs are
  available, or fetching other scripts.
- ``cut`` for the output manipulation done by the script
- ``gpg`` for importing GPG keys in the script
- ``sed`` for various textual substitutions done by the script
- ``sudo`` for elevating privileges required by the package manager
- ``apt-get`` for a Debian-based distribution
- ``rpm`` for an RPM-based distribution
- one of ``dnf``, ``microdnf``, ``yum``, or ``zypper`` for an RPM-based
  distribution
- ``apk`` for an Alpine-based distribution

Other command-line tools may be required based on how the script evolves or
based on what OS the script is running on:

- ``grep`` and ``head`` for filtering output
- ``sort`` for sorting output
- ``fmt`` and ``xargs`` for better reporting of errors
- ``uname`` to detect the OS the script is running on
- ``python`` and ``pip`` for a redundant way of automatically detecting the OS in
  case the main OS detection mechanism does not work

It is possible to install both ``stork-agent`` and ``stork-server`` on
the same machine. It is useful in small deployments with a single
monitored machine, to avoid setting up a dedicated system for the Stork
server. In those cases, however, an operator must consider the potential
impact of the Stork server on other services running on the same
machine.

Installing the Stork Server
---------------------------

.. _install-server-deb:

Installing on Debian/Ubuntu
~~~~~~~~~~~~~~~~~~~~~~~~~~~

The first step for both Debian and Ubuntu is:

.. code-block:: console

   $ curl -1sLf 'https://dl.cloudsmith.io/public/isc/stork/cfg/setup/bash.deb.sh' | sudo bash

Next, install the Stork server package:

.. code-block:: console

   $ sudo apt install isc-stork-server

.. _install-server-rpm:

Installing on CentOS/RHEL/Fedora
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

The first step for RPM-based distributions is:

.. code-block:: console

   $ curl -1sLf 'https://dl.cloudsmith.io/public/isc/stork/cfg/setup/bash.rpm.sh' | sudo bash

Next, install the Stork server package:

.. code-block:: console

   $ sudo dnf install isc-stork-server

If ``dnf`` is not available, ``yum`` can be used instead:

.. code-block:: console

   $ sudo yum install isc-stork-server

Installing on Alpine
~~~~~~~~~~~~~~~~~~~~

The first step for Alpine is installing ``bash`` and ``curl``:

.. code-block:: console

   $ apk add bash curl

Next, set up the Cloudsmith repository:

.. code-block:: console

   $ curl -1sLf 'https://dl.cloudsmith.io/public/isc/stork/cfg/setup/bash.alpine.sh' | bash

Then, install the Stork server package:

.. code-block:: console

   $ apk add isc-stork-server

.. _server-setup:

Setup
~~~~~

The following steps are common for Debian-based and RPM-based distributions
using ``systemd``.

Configure the Stork server settings in ``/etc/stork/server.env``.

.. note::

   The environment file **IS NOT** read by default if the Stork server
   is run manually (without using ``systemd``). To load the environment variables from
   this file, call the ``. /etc/stork/server.env`` command before
   executing the binary (in the same shell instance), or run Stork with
   the ``--use-env-file`` switch.

The following settings are required for the database connection (they have a
common ``STORK_DATABASE_`` prefix):

* ``STORK_DATABASE_HOST`` - the address of a PostgreSQL database; the default is ``localhost``
* ``STORK_DATABASE_PORT`` - the port of a PostgreSQL database; the default is ``5432``
* ``STORK_DATABASE_NAME`` - the name of a database; the default is ``stork``
* ``STORK_DATABASE_USER_NAME`` - the username for connecting to the database; the default is ``stork``
* ``STORK_DATABASE_PASSWORD`` - the password for the username connecting to the database

.. note::

   All of the database connection settings have default values, but we strongly
   recommend protecting the database with a non-default and hard-to-guess password
   in a production environment. The ``STORK_DATABASE_PASSWORD`` setting must be
   adjusted accordingly.

The remaining settings pertain to the server's RESTful API configuration (the ``STORK_REST_`` prefix):

* ``STORK_REST_HOST`` - the IP address on which the server listens
* ``STORK_REST_PORT`` - the port number on which the server listens; the default is ``8080``
* ``STORK_REST_TLS_CERTIFICATE`` - a file with a certificate to use for secure connections
* ``STORK_REST_TLS_PRIVATE_KEY`` - a file with a private key to use for secure connections
* ``STORK_REST_TLS_CA_CERTIFICATE`` - a certificate authority file used for mutual TLS authentication

   Providing the CA certificate path enables the TLS client certificate
   verification. Any HTTP request to the server assigned with a missing,
   invalid, or untrusted TLS certificate is rejected.

* ``STORK_REST_STATIC_FILES_DIR`` - a directory with static files served in the user interface

   The ``STORK_REST_STATIC_FILES_DIR`` should be set to ``/usr/share/stork/www``
   for the Stork server installed from binary packages; this is the default location
   for static content.

* ``STORK_REST_BASE_URL`` - the base URL of the UI

   Specify this flag if the UI is served from a subdirectory (not the root URL).
   It must start and end with a slash. Example: https://www.example.com/admin/stork/
   would need to have "/admin/stork/" as the ``rest-base-url``.

.. note::

   The Stork agent must trust the REST TLS certificate presented by the Stork server.
   Otherwise, the registration process fails due to invalid Stork server
   certificate verification. We strongly recommend using proper, trusted
   certificates for security reasons. To use a self-signed
   certificate (e.g., for deployment in the Docker environment), it is possible to
   add its CA certificate to the system certificates on the Stork agent machine.
   See this `Stack Overflow conversation <https://stackoverflow.com/a/42292623>`_ and
   `discussion in Stork GitLab issue #859 <https://gitlab.isc.org/isc-projects/stork/-/issues/859>`_.

* ``STORK_REST_VERSIONS_URL`` - the URL of the file with current Kea, Stork and BIND 9 software versions metadata; the default is ``https://www.isc.org/versions.json``

   Stork can automatically check for software updates available for Kea, BIND 9 and Stork itself.
   To be able to do that, Stork server downloads a JSON file with the latest software releases metadata.
   The file's URL by default is set to `https://www.isc.org/versions.json <https://www.isc.org/versions.json>`_.
   There is no need to modify this setting until the software releases metadata file is published by ISC under a different URL.
   See also chapters :ref:`usage-software-versions-page` and :ref:`Automatic software updates checking <usage-online-version-check-setting>`.

The remaining settings pertain to the server's Prometheus ``/metrics`` endpoint configuration (the ``STORK_SERVER_`` prefix is for general purposes):

* ``STORK_SERVER_ENABLE_METRICS`` - enable the Prometheus metrics collector and ``/metrics`` HTTP endpoint

.. warning::

   The Prometheus ``/metrics`` endpoint does not require authentication. Therefore, securing this endpoint
   from external access is highly recommended to prevent unauthorized parties from gathering the server's
   metrics. One way to restrict endpoint access is by using an appropriate HTTP proxy configuration
   to allow only local access or access from the Prometheus host. Please consult the NGINX example
   configuration file shipped with Stork.

With the settings in place, the Stork server service can now be enabled and
started:

.. code-block:: console

   $ sudo systemctl enable isc-stork-server
   $ sudo systemctl start isc-stork-server

To check the status:

.. code-block:: console

   $ sudo systemctl status isc-stork-server

.. note::

   By default, the Stork server web service is exposed on port 8080 and
   can be tested using a web browser at http://localhost:8080. To use a different IP address or port,
   set the ``STORK_REST_HOST`` and ``STORK_REST_PORT`` variables in the ``/etc/stork/stork.env``
   file.

The Stork server can be configured to run behind an HTTP reverse proxy
using ``nginx`` or ``Apache``. The Stork server package contains an example
configuration file for ``nginx``, in ``/usr/share/stork/examples/nginx-stork.conf``.

The logging details, including colorization, are configured in the same way as the
:ref:`Stork agent logging settings <logging-settings>`.

Stork can read and combine the configuration parameters from multiple sources simultaneously.
The command-line flags have precedence over the environment variables read from the file,
when the ``--use-env-file`` flag is specified. The environment variables read from the file
take precedence over the environment variables set in the current shell.


.. _web-ui-reverse-proxy:

Stork UI Behind a Reverse Proxy
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

A reverse proxy is a server solution responsible for preliminary processing
of incoming requests from the Internet and redirecting them to specific web
services running in the internal network. Reverse proxies may help increase
performance (e.g., by caching responses), security (e.g., by enveloping the
responses in TLS, logging the requests), and reliability (e.g., by allowing
switching of web service instances).

Stork is distributed with a basic configuration for NGINX. It is available,
after installation from a package, in the ``dist/server/usr/share/stork/examples/nginx-stork.conf`` file.
The same file is located in the git repository: ``etc/nginx-stork.conf``.

The Stork server can be configured to expose the web application from a URL subdirectory.
It may be needed when there is no dedicated domain for the Stork server, and the
web application is served from the subdirectory of an existing domain
(e.g., ``http://example.com/stork``).

If the backend executable (``stork-server``) and UI files (``/usr/share/stork/www``
by default) are on the same machine, the backend is responsible for sharing the UI
static files. The necessary subdirectory can be configured using the ``--rest-base-url``
CLI flag or the ``STORK_REST_BASE_URL`` environment variable.
The value must be surrounded by slashes (e.g.: ``/stork/``). The ``--rest-base-url``
CLI flag affects both the backend and UI. It changes the value of the ``<base>``
HTML tag in the ``index.html`` file (which modifies all links and URLs used by
the UI) and turns on the simple remapping of the requested URL (the backend trims
the base path from processed URLs). The reverse proxy does not require any
special configuration.

If the backend and the UI files are located on different machines, the value of
the ``<base>`` HTML tag must be manually modified in the ``index.html``
file. The ``href`` attribute must be set to a necessary subdirectory.
The value must be surrounded by slashes (e.g.: ``/stork/``). Configure the
HTTP proxy server to rewrite the requested URL and remove the base URL before
passing the requests to the Stork server. Below is an example of configuring the
``<VirtualHost>`` section for Apache; see the ``etc/httpd-stork.conf`` file for
the full configuration.

.. code-block::

   <VirtualHost *:81>
      <LocationMatch "^/stork">
         RewriteEngine On
         RewriteCond %{REQUEST_URI} ^/stork/(.*)
         RewriteRule ^ /%1 [L]
      </LocationMatch>

      # Other location rules here...

   </VirtualHost>

.. _securing-the-database-connection:

Securing the Database Connection
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

The PostgreSQL server can be configured to encrypt communications between the clients and
the server. Detailed information on how to enable encryption on the database
server, and how to create the suitable certificate and key files, is available
in the `PostgreSQL documentation
<https://www.postgresql.org/docs/14/ssl-tcp.html>`_.

The Stork server supports secure communications with the database. The following
configuration settings in the ``server.env`` file enable and configure communication
encryption with the database server. They correspond with the SSL settings provided
by ``libpq``, the native PostgreSQL client library written in C:

* ``STORK_DATABASE_SSLMODE`` - the SSL mode for connecting to the database (i.e., ``disable``,
  ``require``, ``verify-ca``, or ``verify-full``); the default is ``disable``
* ``STORK_DATABASE_SSLCERT`` - the location of the SSL certificate used by the server
  to connect to the database
* ``STORK_DATABASE_SSLKEY`` - the location of the SSL key used by the server to connect
  to the database
* ``STORK_DATABASE_SSLROOTCERT`` - the location of the root certificate file used to
  verify the database server's certificate

The default SSL mode setting, ``disable``, configures the server to use unencrypted
communication with the database. Other settings have the following meanings:

* ``require`` - use secure communication but do not verify the server's identity
  unless the root certificate location is specified and that certificate exists.
  If the root certificate exists, the behavior is the same as in the case of ``verify-ca``.
* ``verify-ca`` - use secure communication and verify the server's identity by
  checking it against the root certificate stored on the Stork server machine.
* ``verify-full`` - use secure communication and verify the server's identity against
  the root certificate. In addition, check that the server hostname matches the
  name stored in the certificate.

Specifying the SSL certificate and key location is optional. If they are not
specified, the Stork server uses the ones from the current user's home
directory: ``~/.postgresql/postgresql.crt`` and ``~/.postgresql/postgresql.key``.
If they are not present, Stork tries to find suitable keys in common system
locations.

Please consult the `libpq documentation <https://www.postgresql.org/docs/14/libpq-ssl.html>`_
for similar ``libpq`` configuration details.

Installing the Stork Agent
--------------------------

There are two ways to install the packaged Stork agent on a monitored machine.
The first method is to use the Cloudsmith repository, as with the
Stork server installation. The second method
is to use an installation
script provided by the Stork server, which downloads the agent packages
embedded in the server package. The preferred installation method depends on
the selected agent registration type. Supported registration methods are
described in :ref:`secure-server-agent`.

.. _agent-configuration-settings:

Agent Configuration Settings
~~~~~~~~~~~~~~~~~~~~~~~~~~~~

The following are the Stork agent configuration settings available in the
``/etc/stork/agent.env`` file after installing the package. All these settings use
the ``STORK_AGENT_`` prefix to indicate that they configure the Stork agent.
Configuring Stork using the environment variables is recommended for deployments
using ``systemd``.

.. note::

   The environment file **IS NOT** read by default if the Stork agent is run
   manually (without using ``systemd``). To load the environment variables from
   this file, call the ``. /etc/stork/agent.env`` command before
   executing the binary (in the same shell instance) or run Stork with
   the ``--use-env-file`` switch.

The general settings:

* ``STORK_AGENT_HOST`` - the IP address of the network interface or DNS name which ``stork-agent``
  should use to receive connections from the server; the default is ``0.0.0.0``
  (i.e. listen on all interfaces)
* ``STORK_AGENT_PORT`` - the port number the agent should use to receive
  connections from the server; the default is ``8080``
* ``STORK_AGENT_LISTEN_STORK_ONLY`` - this enables Stork functionality only,
  i.e. disables Prometheus exporters; the default is ``false``
* ``STORK_AGENT_LISTEN_PROMETHEUS_ONLY`` - this enables the Prometheus exporters
  only, i.e. disables Stork functionality; the default is ``false``
* ``STORK_AGENT_SKIP_TLS_CERT_VERIFICATION`` - this skips TLS certificate verification when ``stork-agent``
  connects to Kea over TLS and Kea uses self-signed certificates; the default is ``false``

The following settings are specific to the Prometheus exporters:

* ``STORK_AGENT_PROMETHEUS_KEA_EXPORTER_ADDRESS`` - the IP address or hostname the
  agent should use to receive connections from Prometheus fetching Kea
  statistics; the default is ``0.0.0.0``
* ``STORK_AGENT_PROMETHEUS_KEA_EXPORTER_PORT`` - the port the agent should use to
  receive connections from Prometheus when fetching Kea statistics; the default is
  ``9547``
* ``STORK_AGENT_PROMETHEUS_KEA_EXPORTER_INTERVAL`` - this specifies how often
  the agent collects stats from Kea, in seconds; the default is ``10``
* ``STORK_AGENT_PROMETHEUS_KEA_EXPORTER_PER_SUBNET_STATS`` - this enables or disables
  the collection of per-subnet stats from Kea; the default is ``true`` (collecting enabled).
  This option can be used to limit the data passed to Prometheus/Grafana in large networks.
* ``STORK_AGENT_PROMETHEUS_BIND9_EXPORTER_ADDRESS`` - the IP address or hostname the
  agent should use to receive the connections from Prometheus fetching BIND 9
  statistics; the default is ``0.0.0.0``
* ``STORK_AGENT_PROMETHEUS_BIND9_EXPORTER_PORT`` - the port the agent should use to
  receive connections from Prometheus fetching BIND 9 statistics; the default is
  ``9119``
* ``STORK_AGENT_PROMETHEUS_BIND9_EXPORTER_INTERVAL`` - this specifies how often
  the agent collects stats from BIND 9, in seconds; the default is ``10``

The last setting is used only when Stork agents register in the Stork server
using an agent token:

* ``STORK_AGENT_SERVER_URL`` - the ``stork-server`` URL used by the agent to send REST
  commands to the server during agent registration

.. warning::

   ``stork-server`` does not currently support communication with ``stork-agent``
   via an IPv6 link-local address with zone ID (e.g., ``fe80::%eth0``). This means
   that the ``STORK_AGENT_HOST`` variable must be set to a DNS name, an IPv4
   address, or a non-link-local IPv6 address.

Stork can read and combine the configuration parameters from multiple sources simultaneously.
The command-line flags have precedence over the environment variables read from the file,
when the `--use-env-file` flag is specified. The environment variables read from the file
take precedence over the environment variables set in the current shell.

.. _logging-settings:

Logging Settings
~~~~~~~~~~~~~~~~

Unless otherwise specified using ``STORK_LOG_LEVEL``, the default value of ``INFO``
log level is used. Supported log levels are: ``DEBUG``, ``INFO``, ``WARN``, and ``ERROR``.

To control the logging colorization, Stork supports the ``CLICOLOR`` and
``CLICOLOR_FORCE`` standard UNIX environment variables. When set, the following
rules are applied:

* ``CLICOLOR_FORCE`` != ``0``
   ANSI colors should be enabled no matter what.
* ``CLICOLOR_FORCE`` == ``0``
   Don't output ANSI color escape codes.
* ``CLICOLOR_FORCE`` is unset and ``CLICOLOR`` == ``0``
   Don't output ANSI color escape codes.
* Otherwise
   ANSI colors are enabled if TTY is used.

..
   The above rules are defined in the ``isColored()`` method in the
   ``TextFormatter`` class of the ``logrus`` package.

For example, to disable the output colorization:

.. code-block:: console

   rake run:agent CLICOLOR=0

.. note::

   The values ``true`` and ``false`` are also accepted instead of ``1`` and ``0``.

.. _secure-server-agent:

Securing Connections Between the Stork Server and a Stork Agent
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

Connections between the server and the agents are secured using
standard cryptography solutions, i.e. PKI and TLS.

The server generates the required keys and certificates during its first startup.
They are used to establish safe, encrypted connections between the server
and the agents, with authentication at both ends of these connections.
The agents use the keys and certificates generated by the server to
create agent-side keys and certificates, during the agents' registration
procedure described in the next sections. The private key and CSR
certificate generated by an agent and signed by the server are used for
authentication and connection encryption.

An agent can be registered in the server using one of the two supported
methods:

#. using an agent token
#. using a server token

In the first case, an agent generates a token and passes it to the server
when requesting registration. The server associates the token with the particular
agent. A Stork super administrator must approve the registration request in the web UI,
ensuring that the token displayed in the UI matches the agent's token in the
logs. The Stork agent is typically installed from the Cloudsmith repository
when this registration method is used.

In the second registration method, a server generates a common token for all
new registrations. The super admin must copy the token from the UI and paste
it into the agent's terminal during the interactive agent registration procedure.
This registration method does not require any additional approval of the agent's
registration request in the web UI. If the pasted server token is correct,
the agent should be authorized in the UI when the interactive registration
completes. When this registration method is used, the Stork agent is
typically installed using a script that
downloads the agent packages embedded in the server.

The applicability of the two methods is described in
:ref:`registration-methods-summary`.

The installation and registration processes using each method are described
in the following sections.

.. _securing-connections-between-agent-and-kea-ca:

Securing Connections Between ``stork-agent`` and the Kea Control Agent
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

The Kea Control Agent (CA) may be configured to accept connections only over TLS.
This requires specifying ``trust-anchor``, ``cert-file``, and ``key-file`` values in
the ``kea-ctrl-agent.conf`` file. For details, see the
`Kea Administrator Reference Manual <https://kea.readthedocs.io/en/latest/index.html>`_.

The Stork agent can communicate with Kea over TLS, via the same certificates
that the agent uses in communication with the Stork server.

The Stork agent, by default, requires that the Kea Control Agent provide a trusted TLS certificate.
If Kea uses a self-signed certificate, the Stork agent can be launched with the
``--skip-tls-cert-verification`` flag or ``STORK_AGENT_SKIP_TLS_CERT_VERIFICATION`` environment
variable set to 1, to disable Kea certificate verification.

The Kea CA accepts only requests signed with a trusted certificate, when the ``cert-required`` parameter
is set to ``true`` in the Kea CA configuration file. In this case, the Stork agent must use valid
certificates; it cannot use self-signed certificates created during Stork agent registration.

If the Kea CA is configured to use Basic Auth, the Stork agent will read the credentials from the Kea CA configuration
file. The Stork agent chooses credentials with user name beginning with ``stork``. If there is no such user, the agent
will use the first user from the list.

For example, set the following in the Kea CA configuration file, and save the
password in the ``/etc/kea/stork-api-password`` file:

.. code-block:: json

   "authentication": {
     "type": "basic",
      "realm": "Kea Control Agent",
      "directory": "/etc/kea",
      "clients": [
        {
          "user": "stork-api",
          "password-file": "stork-api-password"
        }
      ]
    }

The Stork agent will use the credentials with the user name ``stork-api`` because it starts with ``stork``. Please
remember that the system user that runs the Stork agent must have read access to the password file.

.. warning::

   Basic HTTP authentication is weak on its own as there are known dictionary attacks,
   but those attacks require a "man in the middle" to get access to the HTTP traffic. That can be eliminated
   by using basic HTTP authentication exclusively over TLS.
   In fact, if possible, using client certificates for TLS is better than using basic HTTP authentication.

.. _register-agent-token-cloudsmith:

Installation From Cloudsmith and Registration With an Agent Token
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

This section describes how to install an agent from a Cloudsmith repository and
perform the agent's registration using an agent token.

The Stork agent installation steps are similar to the Stork server
installation steps described in :ref:`install-server-deb` and
:ref:`install-server-rpm`. Use one of the following commands, instead
of the server installation commands, depending on the local Linux distribution:

.. code-block:: console

   $ sudo apt install isc-stork-agent

.. code-block:: console

   $ sudo dnf install isc-stork-agent

Next, specify the required settings in the ``/etc/stork/agent.env`` file.
The ``STORK_AGENT_SERVER_URL`` should be the URL on which the server receives the
REST connections, e.g. ``http://stork-server.example.org:8080``. The
``STORK_AGENT_HOST`` should point to the agent's address (or name), e.g.
``stork-agent.example.org``. Finally, a non-default agent port can be
specified with the ``STORK_AGENT_PORT`` variable.

.. note::

   Even though the examples provided in this documentation use the ``http``
   scheme, we highly recommend using secure protocols in production
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

The following log messages should be returned when the agent successfully
sends the registration request to the server:

.. code-block:: text

    machine registered
    stored agent signed cert and CA cert
    registration completed successfully

A server administrator must approve the registration request via the
web UI before a machine can be monitored. Visit the ``Services -> Machines``
page in the Stork UI, and click the ``Unauthorized`` button located above the list of machines
on the right side. This list contains all machines pending registration approval.
Before authorizing a machine, ensure that the agent token displayed on this
list is the same as the agent token in the agent's logs or the
``/var/lib/stork-agent/tokens/agent-token.txt`` file. If they match,
click on the ``Action`` button and select ``Authorize``. The machine
should now be visible on the list of authorized machines.

.. _register-server-token-script:

Installation With a Script and Registration With a Server Token
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

This section describes how to install an agent using a script and packages
downloaded from the Stork server, and register the agent
using a server token.

To enable the installation, download the Stork agent packages from
Cloudsmith for the operating systems on which the agents will be
installed. Next, put the downloaded packages in the ``assets/pkgs``
subdirectory of the directory holding the Stork server's static UI content;
it is defined by the ``STORK_REST_STATIC_FILES_DIR`` environment variable,
and its default location is ``/usr/share/stork/www``. The supported
package types are deb, RPM, and APK. The package file names must start
with the ``isc-stork-agent`` prefix and end with the ``.deb``, ``.rpm``,
or ``.apk`` extension. It is recommended to keep the original filenames.

Open Stork in the web browser and log in as a user from the "super admin" group.
Select ``Services`` and then ``Machines`` from the menu. Click on the
``Installing Stork Agent on a New Machine`` button to display the agent
installation instructions. Copy and paste the commands from the displayed
window into the terminal on the machine where the agent is installed.
These commands are also provided here for convenience:

.. code-block:: console

   $ wget http://stork.example.org:8080/stork-install-agent.sh
   $ chmod a+x stork-install-agent.sh
   $ sudo ./stork-install-agent.sh

``stork.example.org`` is an example URL for the Stork server;
it must be replaced with the real server URL used in the deployment.

The script downloads an OS-specific agent package from the Stork server
(deb or RPM), installs the package, and starts the agent's registration procedure.

In the agent machine's terminal, a prompt for a server token is presented:

.. code-block:: text

    >>>> Server access token (optional):

The server token is available for a super admin user after clicking on the
``Installing Stork Agent on a New Machine`` button in the ``Services -> Machines`` page.
Copy the server token from the dialog box and paste it in at the prompt
displayed on the agent machine.

The following prompt appears next:

.. code-block:: text

    >>>> IP address or FQDN of the host with Stork Agent (the Stork Server will use it to connect to the Stork Agent):

Specify an IP address or fully qualified domain name (FQDN) that the server should use to reach out to an
agent via the secure gRPC channel.

When asked for the port:

.. code-block:: text

   >>>> Port number that Stork Agent will use to listen on [8080]:

specify the port number for the gRPC connections, or hit Enter if the
default port 8080 matches the local settings.

If the registration is successful, the following messages are displayed:

.. code-block:: text

   machine ping over TLS: OK
   registration completed successfully

Unlike with :ref:`register-agent-token-cloudsmith`, this registration method
does not require approval via the web UI. The machine should
already be listed among the authorized machines.

.. _register-agent-token-script:

Installation With a Script and Registration With an Agent Token
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

This section describes how to install an agent using a script and packages downloaded from
the Stork server, and perform the agent's registration using an agent token. It
is an interactive alternative to the procedure described in
:ref:`register-agent-token-cloudsmith`.

Start the interactive registration procedure following the steps in
the :ref:`register-server-token-script` section.

In the agent machine's terminal, a prompt for a server token is presented:

.. code-block:: text

    >>>> Server access token (optional):

Because this registration method does not use the server token, do not type anything
at this prompt. Hit Enter to move on.

The following prompt appears next:

.. code-block:: text

    >>>> IP address or FQDN of the host with Stork Agent (the Stork Server will use it to connect to the Stork Agent):

Specify an IP address or FQDN that the server should use to reach out to an
agent via the secure gRPC channel.

When asked for the port:

.. code-block:: text

   >>>> Port number that Stork Agent will use to listen on [8080]:

specify the port number for the gRPC connections, or hit Enter if the
default port 8080 matches the local settings.

The following log messages should be returned when the agent successfully
sends the registration request to the server:

.. code-block:: text

    machine registered
    stored agent signed cert and CA cert
    registration completed successfully

As with :ref:`register-agent-token-cloudsmith`, the agent's registration
request must be approved in the UI to start monitoring the newly registered
machine.

.. _register-server-token-cloudsmith:

Installation From Cloudsmith and Registration With a Server Token
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

This section describes how to install an agent from the Cloudsmith repository and
perform the agent's registration using a server token. It is an alternative to
the procedure described in :ref:`register-server-token-script`.

.. note::

   During registration with the server token, the Stork agent verifies that
   the server can establish a connection with it using the specified address
   and port. This agent port must be free, meaning the agent must not run
   in the background (e.g., as a ``systemd`` service). If the agent is running,
   it must be stopped before running the registration commands below.

The Stork agent installation steps are similar to the Stork server
installation steps described in :ref:`install-server-deb` and
:ref:`install-server-rpm`. Use one of the following commands, depending on
the Linux distribution:

.. code-block:: console

   $ sudo apt install isc-stork-agent

.. code-block:: console

   $ sudo dnf install isc-stork-agent

Start the interactive registration procedure with the following command:

.. code-block:: console

   $ su stork-agent -s /bin/sh -c 'stork-agent register -u http://stork.example.org:8080'

The last parameter should be the appropriate Stork server URL.

Follow the same registration steps described in :ref:`register-server-token-script`.

Now, start the agent service:

.. code-block:: console

   $ sudo systemctl enable isc-stork-agent
   $ sudo systemctl start isc-stork-agent

To check the status:

.. code-block:: console

   $ sudo systemctl status isc-stork-agent

.. _registration-methods-summary:

Registration Methods Summary
~~~~~~~~~~~~~~~~~~~~~~~~~~~~

Stork supports two different agent-registration methods, described above.
Both methods can be used interchangeably, and it is often a matter of
preference which one the administrator selects. However,
the agent token registration may be more suitable in
some situations. This method requires a server URL, agent address
(or name), and agent port as registration settings. If they are known
upfront, it is possible to prepare a system (or container) image with
the agent offline. After starting the image, the agent sends the
registration request to the server and awaits authorization in the web UI.

The agent registration with the server token is always manual. It
requires copying the token from the web UI, logging into the agent,
and pasting the token. Therefore, the registration using the server
token is not appropriate when it is impossible or awkward to access
the machine's terminal, e.g. in Docker. On the other hand, the
registration using the server token is more straightforward because
it does not require unauthorized agents' approval via the web UI.

If the server token leaks, it poses a risk that rogue agents might register.
In that case, the administrator should regenerate the token to prevent
the uncontrolled registration of new agents. Regeneration of the token
does not affect already-registered agents. The new token must be used
for any new registrations.

The server token can be regenerated in the ``Installing Stork Agent on a New Machine``
dialog box available after navigating to the ``Services -> Machines`` page.

Agent Setup Summary
~~~~~~~~~~~~~~~~~~~

After successful agent setup, the agent periodically tries to detect installed
Kea DHCP or BIND 9 services on the system. If it finds them, they are
reported to the Stork server when it connects to the agent.

Further configuration and usage of the Stork server and the
Stork agent are described in the :ref:`usage` chapter.

.. _inspecting-keys-and-certificates:

Inspecting Keys and Certificates
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

The Stork server maintains TLS keys and certificates internally to secure the
communication between ``stork-server`` and any agents. They can be inspected
and exported using ``stork-tool``, with a command such as:

.. code-block:: console

    $ stork-tool cert-export --db-url postgresql://user:pass@localhost/dbname -f srvcert -o srv-cert.pem

The above command may fail if the database password contains any characters requiring URL
encoding. In this case, a command line with multiple switches can be used instead:

.. code-block:: console

    $ stork-tool cert-export --db-user user --db-password pass --db-host localhost --db-name dbname -f srvcert -o srv-cert.pem

The certificates and secret keys can be inspected using OpenSSL, using commands such as
``openssl x509 -noout -text -in srv-cert.pem`` (for the certificates) and
``openssl ec -noout -text -in cakey`` (for the keys).

There are five secrets that can be
exported or imported: the Certificate Authority secret key (``cakey``), the Certificate Authority certificate (``cacert``),
the Stork server private key (``srvkey``), the Stork server certificate (``srvcert``), and a server token (``srvtkn``).

For more details, please see :ref:`man-stork-tool`.

Using External Keys and Certificates
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

It is possible to use external TLS keys and certificates. They can be imported
to the Stork server using ``stork-tool``:

.. code-block:: console

    $ stork-tool cert-import --db-url postgresql://user:pass@localhost/dbname -f srvcert -i srv-cert.pem

The above command may fail if the database password contains any characters requiring URL
encoding. In this case, a command line with multiple switches can be used instead:

.. code-block:: console

    $ stork-tool cert-import --db-user user --db-password pass --db-host localhost --db-name dbname -f srvcert -i srv-cert.pem

Both the Certificate Authority key and the Certificate Authority certificate must be changed at the same time, as
the certificate depends on the key. If they are changed, then the server key
and certificate must also be changed.

.. note::

   Imported certificates and keys must follow the same standards as those self-generated by
   the Stork server. They must also have the same format. This
   `ISC Knowledgebase article <https://kb.isc.org/docs/importing-external-certificates-to-stork>`_
   provides step-by-step instructions for generating a new set of certificates using OpenSSL and importing
   them to Stork. Note that the example OpenSSL configurations from this article may have to
   be adjusted to the specifics of a given deployment.

For more details, please see :ref:`man-stork-tool`.

Installing the Hooks
--------------------

Hooks are additional files (plugins) that extend the standard Stork
functionalities. They contain functions that are called during the handling of
various operations and can change the typical flow or run in parallel.
Independent developers may create hooks and enhance the Stork applications
with new, optional features.

Hook packages are distributed as RPM and deb packages on Cloudsmith.

Hooks are binary files with the ``.so`` extension, and must be
placed in the hook directory. The default location is
``/usr/lib/stork-agent/hooks`` for the Stork agent and
``/usr/lib/stork-server/hooks`` for the Stork server. The location can be changed using
the ``--hook-directory`` CLI option or by setting the
``STORK_AGENT_HOOK_DIRECTORY`` or ``STORK_SERVER_HOOK_DIRECTORY`` environment
variable.

All hooks must be compatible with the used Stork application (agent or
server) and its exact version. If the hook directory contains non-hook files or
out-of-date hooks, then Stork does not run.

Hooks may provide their own configuration options. The list of available options
is listed in the output of the ``stork-agent --help`` and
``stork-server --help`` commands.

Here is the list of supported Stork server hooks:

* LDAP authentication

   This hook provides the possibility to authenticate users by LDAP credentials,
   fetch their profiles, and map LDAP groups into Stork roles.

Upgrading
---------

Due to the new security model introduced with TLS in Stork 0.15.0,
upgrades from versions 0.14.0 and earlier require the agents
to be re-registered.

The server upgrade procedure is the same as the initial installation procedure.

Install the new packages on the server. Installation scripts in
the deb/RPM package perform the required database and other migrations.

.. _installation_sources:

Installing From Sources
=======================

Compilation Prerequisites
-------------------------

Usually, it is most convenient to install Stork using native packages; see :ref:`compatible_systems` and :ref:`install-pkgs` for
details regarding supported systems. However, the sources can also be built separately.

The dependencies that need to be installed to build the Stork sources are:

 - Rake
 - Java Runtime Environment (only if building natively, not using Docker)
 - Docker (only if running in containers; this is needed to build the demo)
 - Python 3.10 or newer (only if building natively, not using Docker)

Other dependencies are installed automatically in a local directory by Rake tasks, which does not
require root privileges. If the demo environment will be run, Docker is needed but not
Java; Docker installs Java within a container.

For details about the environment, please see the Stork wiki at
https://gitlab.isc.org/isc-projects/stork/-/wikis/Install .

Download Sources
----------------

The Stork sources are available in ISC's GitLab instance:
https://gitlab.isc.org/isc-projects/stork.

To get the latest sources, invoke:

.. code-block:: console

   $ git clone https://gitlab.isc.org/isc-projects/stork

Building
--------

There are two Stork components:

- ``stork-agent`` - this is a binary, written in Go
- ``stork-server`` - this is comprised of two parts:
  - backend service - a binary, written in Go
  - frontend - an Angular application written in Typescript

All components can be built using the following command:

.. code-block:: console

   $ rake build

The agent component is installed using this command:

.. code-block:: console

   $ rake install:agent

and the server component with this command:

.. code-block:: console

   $ rake install:server

By default, all components are installed in the specific system directories;
this is useful for installation in a production environment. For testing
purposes the installation can be customized via the ``DEST`` variable, e.g.:

.. code-block:: console

   $ rake install:server DEST=/home/user/stork

Installing on FreeBSD
---------------------

Stork is not regularly tested on FreeBSD but can be installed on this operating
system with the manual steps provided below.

The first step is the installation of packages from the repository:

.. code-block:: console

   pkg install ruby
   pkg install rubygem-rake
   pkg install wget
   pkg install openjdk11-jre
   pkg install node14
   pkg install npm-node14
   npm install -g npm
   pkg install python3
   pkg install protobuf
   pkg install gcc
   pkg install gtar

The Stork build system can install all remaining dependencies automatically.

Unfortunately, there is no way to build binary packages for OpenBSD.
However, it is possible to build the contents of the packages (executables, UI, man, and docs).

.. code-block:: console

   rake build:server_dist
   rake build:agent_dist

The output files are located in the ``dist/`` directory.

Installing on OpenBSD
---------------------

Stork is not regularly tested on OpenBSD but can be installed on this operating
system with the manual steps provided below. The installation is similar
to the FreeBSD process.

The first step is the installation of packages from the repository:

.. code:: console

   pkg_add ruby
   ln -s /usr/local/bin/gem31 /usr/local/bin/gem
   gem install --user-install rake
   pkg_add wget
   pkg_add jdk
   pkg_add node
   pkg_add unzip
   pkg_add protobuf
   pkg_add gcc
   pkg_add go

Stork requires Golang version 1.18 or later.

The Stork build system can install all remaining dependencies automatically.

Unfortunately, there is no way to build binary packages for OpenBSD.
However, it is possible to build the contents of the packages (executables, UI, man, and docs).

.. code-block:: console

   rake build:server_dist
   rake build:agent_dist

The output files are located in the ``dist/`` directory.

Cross-compilation
-----------------

.. warning::

   Our tests do not cover the cross-compilation feature and we cannot guarantee that it will work correctly for all users.

The Stork build system fully supports Linux and MacOS operating systems on the AMD64 and ARM64 architectures. It is also
prepared to handle FreeBSD and OpenBSD with some limitations, but support for these systems is not actively maintained.

The Stork agent, server, and tool are written in pure Golang, which means they can be easily cross-compiled on all
supported platforms.

The ``rake utils:list_go_supported_platforms`` command gives a list of all supported operating systems and
architectures.

To build any Stork component for a specific platform, the following environment variables must be provided:
``STORK_GOOS`` (for the operating system), ``STORK_GOARCH`` (for the architecture), and (optionally)
``STORK_GOARM`` (for the ARM version, ARM architectures only). For example:

.. code-block:: console

   rake build:server STORK_GOOS=darwin STORK_GOARCH=arm64 STORK_GOARM=8
   rake build:agent STORK_GOOS=freebsd STORK_GOARCH=amd64

These variables are supported for the ``build:server``, ``build:agent``, and ``build:tool`` commands to compile the
executable binaries. They can also be used with a combination of the ``build:server_pkg`` and ``build:agent_pkg`` commands
to build the packages:

.. code-block:: console

   rake build:server_pkg STORK_GOOS=darwin STORK_GOARCH=arm64
   rake build:agent_pkg STORK_GOOS=freebsd STORK_GOARCH=amd64

.. warning::

   Remember that the output package type always depends on the current operating system, not the executable type. This
   means that specifying the ``darwin`` operating system in ``STORK_GOOS`` and building the package on Debian causes
   the generation of a deb package with a macOS-compatible executable, which is useless.

It is not recommended to compile Stork for 32-bit architectures, as this may cause problems with unexpected integer
overflows. Stork is not designed to operate on non-POSIX platforms, so Windows is not
and will not be supported. Compiling Stork components for Windows is discouraged because Golang's standard library
may suppress some errors related to file operations on the NTFS filesystem.

Security Checklist for Stork Configurations
===========================================

The following list provides a set of recommendations to secure the Stork server and agent installations. The list is not
exhaustive and should be adjusted to the specific deployment requirements.

The Stork Server
----------------

The Stork server configuration is described in detail in the :ref:`server-setup` section.

- Run the Stork server as a non-privileged, dedicated user.
- Limit the Stork server user rights to only the necessary directories and files:

   - ``/etc/stork/server.env`` - the configuration file (read-only)
   - ``/etc/stork/versions.json`` - the current Kea, Stork and BIND 9 software versions metadata file (read-only)
   - ``/share/stork/www`` - the static web files (read-only)
   - ``/share/stork/www/index.html`` - the main web page (write and read)
   - ``/share/stork/www/assets/authentication-methods`` - the authentication icons (write and read)

- (Optional) Set up the Stork server as a ``systemd`` service to restart the server automatically after a system reboot
  or crash and to manage the server's logs.
- (Advanced) Run the Stork server behind a reverse proxy to protect the server from direct access from the Internet, to
  enable more extensive logging, or to restrict access to the server from specific IP addresses.
- Set up TLS/SSL certificates for the web UI and REST API.

If the metrics endpoint is enabled:

- Ensure the ``/metrics`` endpoint is not accessible from the Internet and is allowed only for the Prometheus server. This
  may be achieved by setting up firewall rules or using a reverse proxy.

During Stork server operation:

- Verify the agent token fingerprints before authorizing an agent registration.
- Disable agent registration in settings if new agents are not anticipated.
- Force users to change their passwords if they may have been compromised.

The Stork Agent
---------------

See the :ref:`agent-configuration-settings` section for Stork agent configuration details.

- Run the Stork agent as a dedicated user.
- Limit the Stork agent user rights to only the necessary directories and files. No one except the ``stork-agent`` user and
  the administrator should have access to the agent's data directory:

  - ``/etc/stork/agent.env`` - the configuration file (read-only)
  - ``/var/lib/stork`` - the agent's data directory (write and read)
  - the system process details (i.e., the current working directory, the command-line arguments).

- The Stork agent must have rights to read the system process list.
- (Optional) Set up the Stork agent as a ``systemd`` service to restart the server automatically after a system reboot or
  crash and to manage the server's logs.

If the Stork agent acts as a Prometheus exporter:

- Ensure the connection between the Stork agent and Prometheus is secure and cannot be intercepted. These two services
  exchange data over the network via an insecure protocol (HTTP).

Monitoring Kea
~~~~~~~~~~~~~~

For more details on monitoring Kea with Stork, refer to the
:ref:`securing-connections-between-agent-and-kea-ca` section of this document.

- The Stork agent must have rights to read:

   - the Kea configuration files (e.g., ``/etc/kea/kea-ctrl-agent.conf``)
   - the Kea logs (e.g., ``/var/log/kea/kea-dhcp4.log``)

- The Kea Control Agent must have configured control sockets for each monitored Kea daemon (the ``control-sockets`` property).
   See the `Configuration section in the Kea ARM <https://kea.readthedocs.io/en/latest/arm/agent.html#configuration>`_ for
   a sample configuration.
- All monitored Kea daemons must have the ``control-socket`` property set in the configuration file. Please refer to
  the `Management API section in the Kea ARM <https://kea.readthedocs.io/en/latest/arm/dhcp4-srv.html#management-api-for-the-dhcpv4-server>`_ for more details.

If the Kea Control Agent listens on non-localhost interfaces, it is recommended to:

- Configure the Basic Auth in Kea CA.
- Configure the Kea REST API to be served over TLS by setting the ``trust-anchor``, ``cert-file``, and ``key-file`` properties.

Monitoring BIND 9
~~~~~~~~~~~~~~~~~

- The Stork agent must have rights to:

   - read the BIND 9 configuration files (e.g., ``/etc/bind/named.conf``) and its references (e.g., ``/etc/bind/rndc.key``)
   - read the BIND 9 logs (e.g., ``/var/log/named/named.log``)
   - execute the ``rndc`` and ``named-checkconf`` commands

If BIND 9 listens on non-localhost interfaces, it is recommended to:

- Secure its control channel by setting the RNDC key.

PostgreSQL
----------

Check the :ref:`securing-the-database-connection` section for details on how to configure the database.

- Create a dedicated user for the Stork server and assign a strong password.
- Create a dedicated database for the Stork server.
- Schedule regular backups of the database.
- (Advanced) Use a separate user to perform the database migrations and run the Stork server. The application user should
  have only the rights to perform queries (SELECT, INSERT, UPDATE, DELETE) on the database tables, without the rights to
  create or drop tables. This approach requires the database schema to be manually updated before starting the Stork server
  using the Stork tool.

If the database is not installed on the same machine as the Stork server:

- Configure SSL/TLS for the database connection.

Integration With Prometheus and Grafana
=======================================

Stork can optionally be integrated with `Prometheus <https://prometheus.io/>`_, an open source monitoring and alerting toolkit,
and `Grafana <https://grafana.com/>`_, an easy-to-view analytics platform for querying, visualization, and alerting. Grafana
requires external data storage. Prometheus is currently the only environment supported by both Stork and Grafana; it is possible
to use Prometheus without Grafana, but using Grafana requires Prometheus.

Prometheus Integration
----------------------

The Stork agent, by default, makes
Kea statistics, as well as some BIND 9 statistics, available in a format understandable by Prometheus. In Prometheus nomenclature, the
Stork agent works as a Prometheus "exporter." If the Prometheus server is available, it can
be configured to monitor Stork agents. To enable ``stork-agent``
monitoring, the ``prometheus.yml`` file (which is typically stored in ``/etc/prometheus/``, but this may vary depending on the
installation) must be edited to add the following entries:

.. code-block:: yaml

  # statistics from Kea
  - job_name: 'kea'
    static_configs:
      - targets: ['agent-kea.example.org:9547', 'agent-kea6.example.org:9547', ... ]

  # statistics from bind9
  - job_name: 'bind9'
    static_configs:
      - targets: ['agent-bind9.example.org:9119', 'another-bind9.example.org:9119', ... ]

By default, the Stork agent exports Kea data on TCP port 9547 and BIND 9 data on TCP port 9119. This can be configured using
command-line parameters, or the Prometheus export can be disabled altogether. For details, see the Stork agent manual page
at :ref:`man-stork-agent`.

The Stork server can also be integrated, but Prometheus support for it is disabled by default. To enable it,
run the server with the ``-m`` or ``--metrics`` flag or set the ``STORK_SERVER_ENABLE_METRICS`` environment variable.
Next, update the ``prometheus.yml`` file:

.. code-block:: yaml

   # statistics from Stork Server
   - job_name: 'storkserver'
      static_configs:
         - targets: ['server.example.org:8080']

The Stork server exports metrics on the assigned HTTP/HTTPS port (defined via the ``--rest-port`` flag).

.. note::

   The Prometheus client periodically collects metrics from the clients (``stork-server`` or ``stork-agent``, for example),
   via an HTTP call. By convention, the endpoint that shares the metrics has the ``/metrics`` path.
   This endpoint returns data in Prometheus-specific format.

.. warning::

   The Prometheus ``/metrics`` endpoint does not require authentication. Therefore, securing this endpoint
   from external access is strongly recommended to prevent unauthorized parties from gathering the server's
   metrics. One way to restrict endpoint access is by using an appropriate HTTP proxy configuration
   to allow only local access or access from the Prometheus host. Please consult the NGINX example
   configuration file shipped with Stork.

After restarting, the Prometheus web interface can be used to inspect whether the statistics have been exported properly.
Kea statistics use the ``kea_`` prefix (e.g. ``kea_dhcp4_addresses_assigned_total``); BIND 9
statistics will eventually use the ``bind_`` prefix (e.g. ``bind_incoming_queries_tcp``); and Stork server statistics use the
``storkserver_`` prefix.

Alerting in Prometheus
----------------------

Prometheus provides the ability to configure alerting. A good starting point is the `Prometheus
documentation on alerting <https://prometheus.io/docs/alerting/latest/overview/>`_. Briefly, the
three main steps are: configure the Alertmanager; configure Prometheus to talk to the Alertmanager; and
define the alerting rules in Prometheus. There are no specific requirements or recommendations,
as these are very deployment-dependent. The following is an incomplete list of ideas that could be
considered:

- The ``storkserver_auth_unreachable_machine_total`` metric is reported by ``stork-server`` and shows the
  number of unreachable machines. Its value under normal circumstances should be zero. Configuring
  an alert for non-zero values may be the best indicator of a large-scale problem, such as a whole VM
  or server becoming unavailable.
- The ``storkserver_auth_authorized_machine_total`` and ``storkserver_auth_unauthorized_machine_total``
  metrics may be used to monitor situations when new machines (e.g. by automated VM cloning) may
  appear in the network or existing machines disappear.
- The ``kea_dhcp4_addresses_assigned_total`` metric, along with ``kea_dhcp4_addresses_total``, can be used to
  calculate pool utilization. If the server allocates all available addresses, it is not able to
  handle new devices, which is one of the most common failure cases of the DHCPv4 server. Depending
  on the deployment specifics, a threshold alert when the pool utilization approaches 100% should be
  seriously considered.
- Contrary to popular belief, DHCPv6 can also run out of resources, in particular with prefix
  delegation (PD). The ``kea_dhcp6_pd_assigned_total`` metric divided by ``kea_dhcp6_pd_total`` can be considered
  an indicator of PD pool utilization. It is an important metric if PD is being used.

The alerting mechanism configured in Prometheus has the relative
advantage of not requiring an additional component (Grafana). The alerting rules are defined in a text
file using simple YAML syntax. For details, see the `Prometheus documentation on alerting rules
<https://prometheus.io/docs/prometheus/latest/configuration/alerting_rules/>`_. One potentially
important feature is Prometheus' ability to automatically discover available
Alertmanager instances, which may be helpful in various redundancy considerations. The Alertmanager
provides a rich list of receivers, which are the actual notification mechanisms used: email,
PagerDuty, Pushover, Slack, Opsgenie, webhook, WeChat, and more.

ISC makes no specific recommendations between Prometheus or Grafana; this is a deployment
consideration.

Grafana Integration
-------------------

Stork provides several Grafana templates that can easily be imported, available in the ``grafana/`` directory of the
Stork source code. The currently available templates are ``bind9-resolver.json``, ``kea-dhcp4.json``, and ``kea-dhcp6.json``. Grafana integration requires three steps:

1. Prometheus must be added as a data source. This can be done in several ways, including using the user interface to edit the Grafana
configuration files. This is the easiest method; for details, see the Grafana documentation about Prometheus integration.
Using the Grafana user interface, select Configuration, then Data Sources, then click "Add data source," and choose
Prometheus; then specify the necessary parameters to connect to the Prometheus instance. In test environments, the only
necessary parameter is the URL, but authentication is desirable in most production deployments.

2. Import the existing dashboard. In the Grafana UI, click Dashboards, then Manage, then Import, and select one of the templates, e.g.
``kea-dhcp4.json``. Make sure to select the Prometheus data source added in the previous step. Once imported, the
dashboard can be tweaked as needed.

3. Once Grafana is configured, go to the Stork user interface, log in as "super admin", click Settings in the Configuration menu, and
then add the URL for Grafana that points to the installation. Once this is done, Stork is able to show links
for subnets leading to specific subnets.

Alternatively, a Prometheus data source can be added by editing ``datasource.yaml`` (typically stored in ``/etc/grafana``,
but this may vary depending on the installation) and adding entries similar to this one:

.. code-block:: yaml

   datasources:
   - name: Stork-Prometheus instance
     type: prometheus
     access: proxy
     url: http://prometheus.example.org:9090
     isDefault: true
     editable: false

The Grafana dashboard files can also be copied to ``/var/lib/grafana/dashboards/`` (again, the exact location may vary depending on the
installation).

Example dashboards with some live data can be seen in the `Stork screenshots gallery
<https://gitlab.isc.org/isc-projects/stork/-/wikis/Screenshots#grafana>`_ .

Subnet Identification
---------------------

The Kea Control Agent shares subnet statistics labeled with internal Kea IDs.
The Prometheus/Grafana subnet labels depend on the installed Kea hooks.
By default, the internal, numeric Kea IDs are used.
However, if the ``subnet_cmds`` hook is installed, then the numeric IDs are resolved to subnet prefixes.
This makes the Grafana dashboard more human-friendly and descriptive.

Alerting in Grafana
-------------------

Grafana offers multiple alerting mechanism options that can be used with Stork; users
are encouraged to see the `Grafana page on alerting
<https://grafana.com/docs/grafana/latest/alerting/?pg=docs>`_.

The list of notification channels (i.e. the delivery mechanisms) is extensive, as it supports
email, webhook, Prometheus' Alertmanager, PagerDuty, Slack, Telegram, Discord, Google Hangouts,
Kafka REST Proxy, Microsoft Teams, Opsgenie, Pushover, and more. Existing dashboards provided by
Stork can be modified and new dashboards can be created. Grafana first requires a notification
channel to be configured, via the Alerting -> Notifications Channel menu; once configured, existing panels
can be edited with alert rules. One caveat is that most panels in the Stork dashboards use template
variables, which are not supported in alerting. This `Stack Overflow thread
<https://stackoverflow.com/questions/51053893/grafana-template-variables-are-not-supported-in-alert-queries>`_
discusses several ways to overcome this limitation.

Compared to Prometheus alerting, Grafana alerting is a bit more user-friendly. The alerts
are set using a web interface, with a flexible approach that allows custom notification messages, such as
instructions on what to do when receiving an alert, information on how to treat situations
where received data is null or there is a timeout, etc.

The defined alerts are considered an integral part of a dashboard. This may be a factor in a deployment
configuration, e.g. the dashboard can be tweaked to specific needs and then deployed to multiple
sites.

.. _configuring-deployment-specific-views:

Configuring Deployment-Specific Views
=====================================

Selected UI pages can be customized with deployment-specific information. This section describes
how this can be configured.

Login Page Welcome Message
--------------------------

A custom welcome message can be displayed on the login page. Typically, the login page is used to provide
contact information to a server administrator to request access credentials. The welcome
message should be written to a file using basic HTML format. For example:

.. code-block:: html

   <h3>Welcome to Stork!</h3>
   <p>
      If you are not familiar with the system please first consult
      <a href="https://stork.readthedocs.io/en/latest/" target="_blank" rel="noopener noreferrer"></a>.
   </p>
   <p>
      Please contact <a href="mailto:someone@example.com">System Administrator</a> to request
      service access.
   </p>

This file must be copied to the Stork UI assets directory, which is part of the Stork
server installation tree. If the Stork server is installed in the ``/usr``
directory, the welcome message should be saved as ``/usr/share/stork/www/assets/static-page-content/login-screen-welcome.html``.
Alternatively, a symbolic link can be created. For example:

.. code-block:: console

   $ ln -s ./welcome.html /usr/share/stork/www/assets/static-page-content/login-screen-welcome.html

Remove or unlink the file to remove the custom welcome message.

Deploying and undeploying the welcome message file is also possible using the
Stork tool. To deploy, run:

.. code-block:: console

   $ stork-tool deploy-login-page-welcome -i ./welcome.html

To undeploy:

.. code-block:: console

   $ stork-tool undeploy-login-page-welcome

The optional ``--rest-static-files-dir`` parameter can be used with both commands to specify
the location of the static UI files directory. For example:

.. code-block:: console

   $ stork-tool undeploy-login-page-welcome --rest-static-files-dir /usr/share/stork/www/

.. warning::

   The deployed HTML file length must not exceed 2048 characters.
