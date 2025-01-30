..
   Copyright (C) 2019-2025 Internet Systems Consortium, Inc. ("ISC")

   This Source Code Form is subject to the terms of the Mozilla Public
   License, v. 2.0. If a copy of the MPL was not distributed with this
   file, You can obtain one at http://mozilla.org/MPL/2.0/.

   See the COPYRIGHT file distributed with this work for additional
   information regarding copyright ownership.

.. _man-stork-server:

``stork-server`` - The Main Stork Server
----------------------------------------

Synopsis
~~~~~~~~

:program:`stork-server` [**-h**] [**-v**] [**-m**] [**-u**] [**--dbhost**] [**-p**] [**-d**] [**--db-sslmode**] [**--db-sslcert**] [**--db-sslkey**] [**--db-sslrootcert**] [**--db-trace-queries=**] [**--rest-cleanup-timeout**] [**--rest-graceful-timeout**] [**--rest-max-header-size**] [**--rest-host**] [**--rest-port**] [**--rest-listen-limit**] [**--rest-keep-alive**] [**--rest-read-timeout**] [**--rest-write-timeout**] [**--rest-tls-certificate**] [**--rest-tls-key**] [**--rest-tls-ca**] [**--rest-static-files-dir**] [**--rest-base-url**] [**--rest-versions-url**]

Description
~~~~~~~~~~~

``stork-server`` provides the main Stork server capabilities. In
every Stork deployment, there should be exactly one Stork server.

Arguments
~~~~~~~~~

The Stork server's behavior is controlled with command-line switches and/or
environment variables. The environment variables can be set before running the
server, or they can be loaded from a file using the ``--use-env-file`` and
``--env-file`` flags. Note that some parts of the server configuration can only
be controlled from the web UI, after the server has been started.

``stork-server`` takes the following arguments (equivalent environment
variables are listed in square brackets, where applicable):

``-h|--help``
   Returns the list of available parameters.

``-v|--version``
   Returns the ``stork-server`` version.

``--hook-directory``
   The path to the hook directory. ``[$STORK_SERVER_HOOK_DIRECTORY]``

``--env-file``
   The environment file location; applicable only if ``--use-env-file`` is provided. The default is ``/etc/stork/server.env``.

``--use-env-file``
   Reads the environment variables from the environment file. The default is ``false``.

``-m|--metrics``
   Enables the periodic metrics collector and /metrics HTTP endpoint for Prometheus. This endpoint requires no authentication; it is recommended to restrict external access to it (e.g. using the HTTP proxy). It is disabled by default. ``[$STORK_SERVER_ENABLE_METRICS]``

``--initial-puller-interval``
   The default interval used by pullers fetching data from Kea; if not provided, the recommended values for each puller are used. ``[$STORK_SERVER_INITIAL_PULLER_INTERVAL]``

``-u|--db-user``
   Specifies the user name to be used for database connections. The default is ``stork``. ``[$STORK_DATABASE_USER_NAME]``

``--db-password=``
   Specifies the database password for database connections. If not specified, the user is prompted for the password if necessary. ``[$STORK_DATABASE_PASSWORD]``

``--db-url``
   Specifies the URL to locate and connect to a database; mutually exclusive with the host, port, username, and password. ``[$STORK_DATABASE_URL]``

``--db-host``
   Specifies the name of the host, IP address, or socket path for the database connection. The default value depends on the system. ``[$STORK_DATABASE_HOST]``

``-p|--db-port``
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
   Enables tracing of SQL queries. Possible values are ``run`` - only runtime, without migrations, ``all`` - both migrations and runtime, or ``none`` - disables the query logging.
   ``[$STORK_DATABASE_TRACE]``

``--db-read-timeout``
   The timeout for socket reads; if reached, commands fail instead of blocking. Zero disables the timeout. Requires a unit: either ms (milliseconds), s (seconds), or m (minutes), e.g.: 42s. The default is 0. ``[$STORK_DATABASE_READ_TIMEOUT]``

``--db-write-timeout``
   The timeout for socket writes; if reached, commands fail instead of blocking. Zero disables the timeout Requires a unit: either ms (milliseconds), s (seconds), or m (minutes), e.g.: 42s. The default is 0. ``[$STORK_DATABASE_WRITE_TIMEOUT]``

``--rest-cleanup-timeout``
   Specifies the period, in seconds, to wait before killing idle connections. The default is 10.

``--rest-graceful-timeout``
   Specifies the period, in seconds, to wait before shutting down the server. The default is 15.

``--rest-max-header-size``
   Specifies the maximum number of bytes the server reads when parsing the request header's keys and
   values, including the request line. It does not limit the size of the request body. The default is 1024 (1MB).

``--rest-host``
   Specifies the IP address to listen on for connections over the RESTful API. ``[$STORK_REST_HOST]``

``--rest-port``
   Specifies the port to listen on for connections over the RESTful API. The default is 8080. ``[$STORK_REST_PORT]``

``--rest-listen-limit``
   Specifies the maximum number of outstanding requests.

``--rest-keep-alive``
   Specifies the TCP keep-alive timeout, in minutes, on accepted connections. After this period, the server prunes dead TCP connections (e.g. if a laptop is closed mid-download). The default is 3.

``--rest-read-timeout``
   Specifies the maximum duration, in seconds, before timing out the read of a request. The default is 30.

``--rest-write-timeout``
   Specifies the maximum duration, in seconds, before timing out the write of a response. The default is 60.

``--rest-tls-certificate``
   Specifies the certificate to use for secure connections. ``[$STORK_REST_TLS_CERTIFICATE]``

``--rest-tls-key``
   Specifies the private key to use for secure connections. ``[$STORK_REST_TLS_PRIVATE_KEY]``

``--rest-tls-ca``
   Specifies the Certificate Authority file to be used with a mutual TLS authority. ``[$STORK_REST_TLS_CA_CERTIFICATE]``

``--rest-static-files-dir``
   Specifies the directory with static files for the UI. ``[$STORK_REST_STATIC_FILES_DIR]``

``--rest-base-url``
   The base URL of the UI. This flag should be set if the UI is served from a subdirectory instead of the root URL. It must start and end with a slash. For example: https://www.example.com/admin/stork/ would need to have ``/admin/stork/`` as the base url. The default is ``/``. ``[$STORK_REST_BASE_URL]``

``--rest-versions-url``
   Specifies the URL of the file with current Kea, Stork and BIND 9 software versions metadata. By default, it is `https://www.isc.org/versions.json <https://www.isc.org/versions.json>`_. ``[$STORK_REST_VERSIONS_URL]``

Note that there is no argument for the database password, as command-line arguments can sometimes be seen
by other users. The password can be set using the ``STORK_DATABASE_PASSWORD`` variable.

Stork logs on INFO level by default. Other levels can be configured using the
``STORK_LOG_LEVEL`` variable. Allowed values are: DEBUG, INFO, WARN, ERROR.

To control the logging colorization, Stork supports the ``CLICOLOR`` and
``CLICOLOR_FORCE`` standard UNIX environment variables. Use ``CLICOLOR_FORCE`` to
enforce enabling or disabling ANSI colors usage. Set ``CLICOLOR`` to ``0`` or
``false`` to disable colorization even if the TTY is attached.

Stork evaluates and prioritizes the settings it receives based on where they are applied.
Command-line flags have the highest priority; next are parameters from the
environment file, if the ``--use-env-file`` flag is used. The lowest priority is given
to environment variables.

Examples
~~~~~~~~

To start the Stork server with the local PostgreSQL database, run the following command:

.. code-block:: bash

   $ stork-server

Custom database connection options can also be specified, e.g. host, port, and user:

.. code-block:: bash

   $ stork-server --db-host=localhost --db-port=5432 --db-user=stork

The host may be a socket path. The default value works on most systems, but it
may need to be explicitly specified if a non-standard PostgreSQL
distribution is being used. For example, on a macOS system it may be necessary to run:

.. code-block:: bash

   $ stork-server --db-host=/tmp

To listen on a non-default port and host, run the following command:

.. code-block:: bash

   $ stork-server (...) --rest-host=hostname --rest-port=80

The REST API can be secured with TLS. To enable it, provide the certificate and key:

.. code-block:: bash

   $ stork-server (...) --rest-tls-certificate=/path/to/cert.pem --rest-tls-ca=/path/to/ca.pem --rest-tls-key=/path/to/key.pem

To enable the server's /metrics HTTP endpoint for Prometheus, run the following command:

.. code-block:: bash

   $ stork-server (...) --metrics

The Stork server can be served from a subdirectory. For example, to run it from the http://example.com/stork/ URL, use the following command:

.. code-block:: bash

   $ stork-server (...) --rest-base-url=/stork/

By default, the Stork server reads arguments only from the command line. To read arguments from the environment
file, run the following command:

.. code-block:: bash

   $ stork-server --use-env-file

The default environment file location is ``/etc/stork/server.env``. To specify a different location, run the following
command:

.. code-block:: bash

   $ stork-server --use-env-file --env-file=/path/to/agent.env

Mailing Lists and Support
~~~~~~~~~~~~~~~~~~~~~~~~~

There are public mailing lists available for the Stork project. **stork-users**
(stork-users at lists.isc.org) is intended for Stork users. **stork-dev**
(stork-dev at lists.isc.org) is intended for Stork developers, prospective
contributors, and other advanced users. The lists are available at
https://www.isc.org/mailinglists/. The community provides best-effort support
on both of those lists.

History
~~~~~~~

``stork-server`` was first coded in November 2019 by Michal
Nowikowski and Marcin Siodelski.

See Also
~~~~~~~~

:manpage:`stork-agent(8)`
