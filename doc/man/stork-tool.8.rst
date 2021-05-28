..
   Copyright (C) 2020-2021 Internet Systems Consortium, Inc. ("ISC")

   This Source Code Form is subject to the terms of the Mozilla Public
   License, v. 2.0. If a copy of the MPL was not distributed with this
   file, You can obtain one at http://mozilla.org/MPL/2.0/.

   See the COPYRIGHT file distributed with this work for additional
   information regarding copyright ownership.

.. _man-stork-tool:

stork-tool - A tool for managing Stork Server
---------------------------------------------

Synopsis
~~~~~~~~

:program:`stork-tool [global options] command [command options]`

Description
~~~~~~~~~~~

The ``stork-tool`` operates in two areas:

- Certificates Management - it allows for exporting Stork Server keys, certificates
  and token that are used for securing communication between Stork Server
  and Stork Agents

- Database Migration - it allows for performing database schema migrations,
  overwriting db schema version and getting its current value;
  usually, there is no need to use this area, as the Stork server always runs
  the migration scripts on startup


Certificates Management
~~~~~~~~~~~~~~~~~~~~~~~

``stork-tool`` offers the following commands:

- ``cert-export``     Export certificate or other secret data

Options specific to ``cert-export`` command:

``-f``, ``--object=``
   the object to dump, it can be one of ``cakey``, ``cacert``, ``srvkey``, ``srvcert``, ``srvtkn``.
   [$STORK_TOOL_CERT_OBJECT]

``-o``, ``--file=``
   the file location where the object should be saved. [$STORK_TOOL_CERT_FILE]

Examples
........

Print CA key in the console:

.. code-block:: console

    $ stork-tool cert-export --db-url postgresql://user:pass@localhost/dbname -f cakey
    INFO[2021-05-25 12:36:07]       connection.go:59    checking connection to database
    INFO[2021-05-25 12:36:07]            certs.go:225   CA key:
    -----BEGIN PRIVATE KEY-----
    MIGHAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBG0wawIBAQQghrTv9SVZ/hv0xSM+
    jvUk+VehIcf1tD/yMfAF4IiVXaahRANCAATgene6dVwo1xCmYjMKYxSrxgOWRm2G
    R5X1x72axq2cAhCFm7EpD88oYZ3EBdoXmG9fihV5ZGtfFkSpIdzCNPQI
    -----END PRIVATE KEY-----

Export server certificate to a file:

.. code-block:: console

    $ stork-tool cert-export --db-url postgresql://user:pass@localhost/dbname -f srvcert -o srv-cert.pem
    INFO[2021-05-25 12:36:46]       connection.go:59    checking connection to database
    INFO[2021-05-25 12:36:46]            certs.go:221   server cert saved to file: srv-cert.pem

Database Migration
~~~~~~~~~~~~~~~~~~

``stork-tool`` offers the following commands:

- ``db-init``         Create schema versioning table in the database

- ``db-up``           Run all available migrations (or use -t to migrate to a specific version)

- ``db-down``         Revert last migration (or use -t to migrate to a specific version)

- ``db-reset``        Revert all migrations

- ``db-version``      Print current migration version

- ``db-set-version``  Set database version without running migrations

Options specific to ``db-up``, ``db-down`` and ``db-set-version`` commands:

``-t``, ``--version=``
   target database schema version. (default: stork) [$STORK_TOOL_DB_VERSION]

Examples
........

Initialize database schema:

.. code-block:: console

    $ STORK_TOOL_DB_PASSWORD=pass stork-tool db-init -u user -d dbname
    INFO[2021-05-25 12:30:53]       connection.go:59    checking connection to database
    INFO[2021-05-25 12:30:53]             main.go:100   Database version is 0 (new version 33 available)

Overwrite the current schema version to an arbitrary value:

.. code-block:: console

    $ STORK_TOOL_DB_PASSWORD=pass stork-tool db-set-version -u user -d dbname -t 42
    INFO[2021-05-25 12:31:30]             main.go:77    Requested setting version to 42
    INFO[2021-05-25 12:31:30]       connection.go:59    checking connection to database
    INFO[2021-05-25 12:31:30]             main.go:94    Migrated database from version 0 to 42

Common Options
~~~~~~~~~~~~~~

Options common for db-* and cert-* commands:

``--db-url=``
   the URL to locate Stork PostgreSQL database. [$STORK_TOOL_DB_URL]

``-u``, ``--db-user=``
   the user name to be used for database connections. (default: stork) [$STORK_TOOL_DB_USER]

``--db-password=``
   the database password to be used for database connections. [$STORK_TOOL_DB_PASSWORD]

``--db-host=``
   the name of the host where the database is available. (default: localhost) [$STORK_TOOL_DB_HOST]

``-p``, ``--db-port=``
   the port on which the database is available. (default: 5432) [$STORK_TOOL_DB_PORT]

``-d``, ``--db-name=``
   the name of the database to connect to. (default: stork) [$STORK_TOOL_DB_NAME]

``--db-trace-queries=``
   enable tracing SQL queries: "run" - only runtime, without migrations, "all" - migrations and run-time.
   [$STORK_TOOL_DB_TRACE_QUERIES]

``-h``, ``--help``
   show help message

Note that there is no argument for the database password, as the command-line arguments can sometimes be seen
by other users. It can be passed using the STORK_TOOL_DB_PASSWORD variable.

Mailing Lists and Support
~~~~~~~~~~~~~~~~~~~~~~~~~

There are public mailing lists available for the Stork project. **stork-users**
(stork-users at lists.isc.org) is intended for Stork users. **stork-dev**
(stork-dev at lists.isc.org) is intended for Stork developers, prospective
contributors, and other advanced users. The lists are available at
https://lists.isc.org. The community provides best-effort support
on both of those lists.

Once stork becomes more mature, ISC will provide professional support
for Stork services.

History
~~~~~~~

The ``stork-tool`` tool was first coded in October 2019 by Marcin Siodelski. That time it was called
``stork-db-migrate``. In 2021 it was refactored to ``stork-tool`` and commands for Certificates Management
were added by Michal Nowikowski.

See Also
~~~~~~~~

:manpage:`stork-agent(8)`, :manpage:`stork-server(8)`
