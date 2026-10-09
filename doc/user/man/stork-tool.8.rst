..
   Copyright (C) 2020-2026 Internet Systems Consortium, Inc. ("ISC")

   This Source Code Form is subject to the terms of the Mozilla Public
   License, v. 2.0. If a copy of the MPL was not distributed with this
   file, You can obtain one at http://mozilla.org/MPL/2.0/.

   See the COPYRIGHT file distributed with this work for additional
   information regarding copyright ownership.

.. _man-stork-tool:

``stork-tool`` - A Tool for Managing the Stork Server
-----------------------------------------------------

Synopsis
~~~~~~~~

:program:`stork-tool` [**global options**] command [**command options**]

Description
~~~~~~~~~~~

``stork-tool`` provides the following features:

- Certificate management - The tool allows the Stork server to export keys, certificates,
  and tokens that are used to secure communication between the Stork server
  and Stork agents.

- Database Creation - The tool facilitates creating a new database for the Stork server
  and a user that can access this database with a generated password.

- Database migration - The tool allows database schema migrations to be performed,
  overwriting the existing database schema version and getting its current value.
  There is normally no need to use this, as the Stork server always runs
  the migration scripts on startup.

- Static views deployment - The tool allows custom content to be set in selected
  Stork views (e.g. a custom welcome message on the login page).

- LDAP users migration - The tool updates the unique identifiers of the users
  authenticated via LDAP, which are used to match the LDAP server responses
  with the users in the Stork database.

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

``stork-tool`` offers the following commands for creating the database for the Stork server:

- ``db-create``
  Creates a new database.

- ``db-password-gen``
  Generates a random database password.

There are several options specific to the ``db-create`` command:

``-m``, ``--db-maintenance-name``
   The existing maintenance database name. The default is "postgres". ``[$STORK_DATABASE_MAINTENANCE_NAME]``

``-a``, ``--db-maintenance-user``
   The database administrator user name. The default is "postgres". ``[$STORK_DATABASE_MAINTENANCE_USER_NAME]``

``--db-maintenance-password``
   The database administrator password; if not specified, the user is prompted for the password if necessary. ``[$STORK_DATABASE_MAINTENANCE_PASSWORD]``

``-f``, ``--force``
   Recreates the database and the user if they exist. The default is ``false``.

Examples
........

Create a new database ``stork`` with user ``stork`` and a generated password:

.. code-block:: console

    $ stork-tool db-create --db-maintenance-user postgres --db-name stork --db-user stork
    INFO[2022-01-25 17:04:56]             main.go:145   created database and user for the server with the following credentials  database_name=stork password=L82B+kJEOyhDoMnZf9qPAGyKjH5Qo/Xb user=stork

When a database is created using the ``psql`` tool, it is sometimes useful to generate
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

The following options pertain to the ``db-``, ``cert-``, and ``migrate-ldap-system-users`` commands:

``--db-url=``
   Specifies the URL for the Stork PostgreSQL database; mutually exclusive with the host, port, username, and password. ``[$STORK_DATABASE_URL]``

``-u|--db-user=``
   Specifies the user name for database connections. The default is ``stork``. ``[$STORK_DATABASE_USER_NAME]``

``--db-password=``
   Specifies the database password for database connections. If not specified, the user is prompted for the password if necessary. ``[$STORK_DATABASE_PASSWORD]``

``--db-host=``
   Specifies the name of the host, IP address, or socket path for the database connection. The default value depends on the system. ``[$STORK_DATABASE_HOST]``

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
   Enables tracing of SQL queries. Possible values are ``run`` - only runtime, without migrations, ``all`` - both migrations and runtime, or ``none`` - disables the query logging. ``[$STORK_DATABASE_TRACE_QUERIES]``

``--db-read-timeout``
   The timeout for socket reads. If reached, commands will fail instead of blocking, zero disables the timeout; requires unit: ms (milliseconds), s (seconds), m (minutes), e.g.: 42s The default is 0. ``[$STORK_DATABASE_READ_TIMEOUT]``

``--db-write-timeout``
   The timeout for socket writes; if reached, commands fail instead of blocking. Zero disables the timeout. Requires a unit: ms (milliseconds), s (seconds), or m (minutes), e.g.: 42s. The default is 0. ``[$STORK_DATABASE_WRITE_TIMEOUT]``

``--db-tls-1-2-enabled``
   Enables TLS 1.2 support for database connections. By default, only TLS 1.3 connections are permitted. ``[$STORK_DATABASE_TLS_1_2_ENABLED]``

``-h|--help``
   Shows a help message.

Note that there is no argument for the database password, as command-line arguments can sometimes be seen
by other users. The password can be set using the ``STORK_DATABASE_PASSWORD`` variable.

Stork logs on INFO level by default. Other levels can be configured using the
``STORK_LOG_LEVEL`` variable. Allowed values are: DEBUG, INFO, WARN, ERROR.

To control the logging colorization, Stork supports the ``CLICOLOR`` and
``CLICOLOR_FORCE`` standard UNIX environment variables. Use ``CLICOLOR_FORCE`` to
enforce enabling or disabling ANSI colors usage. Set ``CLICOLOR`` to ``0`` or
``false`` to disable colorization even if the TTY is attached.

Static Views Deployment
~~~~~~~~~~~~~~~~~~~~~~~

To set a custom welcome message on the login screen, first create a short HTML
file with the message contents. Next, deploy the file using the
``deploy-login-page-welcome`` command with the following options:

``-i|--file=``
   An HTML source file with a custom welcome message. ``[$STORK_TOOL_LOGIN_SCREEN_WELCOME_FILE]``

``-d|--rest-static-files-dir=``
   The directory with static files for the UI; if not provided, the tool tries to use default locations. ``[$STORK_TOOL_REST_STATIC_FILES_DIR]``

To remove the welcome message, use the ``undeploy-login-page-welcome`` command
with the following option:

``-d|--rest-static-files-dir=``
   The directory with static files for the UI; if not provided, the tool tries to use default locations. ``[$STORK_TOOL_REST_STATIC_FILES_DIR]``

In a typical installation, there is no need to specify the directory with
the UI static files; ``stork-tool``  assumes the directory relative to its
location. For example, if ``stork-tool`` is installed in the ``/usr/bin`` directory,
it assumes that the directory for UI files is ``/usr/share/stork/www``.

LDAP Users Migration
~~~~~~~~~~~~~~~~~~~~

The default unique identifier attribute of the users authenticated via LDAP
changed in Stork version 2.5.0. The users who logged in via LDAP while Stork
was running a version earlier than 2.5.0 must be migrated after upgrading to
newer versions.

Before version 2.5.0, the Stork LDAP hook used the DN (distinguished name) as
the unique identifier to match the LDAP server authentication responses with
the users in the Stork database. We found that the DN is not suitable for this
purpose because it can change; in larger organizations, it often changes when
users are moved between organizational units. We recommend using fixed, unique
UUID-based identifiers such as ``entryUUID`` or ``objectGUID`` (depending on the
LDAP distribution). Therefore, since version 2.5.0, the default is
``entryUUID``.

As a side effect, the Stork database stores the DN as the unique identifier of
users who logged in while Stork was running a version earlier than 2.5.0, and
it does not match their ``entryUUID``. Such users cannot log in; see the
Troubleshooting chapter of the Stork Administrator Reference Manual (ARM) for
details. Use the ``migrate-ldap-system-users`` command to migrate these users.

The command binds to the LDAP server as the bind user, looks up each user
authenticated via LDAP by their login, and stores the value of the new unique
identifier attribute in the Stork database. The users that cannot be found on
the LDAP server are skipped with a warning. When the migration is completed,
the command logs the number of successfully migrated users and the number of users
for whom the migration failed. It needs to be run only once.

The command takes the database connection options described in the
`Common Options`_ section and the following LDAP options. They are the same as
for the Stork LDAP hook, so use the same values as configured for the Stork
server:

``--ldap.url=``
   The LDAP server access URL (use ldaps:// protocol to connect over TLS) (default: ldap://127.0.0.1:1389). ``[$STORK_SERVER_HOOK_LDAP_URL]``

``--ldap.root=``
   The LDAP root for login user (default: dc=example,dc=org). ``[$STORK_SERVER_HOOK_LDAP_ROOT]``

``--ldap.bind-userdn=``
   The maintenance userdn used to bind to the server for reading user profiles (default: cn=admin,dc=example,dc=org). ``[$STORK_SERVER_HOOK_LDAP_BIND_USERDN]``

``--ldap.bind-password=``
   The maintenance password used to bind to the server for reading user profiles (default: adminpassword). ``[$STORK_SERVER_HOOK_LDAP_BIND_PASSWORD]``

``--ldap.skip-tls-server-verification``
   Skip the TLS server certificate verification - not recommended for the production environments. ``[$STORK_SERVER_HOOK_LDAP_SKIP_SERVER_TLS_VERIFICATION]``

``--ldap.timeout=``
   The LDAP server connection timeout (default: 30s). ``[$STORK_SERVER_HOOK_LDAP_TIMEOUT]``

``--ldap.debug``
   Enable additional debug information about connection to LDAP server. ``[$STORK_SERVER_HOOK_LDAP_DEBUG]``

``--ldap.object-class-user=``
   The name of the user object class in the user schema (default: organizationalPerson). ``[$STORK_SERVER_HOOK_LDAP_OBJECT_CLASS_USER]``

``--ldap.object-class-user-id=``
   The name of the ID property in the user object class (default: uid). ``[$STORK_SERVER_HOOK_LDAP_OBJECT_CLASS_USER_ID]``

``--ldap.object-class-user-unique-identifier=``
   Property name, in the user class, for a unique and persistent identifier for the user (default: entryUUID). ``[$STORK_SERVER_HOOK_LDAP_OBJECT_CLASS_USER_UNIQUE_IDENTIFIER]``

The other options of the Stork LDAP hook (e.g., the group mapping settings)
are accepted but not used by this command.

An example of the command invocation:

.. code-block:: console

    $ stork-tool migrate-ldap-system-users --db-host=localhost --db-user=stork \
        --ldap.url=ldap://ldap.example.org:389 --ldap.root=dc=example,dc=org \
        --ldap.bind-userdn=cn=admin,dc=example,dc=org --ldap.bind-password=secret \
        --ldap.object-class-user-unique-identifier=entryUUID

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
