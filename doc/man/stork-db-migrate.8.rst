..
   Copyright (C) 2020-2021 Internet Systems Consortium, Inc. ("ISC")

   This Source Code Form is subject to the terms of the Mozilla Public
   License, v. 2.0. If a copy of the MPL was not distributed with this
   file, You can obtain one at http://mozilla.org/MPL/2.0/.

   See the COPYRIGHT file distributed with this work for additional
   information regarding copyright ownership.

.. _man-stork-db-migrate:

stork-db-migrate - Stork database migration tool
------------------------------------------------

Synopsis
~~~~~~~~

:program:`stork-db-migrate [options] command`

Description
~~~~~~~~~~~

The ``stork-db-migrate`` tool is an option to assist with database schema migrations.
Usually, there is no need to use this tool, as the Stork server always runs the migration scripts on startup.
However, it may be useful for debugging and manual migrations.

Arguments
~~~~~~~~~

``stork-db-migrate`` takes the following commands:

Available commands:

  ``down``         Revert last migration (or use -t to migrate to a specific version)

  ``init``         Create schema versioning table in the database

  ``reset``        Revert all migrations

  ``set_version``  Set database version without running migrations

  ``up``           Run all available migrations (or use -t to migrate to a specific version)

  ``version``      Print current migration version


Application Options:

``-d``, ``--db-name=``
   the name of the database to connect to. (default: stork) [$STORK_DATABASE_NAME]

``-u``, ``--db-user=``
   the user name to be used for database connections. (default: stork) [$STORK_DATABASE_USER_NAME]

``--db-host=``
   the name of the host where the database is available. (default: localhost) [$STORK_DATABASE_HOST]

``-p``, ``--db-port=``
   the port on which the database is available. (default: 5432) [$STORK_DATABASE_PORT]

``--db-trace-queries=``
   enable tracing SQL queries: run (only runtime, without migrations), all (migrations and run-time)).
   ``all`` is the default and covers both migrations and ``run-time.enable`` tracing SQL queries. [$STORK_DATABASE_TRACE]

``-h``, ``--help``
   show help message

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

The ``stork-db-migrate`` tool was first coded in October 2019 by Marcin Siodelski.

See Also
~~~~~~~~

:manpage:`stork-agent(8)`, :manpage:`stork-server(8)`
