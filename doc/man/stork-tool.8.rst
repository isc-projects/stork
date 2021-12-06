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
  usually, there is no need to use this area, as the Stork Server always runs
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

- ``cert-import``     Import certificate or other secret data

Options specific to ``cert-import`` command:

``-f``, ``--object=``
   the object to dump, it can be one of ``cakey``, ``cacert``, ``srvkey``, ``srvcert``, ``srvtkn``.
   [$STORK_TOOL_CERT_OBJECT]

``-i``, ``--file=``
   the file location from which the object will be loaded. [$STORK_TOOL_CERT_FILE]


Examples
........

Read server token from stdin:

.. code-block:: console

    $ echo abc | stork-tool cert-import --db-url postgresql://user:pass@localhost/dbname -f srvtkn
    INFO[2021-08-11 13:31:55]       connection.go:59    checking connection to database
    INFO[2021-08-11 13:31:55]            certs.go:259   reading server token from stdin
    INFO[2021-08-11 13:31:55]            certs.go:261   server token read from stdin, length 4

Import server certificate from a file:

.. code-block:: console

    $ stork-tool cert-import --db-url postgresql://user:pass@localhost/dbname -f srvcert -i srv.cert
    INFO[2021-08-11 15:22:28]       connection.go:59    checking connection to database
    INFO[2021-08-11 15:22:28]            certs.go:257   server cert loaded from srv.cert file, length 14

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

    $ STORK_DATABASE_PASSWORD=pass stork-tool db-init -u user -d dbname
    INFO[2021-05-25 12:30:53]       connection.go:59    checking connection to database
    INFO[2021-05-25 12:30:53]             main.go:100   Database version is 0 (new version 33 available)

Overwrite the current schema version to an arbitrary value:

.. code-block:: console

    $ STORK_DATABASE_PASSWORD=pass stork-tool db-set-version -u user -d dbname -t 42
    INFO[2021-05-25 12:31:30]             main.go:77    Requested setting version to 42
    INFO[2021-05-25 12:31:30]       connection.go:59    checking connection to database
    INFO[2021-05-25 12:31:30]             main.go:94    Migrated database from version 0 to 42

Common Options
~~~~~~~~~~~~~~

Options common for db-* and cert-* commands:

``--db-url=``
   the URL to locate Stork PostgreSQL database. [$STORK_DATABASE_URL]

``-u``, ``--db-user=``
   the user name to be used for database connections. (default: stork) [$STORK_DATABASE_USER_NAME]

``--db-password=``
   the database password to be used for database connections. [$STORK_DATABASE_PASSWORD]

``--db-host=``
   the name of the host where the database is available. (default: localhost) [$STORK_DATABASE_HOST]

``-p``, ``--db-port=``
   the port on which the database is available. (default: 5432) [$STORK_DATABASE_PORT]

``-d``, ``--db-name=``
   the name of the database to connect to. (default: stork) [$STORK_DATABASE_NAME]

``--db-sslmode``
   the SSL mode for connecting to the database (i.e., disable, require, verify-ca or verify-full). (default: disable) [$STORK_DATABASE_SSLMODE]

``--db-sslcert``
   the location of the SSL certificate used by the server to connect to the database. [$STORK_DATABASE_SSLCERT]

``--db-sslkey``
   the location of the SSL key used by the server to connect to the database. [$STORK_DATABASE_SSLKEY]

``--db-sslrootcert``
   the location of the root certificate file used to verify the database server's certificate. [$STORK_DATABASE_SSLROOTCERT]

``--db-trace-queries=``
   enable tracing SQL queries: "run" - only runtime, without migrations, "all" - migrations and run-time.
   [$STORK_DATABASE_TRACE_QUERIES]

``-h``, ``--help``
   show help message

The ``--db-sslmode`` argument can have one of the following values:

``disable``
  disable encryption between the Stork Server and the PostgreSQL database.

``require``
  use secure communication but do not verify the server's identity unless the
  root certificate location is specified and that certificate exists
  If the root certificate exists, the behavior is the same as in case of `verify-ca`
  mode.

``verify-ca``
  use secure communication and verify the server's identity by checking it
  against the root certificate stored on the Stork Server machine.

``verify-full``
  use secure communication, verify the server's identity against the root
  certificate. In addition, check that the server hostname matches the
  name stored in the certificate.

Note that there is no argument for the database password, as the command-line arguments can sometimes be seen
by other users. It can be passed using the STORK_DATABASE_PASSWORD variable.

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
