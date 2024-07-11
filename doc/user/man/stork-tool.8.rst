..
   Copyright (C) 2020-2024 Internet Systems Consortium, Inc. ("ISC")

   This Source Code Form is subject to the terms of the Mozilla Public
   License, v. 2.0. If a copy of the MPL was not distributed with this
   file, You can obtain one at http://mozilla.org/MPL/2.0/.

   See the COPYRIGHT file distributed with this work for additional
   information regarding copyright ownership.

.. _man-stork-tool:

``stork-tool`` - A Tool for Managing Stork Server
-------------------------------------------------

Synopsis
~~~~~~~~

:program:`stork-tool` [**global options**] command [**command options**]

Description
~~~~~~~~~~~

``stork-tool`` provides three features:

- Certificate management - it allows the Stork server to export keys, certificates
  and tokens that are used to secure communication between Stork server
  and Stork agents.

- Database Creation - it facilitates creating a new database for the Stork Server,
  and a user that can access this database with a generated password

- Database migration - it allows database schema migrations to be performed,
  overwriting the database schema version and getting its current value.
  There is normally no need to use this, as the Stork server always runs
  the migration scripts on startup.

Certificate Management
~~~~~~~~~~~~~~~~~~~~~~

``stork-tool`` takes the following arguments (equivalent environment variables are listed in square brackets, where applicable):

- ``cert-export``
  Exports a certificate or other secret data. The options are:

  ``-f|--object=``
   Specifies the object to dump, which can be one of ``cakey``, ``cacert``, ``srvkey``, ``srvcert``, or ``srvtkn``.
   ``[$STORK_TOOL_CERT_OBJECT]``

  ``-o|--file=``
   Specifies the location of the file where the object should be saved. ``[$STORK_TOOL_CERT_FILE]``

  To print the Certificate Authority key in the console:

  .. code-block:: console

      $ stork-tool cert-export --db-url postgresql://user:pass@localhost/dbname -f cakey
      INFO[2021-05-25 12:36:07]       connection.go:59    checking connection to database
      INFO[2021-05-25 12:36:07]            certs.go:225   CA key:
      -----BEGIN PRIVATE KEY-----
      MIGHAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBG0wawIBAQQghrTv9SVZ/hv0xSM+
      jvUk+VehIcf1tD/yMfAF4IiVXaahRANCAATgene6dVwo1xCmYjMKYxSrxgOWRm2G
      R5X1x72axq2cAhCFm7EpD88oYZ3EBdoXmG9fihV5ZGtfFkSpIdzCNPQI
      -----END PRIVATE KEY-----

  To export the server certificate to a file:

  .. code-block:: console

      $ stork-tool cert-export --db-url postgresql://user:pass@localhost/dbname -f srvcert -o srv-cert.pem
      INFO[2021-05-25 12:36:46]       connection.go:59    checking connection to database
      INFO[2021-05-25 12:36:46]            certs.go:221   server cert saved to file: srv-cert.pem

- ``cert-import``
  Imports a certificate or other secret data. The options are:

  ``-f|--object=``
   Specifies the object to dump, which can be one of ``cakey``, ``cacert``, ``srvkey``, ``srvcert``, or ``srvtkn``.
   ``[$STORK_TOOL_CERT_OBJECT]``

  ``-i``, ``--file=``
  Specifies the location of the file from which the object is loaded. ``[$STORK_TOOL_CERT_FILE]``

  To read the server token from stdin:

  .. code-block:: console

      $ echo abc | stork-tool cert-import --db-url postgresql://user:pass@localhost/dbname -f srvtkn
      INFO[2021-08-11 13:31:55]       connection.go:59    checking connection to database
      INFO[2021-08-11 13:31:55]            certs.go:259   reading server token from stdin
      INFO[2021-08-11 13:31:55]            certs.go:261   server token read from stdin, length 4

  To import the server certificate from a file:

  .. code-block:: console

      $ stork-tool cert-import --db-url postgresql://user:pass@localhost/dbname -f srvcert -i srv.cert
      INFO[2021-08-11 15:22:28]       connection.go:59    checking connection to database
      INFO[2021-08-11 15:22:28]            certs.go:257   server cert loaded from srv.cert file, length 14

Database Creation
~~~~~~~~~~~~~~~~~

``stork-tool`` offers the following commands for creating the database for the Stork Server:

- ``db-create``       Create new database

- ``db-password-gen`` Generate random database password

Options specific to ``db-create`` command:

``-m``, ``--db-maintenance-name``
   Existing maintenance database name. The default is "postgres". ``[$STORK_DATABASE_MAINTENANCE_NAME]``

``-a``, ``--db-maintenance-user``
   Database administrator user name. The default is "postgres". ``[$STORK_DATABASE_MAINTENANCE_USER_NAME]``

``--db-maintenance-password``
   Database administrator password; if not specified, the user will be prompted for the password if necessary. ``[$STORK_DATABASE_MAINTENANCE_PASSWORD]``

``-f``, ``--force``
   Recreate the database and the user if they exist. The default is false.

Examples
........

Create a new database ``stork`` with user ``stork`` and a generated password:

.. code-block:: console

    $ stork-tool db-create --db-maintenance-user postgres --db-name stork --db-user stork
    INFO[2022-01-25 17:04:56]             main.go:145   created database and user for the server with the following credentials  database_name=stork password=L82B+kJEOyhDoMnZf9qPAGyKjH5Qo/Xb user=stork

When a database is created using ``psql`` tool, it is sometimes useful to generate
a hard-to-guess password for this database:

.. code-block:: console

    $ stork-tool db-password-gen
    INFO[2022-01-25 17:56:31]             main.go:157   generated new database password               password=znYDfWzvMhWRZyJJuu3EvUxH5KMi1SmJ

Database Migration
~~~~~~~~~~~~~~~~~~

``stork-tool`` offers the following commands:

- ``db-init``
  Creates a schema versioning table in the database.

- ``db-up``
  Runs all available migrations; use ``-t`` to migrate to a specific version.

- ``db-down``
  Reverts the last migration; use ``-t`` to migrate to a specific version.

- ``db-reset``
  Reverts all migrations.

- ``db-version``
  Prints the current migration version.

- ``db-set-version``
  Sets the database version without running migrations.

  The following option is specific to the ``db-up``, ``db-down``, and ``db-set-version`` commands:

  ``-t|--version=``
   Specifies the target database schema version. The default is ``stork``. ``[$STORK_TOOL_DB_VERSION]``

To initialize a database schema:

.. code-block:: console

    $ STORK_DATABASE_PASSWORD=pass stork-tool db-init -u user -d dbname
    INFO[2021-05-25 12:30:53]       connection.go:59    checking connection to database
    INFO[2021-05-25 12:30:53]             main.go:100   Database version is 0 (new version 33 available)

To overwrite the current schema version to an arbitrary value:

.. code-block:: console

    $ STORK_DATABASE_PASSWORD=pass stork-tool db-set-version -u user -d dbname -t 42
    INFO[2021-05-25 12:31:30]             main.go:77    Requested setting version to 42
    INFO[2021-05-25 12:31:30]       connection.go:59    checking connection to database
    INFO[2021-05-25 12:31:30]             main.go:94    Migrated database from version 0 to 42

Common Options
~~~~~~~~~~~~~~

The following options pertain to both ``db-`` and ``cert-`` commands:

``--db-url=``
   Specifies the URL for the Stork PostgreSQL database. It's mutually exclusive with the host, port, username, and password. ``[$STORK_DATABASE_URL]``

``-u|--db-user=``
   Specifies the user name for database connections. The default is ``stork``. ``[$STORK_DATABASE_USER_NAME]``

``--db-password=``
   Specifies the database password for database connections. If not specified, the user will be prompted for the password if necessary. ``[$STORK_DATABASE_PASSWORD]``

``--db-host=``
   Specifies the name of the host, IP address or a socket path for the database connection. The default value depends on the system. ``[$STORK_DATABASE_HOST]``

``-p|--db-port=``
   Specifies the port on which the database is available. The default is 5432. ``[$STORK_DATABASE_PORT]``

``-d|--db-name=``
   Specifies the name of the database to connect to. The default is ``stork``. ``[$STORK_DATABASE_NAME]``

``--db-sslmode``
   Specifies the SSL mode for connecting to the database; possible values are ``disable``, ``require``, ``verify-ca``, or ``verify-full``. The default is ``disable``. ``[$STORK_DATABASE_SSLMODE]`` Acceptable values are:

   ``disable``
   Disables encryption between the Stork server and the PostgreSQL database.

   ``require``
   Uses secure communication but does not verify the server's identity, unless the
   root certificate location is specified and that certificate exists.
   If the root certificate exists, the behavior is the same as in the case of ``verify-ca``.

   ``verify-ca``
   Uses secure communication and verifies the server's identity by checking it
   against the root certificate stored on the Stork server machine.

   ``verify-full``
   Uses secure communication and verifies the server's identity against the root
   certificate. In addition, checks that the server hostname matches the
   name stored in the certificate.

``--db-sslcert``
   Specifies the location of the SSL certificate used by the server to connect to the database. ``[$STORK_DATABASE_SSLCERT]``

``--db-sslkey``
   Specifies the location of the SSL key used by the server to connect to the database. ``[$STORK_DATABASE_SSLKEY]``

``--db-sslrootcert``
   Specifies the location of the root certificate file used to verify the database server's certificate. ``[$STORK_DATABASE_SSLROOTCERT]``

``--db-trace-queries=``
   Enables tracing of SQL queries. Possible values are ``run`` - only runtime, without migrations, ``all`` - both migrations and runtime, or ``none`` - disable the query logging. ``[$STORK_DATABASE_TRACE_QUERIES]``

``--db-read-timeout``
   Timeout for socket reads. If reached, commands will fail instead of blocking, zero disables the timeout. The default is 0. ``[$STORK_DATABASE_READ_TIMEOUT]``

``--db-write-timeout``
   Timeout for socket writes. If reached, commands will fail instead of blocking, zero disables the timeout. The default is 0. ``[$STORK_DATABASE_WRITE_TIMEOUT]``

``-h|--help``
   Shows a help message.

Note that there is no argument for the database password, as the command-line arguments can sometimes be seen
by other users. It can be passed using the ``STORK_DATABASE_PASSWORD`` variable.

Stork logs on INFO level by default. Other levels can be configured using the
``STORK_LOG_LEVEL`` variable. Allowed values are: DEBUG, INFO, WARN, ERROR.

To control the logging colorization, Stork supports the ``CLICOLOR`` and
``CLICOLOR_FORCE`` standard UNIX environment variables. Use ``CLICOLOR_FORCE`` to
enforce enabling or disabling the ANSI colors usage. Set ``CLICOLOR`` to ``0`` or
``false`` to disable colorization even if the TTY is attached.

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

``stork-tool`` was first coded in October 2019 by Marcin Siodelski; at that time it was called
``stork-db-migrate``. In 2021, it was refactored as ``stork-tool`` and commands for Certificate Management
were added by Michal Nowikowski.

See Also
~~~~~~~~

:manpage:`stork-agent(8)`, :manpage:`stork-server(8)`
