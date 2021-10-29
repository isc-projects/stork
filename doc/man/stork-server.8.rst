..
   Copyright (C) 2019-2021 Internet Systems Consortium, Inc. ("ISC")

   This Source Code Form is subject to the terms of the Mozilla Public
   License, v. 2.0. If a copy of the MPL was not distributed with this
   file, You can obtain one at http://mozilla.org/MPL/2.0/.

   See the COPYRIGHT file distributed with this work for additional
   information regarding copyright ownership.

.. _man-stork-server:

stork-server - Main Stork server
---------------------------------

Synopsis
~~~~~~~~

:program:`stork-server`

Description
~~~~~~~~~~~

``stork-server`` provides the main Stork server capabilities. In
every Stork deployment, there should be exactly one Stork server.

Arguments
~~~~~~~~~

``stork-server`` takes the following arguments:

``-h`` or ``--help``
   the list of available parameters.

``-v`` or ``--version``
   the ``stork-server`` version.

``-m`` or ``--metrics``
   enable the periodic metrics collector and /metrics HTTP endpoint for Prometheus. This endpoint requires no authentication and it is recommended to restrict external access to it (e.g. using the HTTP proxy). (default: disabled)

``-u`` or ``--db-user``
   the user name to be used for database connections. (default: stork) [$STORK_DATABASE_USER_NAME]

``--db-host``
   the name of the host where the database is available. (default: localhost) [$STORK_DATABASE_HOST]

``-p`` or ``--db-port``
   the port on which the database is available. (default: 5432) [$STORK_DATABASE_PORT]

``-d`` or ``--db-name=``
   the name of the database to connect to. (default: stork) [$STORK_DATABASE_NAME]

``--db-trace-queries=``
   enable tracing SQL queries: "run" - only run-time, without migrations), "all" - migrations and run-time.
   [$STORK_DATABASE_TRACE]

``--rest-cleanup-timeout``
   the period to wait before killing idle connections. (default: 10s)

``--rest-graceful-timeout``
   the period to wait before shutting down the server. (default: 15s)

``--rest-max-header-size``
   the maximum number of bytes the server reads parsing the request header's keys and
   values, including the request line. It does not limit the size of the request body. (default: 1MiB)

``--rest-host``
   the IP to listen on for connections over the REST API. [$STORK_REST_HOST]

``--rest-port``
   the port to listen on for connections over the REST API. (default: 8080) [$STORK_REST_PORT]

``--rest-listen-limit``
   the maximum number of outstanding requests.

``--rest-keep-alive``
   the TCP keep-alive timeout on accepted connections. It prunes dead TCP connections ( e.g. closing laptop mid-download). (default: 3m)

``--rest-read-timeout``
   the maximum duration before timing out a read of the request. (default: 30s)

``--rest-write-timeout``
   the maximum duration before timing out a write of the response. (default: 60s)

``--rest-tls-certificate``
   the certificate to use for secure connections. [$STORK_REST_TLS_CERTIFICATE]

``--rest-tls-key``
   the private key to use for secure connections. [$STORK_REST_TLS_PRIVATE_KEY]

``--rest-tls-ca``
   the certificate authority file to be used with a mutual TLS authority. [$STORK_REST_TLS_CA_CERTIFICATE]

``--rest-static-files-dir``
   the directory with static files for the UI. [$STORK_REST_STATIC_FILES_DIR]

Note that there is no argument for database password, as the command-line arguments can sometimes be seen
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

``stork-server`` was first coded in November 2019 by Michal
Nowikowski and Marcin Siodelski.

See Also
~~~~~~~~

:manpage:`stork-agent(8)`
