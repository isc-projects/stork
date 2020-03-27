.. _installation:

************
Installation
************

Stork can be installed from pre-built packages or from sources. The following sections describe
both methods.


Prerequisites
=============

``Stork Server`` and ``Stork Agent`` have been tested thoroughly on Ubuntu 18.04 system.
They are also being tested and run on Fedora 31 system occasionally.

``Stork Agent`` does not require anything specific to run it. It can be installed and immediatelly
can be run.

In case of ``Stork Server`` it is required to set up a PostgreSQL database (https://www.postgresql.org/).
It is required to use at least 11 version of PostgreSQL.

Installation procedure for PostgreSQL  is OS specific and is skipped here. The following instruction
prepares database for working with ``Stork Server``.

Here `stork` database user with `stork` password is created. Next, database called `stork` is created
and `pgcrypto` extension is enabled in this database.

At first connect to PostgreSQL using `psql` and `postgres` administration user:

.. code-block:: console

    $ psql postgres
    psql (11.5)
    Type "help" for help.
    postgres=#

and then prepare the database:

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

Stork packages are stored in repositories located in Cloudsmith service:
https://cloudsmith.io/~isc/repos/stork/packages/. There are stored both Debian/Ubuntu packages
and RPM packages.

Detailed instruction for setting up operating system to use this repository is available under
`Set Me Up` button on this web page.


Installing on Debian/Ubuntu
---------------------------

The basic steps for Debian and Ubuntu look as follows:

.. code-block:: console

   $ curl -1sLf 'https://dl.cloudsmith.io/public/isc/stork/cfg/setup/bash.deb.sh' | sudo bash

The next step is installing package with Stork Server:

.. code-block:: console

   $ sudo apt install isc-stork-server

and installing Stork Agent:

.. code-block:: console

   $ sudo apt install isc-stork-agent

It is possible to install both agent and server on the same machine.


Installing on CentOS/RHEL/Fedora
--------------------------------

The basic steps for RPM based distributions look as follows:

.. code-block:: console

   $ curl -1sLf 'https://dl.cloudsmith.io/public/isc/stork/cfg/setup/bash.rpm.sh' | sudo bash

The next step is installing package with Stork Server:

.. code-block:: console

   $ sudo dnf install isc-stork-server

and installing Stork Agent:

.. code-block:: console

   $ sudo dnf install isc-stork-agent

It is possible to install both agent and server on the same machine.


Initial Setup of Server
-----------------------

These steps are the same for Debian-based and RPM-based distributions that use `SystemD`.

After installing ``Stork Server`` from the package it is required to configure
basic settings. They are stored in ``/etc/stork/server.env``.

The required settings are for connecting with database:

* STORK_DATABASE_HOST - an address of PostgreSQL database, default is `localhost`
* STORK_DATABASE_PORT - a port of PostgreSQL database, default is `5432`
* STORK_DATABASE_NAME - a name of database, default is `stork`
* STORK_DATABASE_USER_NAME - a username for connecting to the database, default is `stork`
* STORK_DATABASE_PASSWORD - a password for username for connecting to the database

Now it is possible to enable and start ``Stork Server`` service:

.. code-block:: console

   $ sudo systemctl enable isc-stork-server
   $ sudo systemctl start isc-stork-server

and then check the status:

.. code-block:: console

   $ sudo systemctl status isc-stork-server

By default ``Stork Server`` web service is exposed on 8080 port,
so now it can be visited in a web browser here: http://localhost:8080.

It is possible to put ``Stork Server`` behind HTTP reverse proxy. For that purpose
`Nginx` or `Apache` can be used. In ``Stork Server`` package there is provided
an exemplary configuration file for `Nginx` which is located
in `/usr/share/stork/examples/nginx-stork.conf`.


Initial Setup of Agent
-----------------------

These steps are the same for Debian-based and RPM-based distributions that use `SystemD`.

After installing ``Stork Agent`` from the package it is required to configure
basic settings. They are stored in ``/etc/stork/agent.env``.

The required settings are for connecting with database:

* STORK_AGENT_ADDRESS - an IP address of network interface which ``Stork Agent``
  should use for listening for ``Stork Server`` incoming connections,
  default is `0.0.0.0` ie. listen on all interfaces
* STORK_AGENT_PORT - a port that should be used for listening, default is `8080`

Now it is possible to enable and start ``Stork Agent`` service:

.. code-block:: console

   $ sudo systemctl enable isc-stork-server
   $ sudo systemctl start isc-stork-server

and then check the status:

.. code-block:: console

   $ sudo systemctl status isc-stork-server

After starting the agent it periodically tries to detect installed services on the system.
It looks for Kea or BIND 9 services. When it finds any of them then they will be reported
to ``Stork Server`` when it connects to this agent.

Further configuration and usage of ``Stork Server`` and ``Stork Agent`` are described
in :ref:`usage` chapter.


.. _installation_sources:

Installing from Sources
=======================

Prerequisites
-------------

``Stork`` sources can be built on Ubuntu 18.04 and Fedora 31.

There are several dependencies that needs to be installed to build ``Stork`` sources :

 - Rake
 - Java Runtime Environment

Other dependencies are installed locally, automatically by Rake tasks.

For details about environment, please see Stork wiki
https://gitlab.isc.org/isc-projects/stork/wikis/Development-Environment .

Download Sources
----------------

Sources of Stork are available on ISC GitLab: https://gitlab.isc.org/isc-projects/stork.

To get the latest sources invoke:
.. code-block:: console

   $ git clone https://gitlab.isc.org/isc-projects/stork

Building
--------

There are several parts of ``Stork``:

- ``Stork Agent`` - this is only one binary `stork-agent` which is written in Go language
- ``Stork Server`` - it compraises of two parts:
  - `backend service` - written in Go language
  - `frontend` - an `Angular` application written in Typescript

All parts can be build using the following command:

.. code-block:: console

   $ rake build_all

Then it is possible to install agent part using this command:

.. code-block:: console

   $ rake install_agent

and server part with this command:

.. code-block:: console

   $ rake install_server

By default all parts are installed to `root` folder in current directory. This is not useful
for production installation. It can be customized by ``DESTDIR`` variable, e.g.:

.. code-block:: console

   $ sudo rake install_server DESTDIR=/usr
