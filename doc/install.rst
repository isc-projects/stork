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

Note that MacOS is not and will not be officially supported. Many developers in our team use macs, so we're trying to keep Stork
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

Stork requires the premium Host Commands hook library to retrieve host reservations stored in an external database. Stork will
work without the Host Commands hook, but will not be able to display host reservations. Stork can retrieve host reservations
stored locally in the Kea configuration without any additional hook libraries.

For the ``Stork Server``, a PostgreSQL database
(https://www.postgresql.org/) version 11 or later
is required. The general installation procedure for PostgreSQL is
OS-specific and is not included here. However, please keep in mind that Stork
uses pgcrypto extensions, which are often coming in a separate package. For
example, you need postgresql-crypto package on Fedora and postgresql12-contrib
on RHEL and CentOS.

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


.. _install-pkgs:

Installing from Packages
========================

Stork packages are stored in repositories located on the Cloudsmith
service: https://cloudsmith.io/~isc/repos/stork/packages/. Both
Debian/Ubuntu and RPM packages may be found there.

Detailed instructions for setting up the operating system to use this
repository are available under the `Set Me Up` button on the
Cloudsmith repository page.


Installing on Debian/Ubuntu
---------------------------

The first step for both Debian and Ubuntu is:

.. code-block:: console

   $ curl -1sLf 'https://dl.cloudsmith.io/public/isc/stork/cfg/setup/bash.deb.sh' | sudo bash

Next, install the package with ``Stork Server``:

.. code-block:: console

   $ sudo apt install isc-stork-server

Then, install ``Stork Agent``:

.. code-block:: console

   $ sudo apt install isc-stork-agent

It is possible to install both agent and server on the same machine.


Installing on CentOS/RHEL/Fedora
--------------------------------

The first step for RPM-based distributions is:

.. code-block:: console

   $ curl -1sLf 'https://dl.cloudsmith.io/public/isc/stork/cfg/setup/bash.rpm.sh' | sudo bash

Next, install the package with ``Stork Server``:

.. code-block:: console

   $ sudo dnf install isc-stork-server

Then, install ``Stork Agent``:

.. code-block:: console

   $ sudo dnf install isc-stork-agent

It is possible to install both agent and server on the same machine. If ``dnf`` is not available, ``yum`` can be used in similar
fashion.

Initial Setup of the Stork Server
---------------------------------

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


Initial Setup of the Stork Agent
--------------------------------

These steps are the same for both Debian-based and RPM-based
distributions that use `SystemD`.

After installing ``Stork Agent`` from the package, the basic settings
must be configured. They are stored in ``/etc/stork/agent.env``.

These are the required settings to connect with the database:

* STORK_AGENT_ADDRESS - the IP address of the network interface which ``Stork Agent``
  should use for listening for ``Stork Server`` incoming connections;
  default is `0.0.0.0` (i.e. listen on all interfaces)
* STORK_AGENT_PORT - the port that should be used for listening; default is `8080`

With those settings in place, the ``Stork Agent`` service can be
enabled and started:

.. code-block:: console

   $ sudo systemctl enable isc-stork-agent
   $ sudo systemctl start isc-stork-agent

To check the status:

.. code-block:: console

   $ sudo systemctl status isc-stork-agent

After starting, the agent periodically tries to detect installed
Kea DHCP or BIND 9 services on the system.  If it finds them, they are
reported to the ``Stork Server`` when it connects to the agent.

Further configuration and usage of the ``Stork Server`` and the
``Stork Agent`` are described in the :ref:`usage` chapter.


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
