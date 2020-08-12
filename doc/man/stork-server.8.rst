..
   Copyright (C) 2019-2020 Internet Systems Consortium, Inc. ("ISC")

   This Source Code Form is subject to the terms of the Mozilla Public
   License, v. 2.0. If a copy of the MPL was not distributed with this
   file, You can obtain one at http://mozilla.org/MPL/2.0/.

   See the COPYRIGHT file distributed with this work for additional
   information regarding copyright ownership.


stork-server - The central Stork server
---------------------------------------

Synopsis
~~~~~~~~

:program:`stork-server`

Description
~~~~~~~~~~~

The ``stork-server`` provides the main Stork Server capabilities. In
every Stork deployment, there should be exactly one stork-server.

Arguments
~~~~~~~~~

The Stork Server takes the following arguments:

``-h`` or ``--help``
   displays list of available parameters.

``-v`` or ``--version``
   returns stork-server version.

``-d`` or ``--db-name=``
   the name of the database to connect to (default: stork) [$STORK_DATABASE_NAME]

``-u`` or ``--db-user``
   the user name to be used for database connections (default: stork) [$STORK_DATABASE_USER_NAME]

``--db-host``
   the name of the host where database is available (default: localhost) [$STORK_DATABASE_HOST]

``-p`` or ``--db-port``
   the port on which the database is available (default: 5432) [$STORK_DATABASE_PORT]

``--db-trace-queries``
   Enable tracing SQL queries [$STORK_DATABASE_TRACE]

``--rest-cleanup-timeout``
   grace period for which to wait before killing idle connections (default: 10s)

``--rest-graceful-timeout``
   grace period for which to wait before shutting down the server (default: 15s)

``--rest-max-header-size``
   controls the maximum number of bytes the server will read parsing the request header's keys and
   values, including the request line. It does not limit the size of the request body. (default: 1MiB)

``--rest-host``
   the IP to listen on for connections over ReST API [$STORK_REST_HOST]

``--rest-port``
   the port to listen on for connections over ReST API (default: 8080) [$STORK_REST_PORT]

``--rest-listen-limit``
   limit the number of outstanding requests

``--rest-keep-alive``
   set the TCP keep-alive timeouts on accepted connections. It prunes dead TCP connections ( e.g. closing laptop mid-download) (default: 3m)

``--rest-read-timeout``
   maximum duration before timing out read of the request (default: 30s)

``--rest-write-timeout``
   maximum duration before timing out write of the response (default: 60s)

``--rest-tls-certificate``
   the certificate to use for secure connections [$STORK_REST_TLS_CERTIFICATE]

``--rest-tls-key``
   the private key to use for secure connections [$STORK_REST_TLS_PRIVATE_KEY]

``--rest-tls-ca``
   the certificate authority file to be used with mutual tls auth [$STORK_REST_TLS_CA_CERTIFICATE]

``--rest-static-files-dir``
   directory with static files for UI [$STORK_REST_STATIC_FILES_DIR]

Note there is no argument for database password, as the command line arguments can sometimes be seen
by other users. You can pass it using STORK_DATABASE_PASSWORD variable.

Mailing List and Support
~~~~~~~~~~~~~~~~~~~~~~~~~

There is a public mailing list available for the Stork project. **stork-dev**
(stork-dev at lists.isc.org) is intended for Kea developers, prospective
contributors, and other advanced users. The list is available at
https://lists.isc.org. The community provides best-effort support
on both of those lists.

Once stork will become more mature, ISC will be providing professional support
for Stork services.

History
~~~~~~~

The ``stork-server`` was first coded in November 2019 by Michal
Nowikowski and Marcin Siodelski.

See Also
~~~~~~~~

:manpage:`stork-agent(8)`
