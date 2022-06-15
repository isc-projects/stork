.. _devel:

*****************
Developer's Guide
*****************

.. note::

   ISC acknowledges that users and developers have different needs, so
   the user and developer documents should eventually be
   separated. However, since the project is still in its early stages,
   this section is kept in the Stork ARM for convenience.

Rakefile
========

Rakefile is a script for performing many development tasks, like
building source code, running linters and unit tests, and running
Stork services directly or in Docker containers.

There are several other Rake targets. For a complete list of available
tasks, use ``rake -T``. Also see the Stork `wiki
<https://gitlab.isc.org/isc-projects/stork/-/wikis/Processes/development-Environment#building-testing-and-running-stork>`_
for detailed instructions.

Generating Documentation
========================

To generate documentation, simply type ``rake build:doc``.
`Sphinx <https://www.sphinx-doc.org>`_ and `rtd-theme
<https://github.com/readthedocs/sphinx_rtd_theme>`_ must be installed. The
generated documentation will be available in the ``doc/_build``
directory.

Setting Up the Development Environment
======================================

The following steps install Stork and its dependencies natively,
i.e., on the host machine, rather than using Docker images.

First, PostgreSQL must be installed. This is OS-specific, so please
follow the instructions from the :ref:`installation` chapter.

Once the database environment is set up, the next step is to build all
the tools. The first command below downloads some missing dependencies
and installs them in a local directory. This is done only once
and is not needed for future rebuilds, although it is safe to rerun
the command.

.. code-block:: console

    $ rake build:backend
    $ rake build:ui

The environment should be ready to run. Open three consoles and run
the following three commands, one in each console:

.. code-block:: console

    $ rake run:server

.. code-block:: console

    $ rake build:ui_live

.. code-block:: console

    $ rake run:agent

Once all three processes are running, connect to http://localhost:8080
via a web browser. See :ref:`usage` for information on initial password creation
or addition of new machines to the server.

The ``run:agent`` runs the agent directly on the current operating
system, natively; the exposed port of the agent is 8888.

There are other Rake tasks for running preconfigured agents in Docker
containers. They are exposed to the host on specific ports.

When these agents are added as machines in the Stork server UI,
both a localhost address and a port specific to a given container must
be specified. The list of containers can be found in the
:ref:`docker_containers_for_development` section.

Installing Git Hooks
--------------------

There is a simple git hook that inserts the issue number in the commit
message automatically; to use it, go to the ``utils`` directory and
run the ``git-hooks-install`` script. It copies the necessary file
to the ``.git/hooks`` directory.

Agent API
=========

The connection between ``stork-server`` and the agents is established using
gRPC over http/2. The agent API definition is kept in the
``backend/api/agent.proto`` file. For debugging purposes, it is
possible to connect to the agent using the `grpcurl
<https://github.com/fullstorydev/grpcurl>`_ tool. For example, a list
of currently provided gRPC calls may be retrieved with this command:

.. code:: console

    $ grpcurl -plaintext -proto backend/api/agent.proto localhost:8888 describe
    agentapi.Agent is a service:
    service Agent {
      rpc detectServices ( .agentapi.DetectServicesReq ) returns ( .agentapi.DetectServicesRsp );
      rpc getState ( .agentapi.GetStateReq ) returns ( .agentapi.GetStateRsp );
      rpc restartKea ( .agentapi.RestartKeaReq ) returns ( .agentapi.RestartKeaRsp );
    }

Specific gRPC calls can also be made. For example, to get the machine
state, use the following command:

.. code:: console

    $ grpcurl -plaintext -proto backend/api/agent.proto localhost:8888 agentapi.Agent.getState
    {
      "agentVersion": "0.1.0",
      "hostname": "copernicus",
      "cpus": "8",
      "cpusLoad": "1.68 1.46 1.28",
      "memory": "16",
      "usedMemory": "59",
      "uptime": "2",
      "os": "darwin",
      "platform": "darwin",
      "platformFamily": "Standalone Workstation",
      "platformVersion": "10.14.6",
      "kernelVersion": "18.7.0",
      "kernelArch": "x86_64",
      "hostID": "c41337a1-0ec3-3896-a954-a1f85e849d53"
    }

RESTful API
===========

The primary user of the RESTful API is the Stork UI in a web browser. The
definition of the RESTful API is located in the ``api`` folder and is
described in Swagger 2.0 format.

The description in Swagger is split into multiple files. Two files
comprise a tag group:

* \*-paths.yaml - defines URLs
* \*-defs.yaml - contains entity definitions

All these files are combined by the ``yamlinc`` tool into a single
Swagger file, ``swagger.yaml``, which then generates code
for:

* the UI fronted by swagger-codegen
* the backend in Go lang by go-swagger

All these steps are accomplished by Rakefile.

Backend Unit Tests
==================

There are unit tests for the Stork agent and server backends, written in Go.
They can be run using Rake:

.. code:: console

          $ rake unittest:backend

This requires preparing a database in PostgreSQL. 

One way to avoid doing this manually is by using a Docker container with PostgreSQL,
which is automatically created when running the following Rake task:

.. code:: console

          $ rake unittest:backend_db

This task spawns a container with PostgreSQL in the background, which
then runs unit tests. When the tests are completed, the database is
shut down and removed.

Unit Tests Database
-------------------

When a Docker container with a database is not used for unit tests, the
PostgreSQL server must be started and the following role must be
created:

.. code-block:: psql

    postgres=# CREATE USER storktest WITH PASSWORD 'storktest';
    CREATE ROLE
    postgres=# ALTER ROLE storktest SUPERUSER;
    ALTER ROLE

To point unit tests to a specific Stork database, set the ``DB_HOST``
environment variable, e.g.:

.. code:: console

          $ rake unittest:backend DB_HOST=host:port

By default it points to ``localhost:5432``.

Similarly, if the database setup requires a password other than the default
``storktest``,  the ``DB_PASSWORD`` variable can be used by issuing
the following command:

.. code:: console

          $ rake unittest:backend DB_PASSWORD=secret123

Note that there is no need to create the ``storktest`` database itself; it is created
and destroyed by the Rakefile task.

Unit Tests Coverage
-------------------

A coverage report is presented once the tests have executed. If
coverage of any module is below a threshold of 35%, an error is
raised.

Benchmarks
----------

Benchmarks are part of backend unit tests. They are implemented using the
golang "testing" library and they test performance-sensitive parts of the
backend. Unlike unit tests, the benchmarks do not return pass/fail status.
They measure average execution time of functions and print the results to
the console.

In order to run unit tests with benchmarks, the ``BENCHMARK`` environment
variable must be specified as follows:

.. code:: console

          $ rake unittest:backend BENCHMARK=true

This command runs all unit tests and all benchmarks. Running benchmarks
without unit tests is possible using the combination of the ``BENCHMARK`` and
``TEST`` environment variables:

.. code:: console

          $ rake unittest:backend BENCHMARK=true TEST=Bench

Benchmarks are useful to test the performance of complex functions and find
bottlenecks. When working on improving the performance of a function, examining a
benchmark result before and after the changes is a good practice to ensure
that the goals of the changes are achieved.

Similarly, adding new logic to a function often causes performance
degradation, and careful examination of the benchmark result drop for that
function may drive improved efficiency of the new code.

Short Testing Mode
------------------

It is possible to filter out long-running unit tests, by setting the ``SHORT``
variable to ``true`` on the command line:

.. code:: console

          $ rake unittest:backend SHORT=true


Web UI Unit Tests
=================

Stork offers web UI tests, to take advantage of the unit tests generated automatically
by Angular. The simplest way to run these tests is by using Rake tasks:

.. code:: console

   rake unittest:ui

The tests require the Chromium (on Linux) or Chrome (on Mac) browser. The ``rake unittest:ui``
task attempts to locate the browser binary and launch it automatically. If the
browser binary is not found in the default location, the Rake task returns an
error. It is possible to set the location manually by setting the ``CHROME_BIN``
environment variable; for example:

.. code:: console

   export CHROME_BIN=/usr/local/bin/chromium-browser
   rake unittest:ui

By default, the tests launch the browser in headless mode, in which test results
and any possible errors are printed in the console. However, in some situations it
is useful to run the browser in non-headless mode because it provides debugging features
in Chrome's graphical interface. It also allows for selectively running the tests.
Run the tests in non-headless mode using the ``DEBUG`` variable appended to the ``rake``
command:

.. code:: console

   rake unittest:ui DEBUG=true

That command causes a new browser window to open; the tests run there automatically.

The tests are run in random order by default, which can make it difficult
to chase individual errors. To make debugging easier by always running the tests
in the same order, click "Debug" in the new Chrome window, then click
"Options" and unset the "run tests in random order" button. A specific test can
be run by clicking on its name.

.. code:: console

    TEST=src/app/ha-status-panel/ha-status-panel.component.spec.ts rake unittest:ui

By default, all tests are executed. To run only a specific test file,
set the "TEST" environment variable to a relative path to any ``.spec.ts``
file (relative to the project directory).

When adding a new component or service with ``ng generate component|service ...``, the Angular framework
adds a ``.spec.ts`` file with boilerplate code. In most cases, the first step in
running those tests is to add the necessary Stork imports. If in doubt, refer to the commits on
https://gitlab.isc.org/isc-projects/stork/-/merge_requests/97. There are many examples of ways to fix
failing tests.

System Tests
============

Stork system tests interact with its REST API to ensure proper server behavior,
error handling, and stable operation for malformed requests. Depending on the
test case, the system testing framework can automatically set up and run Kea
or Bind9 daemons and the Stork Agents the server will interact with during the
test. It runs these daemons inside the Docker containers.

Dependencies
------------
System tests require:

- Linux or macOS operating system (Windows and BSD were not tested)
- Python >= 3.18
- Rake (as a launcher)
- Docker
- docker-compose >= 1.28

Initial steps
-------------

A user must be a member of the ``docker`` group  to run the system tests.
The following commands create create this group and add the current user
to it on Linux.

1. Create the docker group.

.. code:: console

    $ sudo groupadd docker

2. Add your user to the ``docker`` group.

.. code:: console

    $ sudo usermod -aG docker $USER

3. Log out and log back in so that your group membership is re-evaluated.

Running System Tests
--------------------

After preparing all the dependencies, the tests can be started
using the following command:

.. code-block:: console

    $ rake systemtest

This command first prepares all necessary toolkits (except these listed above)
and configuration files. Next, it calls ``pytest``, a Python testing framework
used in Stork for executing the system tests.

Test results for individual test cases are shown at the end of the tests execution.

.. warning::

    Users should not attempt to run the system tests by directly calling pytest
    because it would bypass the step to generate the necessary configuration files.
    This step is conducted by the rake tasks.

To run a particular test case, specify its name in the TEST variable:

.. code-block:: console

    $ rake systemtest TEST=test_users_management

To list available tests without actually running them, use the following command:

.. code-block:: console

    $ rake systemtest:list



System Tests Framework Structure
--------------------------------

The system tests framework is located in the tests/system directory
that has the following structure:

- ``config`` - the configuration files for specific docker-compose services
- ``core`` - implements the system tests logic, docker-compose controller, wrappers for interacting with the services, and integration with ``pytest``
- ``openapi_client`` - an autogenerated client interacting with the Stork Server API
- ``test-results`` - logs from the last tests execution
- ``tests`` - the test cases (in the files prefixed with the ``test_``)
- ``conftest.py`` - defines hooks for ``pytest``
- ``docker-compose.yaml`` - the docker-compose services and networking

System Test Structure
---------------------

Let's consider the following test:

.. code-block:: python

    from core.wrappers import Server, Kea

    def test_search_leases(kea_service: Kea, server_service: Server):
        server_service.log_in_as_admin()
        server_service.authorize_all_machines()

        data = server_service.list_leases('192.0.2.1')
        assert data['items'][0]['ipAddress'] == '192.0.2.1'

The system tests framework runs in the background and maintains the
docker-compose services that contain different applications. It provides the
wrappers to interact with them using a domain language. They are the
high-level API and encapsulate the internals of the docker-compose and other
applications. The following line:

.. code-block:: python

    from core.wrappers import Server, Kea

imports the typings for these wrappers. Importing them is not necessary to
run the test case, but it enables the hints in IDE, which is very convenient
during the test development.

The next line:

.. code-block:: python

    def test_search_leases(kea_service: Kea, server_service: Server):

defines the test function. It uses the arguments that are handled by the ``pytest``
fixtures. There are four fixtures:

- ``kea_service`` - it starts the container with Kea daemon(s) and Stork Agent.
  If no fixture argument is specified (see later), it also runs the Stork Server
  containers and performs the Stork Agents registration.
  The default configuration is described by the ``agent-kea`` service in the
  ``docker-compose`` file.
- ``server_service`` - it starts the container with Stork Server. The default
  configuration is described by the ``server`` service in the ``docker-compose``
  file.
- ``bind9_service`` - it starts the container with the Bind9 daemon and Stork Agent.
  If not fixture argument was used (see later), it runs also the Stork Server
  containers and Agent registers. The default configuration is described by
  the ``agent-kea`` service in the ``docker-compose`` file.
- ``perfdhcp_service`` - it provides the container with the ``perfdhcp`` utility.
  The default configuration is described by the ``perfdhcp`` service in the
  ``docker-compose`` file.

If the fixture is required, the specified container is automatically built and run.
The test case is executed only when the service is operational - it means it is
started and healthy (i.e., the health check defined in the Docker image passes).
The containers are stopped and removed, and the logs are fetched after the test.

Only one container of a given kind can run in the current version of the system
tests framework.

.. code-block:: python

        server_service.log_in_as_admin()
        server_service.authorize_all_machines()

Test developers should use the methods provided by the wrappers to interact with
the services. Typical operations are already available as functions.

Use ``server_service.log_in_as_admin()`` to login as an administrator and start
the session. Subsequent requests will contain the credentials in the cookie file.

The ``server_service.authorize_all_machines()`` fetches all unauthorized
machines and authorizes them. They are returned by the function. The agent
registration is performed during the fixture preparation.

Use the ``server_service.wait_for_next_machine_states()`` to block and wait
until new machine states are fetched and returned.

The server wrapper provides functions to list, search, create, read, update, or
delete the items via the REST API without a need to manually prepare the
requests and parse the responses. For example:

.. code-block:: python

        data = server_service.list_leases('192.0.2.1')

To verify the data returned by the call above:

.. code-block:: python

        assert data['items'][0]['ipAddress'] == '192.0.2.1'


System Tests with a Custom Service
----------------------------------

Test developers should not reconfigure the docker-compose service in a test
case for the following reasons.

- It is slow - stopping and re-running the service The test case should assume
  that the environment is configured.
- It can be unstable - if a service fails to start or is not operational after
  restart; stopping one service may affect another service. Handling
  unexpected situations increases the test case duration and increases its
  complexity.
- It is hard to write and maintain - it is often needed to use regular
  expressions to modify the content of the existing files, create new files
  dynamically, and execute the custom commands inside the container. It
  requires a lot of work, is complex to audit, and is hard to debug.

The definition of the test case environment should be placed in the
``docker-compose.yaml`` file. Use the environment variables, arguments,
and volumes to configure the services. It allows for using static values
and files that are easy to read and maintain.

Three general services should be sufficient for most test cases and can be
extended in more complex scenarios.

1. ``server-base`` - the standard Stork Server. It doesn't use the TLS to
    connect to the database.
2. ``agent-kea`` - it runs a container with the Stork Agent, Kea DHCPv4, and
    Kea DHCPv6 daemons. The agent connects to Kea over IPv4, does not use the
    TLS or the Basic Auth credentials. Kea is configured to provision 3 IPv4
    and 2 IPv6 networks.
3. ``agent-bind9`` - it runs a container with the Stork Agent and Bind9 daemon.

The services can be customized using the ``extends`` keyword. The test case
should inherit the service configuration and apply suitable modifications.

.. note::

    Test cases should use absolute paths to define the volumes. The host paths
    should begin with ``$PWD`` environment variable returning the root project
    directory.

To run your test case with specific services, use the special helpers:

1. ``server_parametrize``
2. ``kea_parametrize``
3. ``bind9_parametrize``

They accept the name of the docker-compose service to use in the first argument:

.. code-block:: python

    from core.fixtures import kea_parametrize

    @kea_parametrize("agent-kea-many-subnets")
    def test_add_kea_with_many_subnets(server_service: Server, kea_service: Kea):
        pass

The Kea and Bind helpers additionally accept the ``suppress_registration``
parameter. If it is set to ``True`` the server service is not automatically
started, and the Stork Agent does not try to register.

.. code-block:: python

    from core.fixtures import kea_parametrize

    @kea_parametrize(suppress_registration=True)
    def test_kea_only_fixture(kea_service: Kea):
        pass

.. note::

   It is not supported to test Stork with different Kea or Bind9 versions.
   This feature is under construction.

Using perfdhcp to Generate Traffic
----------------------------------

The ``agent-kea`` service includes an initialized lease database. It should be
enough for most test cases. To generate some DHCP traffic, use the
``perfdhcp_service``.

.. code-block:: python

    from core.wrappers import Kea, Perfdhcp

    def test_get_kea_stats(kea_service: Kea, perfdhcp_service: Perfdhcp):
        perfdhcp_service.generate_ipv4_traffic(
            ip_address=kea_service.get_internal_ip_address("subnet_00", family=4),
            mac_prefix="00:00"
        )

        perfdhcp_service.generate_ipv6_traffic(
            interface="eth1"
        )

Please note above that an IPv4 address is used to send DHCPv4 traffic and an
interface name for the DHCPv6 traffic. There is no easy way to recognize
which Docker network is connected to which container interface.
The system tests use the ``priority`` property to ensure that the networks
are assigned to the consecutive interfaces.

.. code-block:: yaml

    networks:
      storknet:
        ipv4_address: 172.20.42.200
        priority: 1000
      subnet_00:
        ipv4_address: 172.100.42.200
        priority: 500

In the configuration above, the ``storknet`` network should be assigned
to the ``eth0`` (the first) interface, and the ``subnet_00`` network to the
``eth1`` interface. Our experiments show that this assumption works
reliably.

System Test Commands
--------------------

The following commands run the system tests and help with troubleshooting:

.. table:: Rake tasks for system testing
    :class: longtable
    :widths: 26 74

    +--------------------------------+----------------------------------------------+
    | Rake Tasks                     | Description                                  |
    +================================+==============================================+
    | ``rake systemtest``            | Runs the system tests. Use TEST variable to  |
    |                                | run a selected test.                         |
    +--------------------------------+----------------------------------------------+
    | ``rake systemtest:build``      | Build the system test containers.            |
    +--------------------------------+----------------------------------------------+
    | ``rake systemtest:down``       | Stops all system test containers and removes |
    |                                | them. It also removes all networks, and      |
    |                                | volumes.                                     |
    +--------------------------------+----------------------------------------------+
    | ``rake systemtest:gen``        | Generates the OpenAPI client and             |
    |                                | configurations.                              |
    +--------------------------------+----------------------------------------------+
    | ``rake systemtest:list``       | Lists the test cases.                        |
    +--------------------------------+----------------------------------------------+
    | ``rake systemtest:logs``       | Displays the container logs. Use the SERVICE |
    |                                | variable to get the logs only for a specific |
    |                                | service.                                     |
    +--------------------------------+----------------------------------------------+
    | ``rake systemtest:perfdhcp``   | Low-level access to the perfdhcp command in  |
    |                                | a container. The Rake-style arguments can be |
    |                                | specified to control ``perfdhcp``, e.g.:     |
    |                                | ``rake systemtest:perfdhcp[-6,-l,eth1]``.    |
    +--------------------------------+----------------------------------------------+
    | ``rake systemtest:regen``      | Re-generates the files created by            |
    |                                | ``rake systemtest:gen``.                     |
    +--------------------------------+----------------------------------------------+
    | ``rake systemtest:sh``         | Low-level access to the docker-compose with  |
    |                                | all necessary parameters. Use Rake-style     |
    |                                | arguments, e.g. ``rake systemtest:sh[ps]``   |
    +--------------------------------+----------------------------------------------+
    | ``rake systemtest:shell``      | Attaches to a shell in a container with      |
    |                                | provided name by SERVICE variable.           |
    +--------------------------------+----------------------------------------------+

Alpine troubleshooting
----------------------

The system tests can be running on the Alpine operating system. Unfortunately, the
Alpine uses the ``libc-musl`` instead of the standard ``libc``. This component
isn't well supported by the external dependencies maintainers. It causes Rake
scripts cannot install some dependencies and they need to be installed manually
from the Alpine package manager ``apk``.

To run the tests on Alpine system you need to install the ``nodejs`` package from the ``apk`` package manager:

.. code-block:: console

   $ apk add --no-cache nodejs

and set the ``USE_SYSTEM_NODEJS`` environment variable to ``true``:

.. code-block:: console

   $ rake demo:up USE_SYSTEM_NODEJS=true


.. _docker_containers_for_development:

Docker Containers for Development
=================================

To ease development, there are several Docker containers available.
These containers are used in the Stork demo and are fully
described in the :ref:`Demo` chapter.

The following ``Rake`` tasks start these containers.

.. table:: Rake tasks for managing development containers
   :class: longtable
   :widths: 26 74

   +----------------------------------------+---------------------------------------------------------------+
   | Rake Task                              | Description                                                   |
   +========================================+===============================================================+
   | ``rake demo:up:kea``                   | Build and run an ``agent-kea`` container with a Stork agent   |
   |                                        | and Kea with DHCPv4. Published port is 8888.                  |
   +----------------------------------------+---------------------------------------------------------------+
   | ``rake demo:up:kea6``                  | Build and run an ``agent-kea6`` container with a Stork agent  |
   |                                        | and Kea with DHCPv6. Published port is 8886.                  |
   +----------------------------------------+---------------------------------------------------------------+
   | ``rake demo:up:kea_ha``                | Build and run two containers, ``agent-kea-ha1`` and           |
   |                                        | ``agent-kea-ha2`` that are configured to work together in     |
   |                                        | High Availability mode, with Stork agents, and Kea DHCPv4.    |
   +----------------------------------------+---------------------------------------------------------------+
   | ``rake demo:up:kea_premium``           | Build and run an ``agent-kea-premium`` container with a Stork |
   |                                        | agent and Kea with DHCPv4 with host reservations stored in    |
   |                                        | a database. This requires **premium** features.               |
   +----------------------------------------+---------------------------------------------------------------+
   | ``rake demo:up:bind9``                 | Build and run an ``agent-bind9`` container with a Stork agent |
   |                                        | and BIND 9. Published port is 9999.                           |
   +----------------------------------------+---------------------------------------------------------------+
   | ``rake demo:up:postgres``              | Build and run a Postgres container.                           |
   +----------------------------------------+---------------------------------------------------------------+
   | ``rake demo:up``                       | Build and run all above containers                            |
   +----------------------------------------+---------------------------------------------------------------+
   | ``rake demo:down``                     | Stop and remove all containers and all referenced volumes and |
   |                                        | networks                                                      |
   +----------------------------------------+---------------------------------------------------------------+

.. note::

    It is recommended that these commands be run using a user account without
    superuser privileges, which may require some previous steps to set up. On
    most systems, adding the account to the ``docker`` group should be enough.
    On most Linux systems, this is done with:

    .. code:: console

        $ sudo usermod -aG docker ${user}

    A restart may be required for the change to take effect.

The Kea and BIND 9 containers connect to the Stork Server container by default.
It can be useful for developers to connect them to the locally running server.
You can specify the target server using the SERVER_MODE environment variable with one of the values:

- host - Do not run the server in Docker. Use the local one instead (it must be run separately on the host).
- no-server - Do not run the server.
- with-ui - Run the server in Docker with UI.
- without-ui - Run the server in Docker without UI.
- default - Use the default service configuration from the Docker compose file (default).

For example, to connect the agent from the Docker container to the locally
running Stork Server:

1. Run the Stork Server locally:

.. code-block:: console

    $ rake run:server

2. Run a specific agent service with the SERVER_MODE parameter set to ``host``:

.. code-block:: console

    $ rake demo:up:kea SERVER_MODE=host

3. Check the unauthorized machines page for a new machine

The Stork Agent containers use the Docker hostnames during communication with
Stork Server.  If you use the server running locally, located on the Docker
host, it cannot resolve the Docker hostnames. You need to explicitly provide
the hostname mapping in your ``/etc/hosts`` file to fix it.
You can use the ``rake demo:check_etchosts`` command to check your actual
``/etc/hosts`` and generate the content that needs to be appended.
This task will automatically run if you use ``SERVER_MODE=host`` then you don't
need to call it manually. It's mainly for diagnostic purposes.

Packaging
=========

There are scripts for packaging the binary form of Stork. There are
two supported formats: RPM and deb.

The package type is selected based on the OS that executes the command.
Use the ``utils:print_pkg_type`` to get the package type supported by your OS.

Use ``rake build:agent_pkg`` to build the agent package and
``rake build:server_pkg`` for server package. The package binaries are located
in the ``dist/pkgs`` directory.

Internally, these packages are built by FPM
(https://fpm.readthedocs.io/). It is installed automatically but it requires
the ``ruby-dev`` and ``make`` to build.

Implementation details
======================

Agent Registration Process
--------------------------

The diagram below shows a flowchart of the agent registration process in Stork.
It merely demonstrates the successful registration path.
The first Certificate Signing Request (CSR) is generated using an existing or new
private key and agent token.
The CSR, server token (optional), and agent token are sent to the Stork server.
A successful server response contains a signed agent certificate, a server CA
certificate, and an assigned Machine ID.
If the agent was already registered with the provided agent token, only the assigned
machine ID is returned without new certificates.
The agent uses the returned machine ID to verify that the registration was successful.

.. figure:: uml/registration-agent.*

    Agent registration
