.. _installation:

************
Installation
************

Stork can be installed from pre-built packages or from sources. The following sections describe
both methods.

Prerequisites
=============

The ``Stork Agent`` does not require any specific dependencies to run. It can be run immediately after installation.

For the ``Stork Server``, a PostgreSQL database (https://www.postgresql.org/) using at least version 11 of PostgreSQL is required.
(The installation procedure for PostgreSQL is OS-specific and is not included here.)

These instructions prepare a database for use with the ``Stork Server``, with the `stork` database user and `stork` password. 
Next, a database called `stork` is created and the `pgcrypto` extension is enabled in the database.

First, connect to PostgreSQL using `psql` and the `postgres` administration user:

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


Installing from Packages
========================

Stork packages are stored in repositories located on the Cloudsmith service:
https://cloudsmith.io/~isc/repos/stork/packages/. Both Debian/Ubuntu 
and RPM packages may be found there.

Detailed instructions for setting up the operating system to use this repository are available under
the `Set Me Up` button on the Cloudsmith repository page.


Installing on Debian/Ubuntu
---------------------------

``Stork Server`` and ``Stork Agent`` have been tested thoroughly on the Ubuntu 18.04 system.

The basic steps for Debian and Ubuntu are:

.. code-block:: console

   $ curl -1sLf 'https://dl.cloudsmith.io/public/isc/stork/cfg/setup/bash.deb.sh' | sudo bash

The next step is to install the package with ``Stork Server``:

.. code-block:: console

   $ sudo apt install isc-stork-server

Then, install ``Stork Agent``:

.. code-block:: console

   $ sudo apt install isc-stork-agent

It is possible to install both agent and server on the same machine.


Installing on CentOS/RHEL/Fedora
--------------------------------

``Stork Server`` and ``Stork Agent`` have been tested and run on the Fedora 31 system.

The basic steps for RPM-based distributions are:

.. code-block:: console

   $ curl -1sLf 'https://dl.cloudsmith.io/public/isc/stork/cfg/setup/bash.rpm.sh' | sudo bash

The next step is to install the package with ``Stork Server``:

.. code-block:: console

   $ sudo dnf install isc-stork-server

Then, install ``Stork Agent``:

.. code-block:: console

   $ sudo dnf install isc-stork-agent

It is possible to install both agent and server on the same machine.


Initial Setup of Server
-----------------------

These steps are the same for both Debian-based and RPM-based distributions that use `SystemD`.

After installing ``Stork Server`` from the package, configuration of the
basic settings is required. They are stored in ``/etc/stork/server.env``.

These are the required settings to connect with the database:

* STORK_DATABASE_HOST - the address of a PostgreSQL database; default is `localhost`
* STORK_DATABASE_PORT - the port of a PostgreSQL database; default is `5432`
* STORK_DATABASE_NAME - the name of a database; default is `stork`
* STORK_DATABASE_USER_NAME - the username for connecting to the database; default is `stork`
* STORK_DATABASE_PASSWORD - the password for the username connecting to the database

With those settings in place, the ``Stork Server`` service can be enabled and started:

.. code-block:: console

   $ sudo systemctl enable isc-stork-server
   $ sudo systemctl start isc-stork-server

To check the status:

.. code-block:: console

   $ sudo systemctl status isc-stork-server

By default, the ``Stork Server`` web service is exposed on port 8080,
so now it can be visited in a web browser here: http://localhost:8080.

It is possible to put ``Stork Server`` behind an HTTP reverse proxy using
`Nginx` or `Apache`. In the ``Stork Server`` package an example configuration file is provided
for `Nginx`,
in `/usr/share/stork/examples/nginx-stork.conf`.


Initial Setup of Stork Agent
----------------------------

These steps are the same for both Debian-based and RPM-based distributions that use `SystemD`.

After installing ``Stork Agent`` from the package, configuration of the
basic settings is required. They are stored in ``/etc/stork/agent.env``.

These are the required settings to connect with the database:

* STORK_AGENT_ADDRESS - the IP address of the network interface which ``Stork Agent``
  should use for listening for ``Stork Server`` incoming connections;
  default is `0.0.0.0` (ie. listen on all interfaces)
* STORK_AGENT_PORT - the port that should be used for listening; default is `8080`

With those settings in place, the ``Stork Agent`` service can be enabled and started:

.. code-block:: console

   $ sudo systemctl enable isc-stork-server
   $ sudo systemctl start isc-stork-server

To check the status:

.. code-block:: console

   $ sudo systemctl status isc-stork-server

After starting the agent, it periodically tries to detect installed Kea DHCP or BIND 9 services on the system.
If it finds them, they are reported to the ``Stork Server`` when it connects to the agent.

Further configuration and usage of the ``Stork Server`` and the ``Stork Agent`` are described
in the :ref:`usage` chapter.


.. _installation_sources:

Installing from Sources
=======================

Prerequisites
-------------

``Stork`` sources can be built on Ubuntu 18.04 and Fedora 31.

There are several dependencies that need to be installed to build ``Stork`` sources:

 - Rake
 - Java Runtime Environment

Other dependencies are installed locally and automatically by Rake tasks.

For details about the environment, please see the Stork wiki
https://gitlab.isc.org/isc-projects/stork/wikis/Development-Environment.

Download Sources
----------------

Sources of Stork are available on the ISC GitLab: https://gitlab.isc.org/isc-projects/stork.

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

By default, all components are installed to the `root` folder in the current directory; however, this is not useful
for installation in a production environment. It can be customized via the ``DESTDIR`` variable, e.g.:

.. code-block:: console

   $ sudo rake install_server DESTDIR=/usr
