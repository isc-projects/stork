.. _devel:

*****************
Developer's Guide
*****************

.. note::

   We acknowledge that users and developers have different needs, so
   the user and developer documents should eventually be
   separated. However, since the project is still in its early stages,
   this section is kept in the Stork ARM for convenience.

Rakefile
========

Rakefile is a script for performing many development tasks like
building source code, running linters, running unit tests, and running
Stork services directly or in Docker containers.

There are several other Rake targets. For a complete list of available
tasks, use `rake -T`.  Also see the Stork `wiki
<https://gitlab.isc.org/isc-projects/stork/wikis/Development-Environment#building-testing-and-running-stork>`_
for detailed instructions.


Generating Documentation
========================

To generate documentation, simply type ``rake doc``.
`Sphinx <http://www.sphinx-doc.org>`_ and `rtd-theme
<https://github.com/readthedocs/sphinx_rtd_theme>`_ must be installed. The
generated documentation will be available in the ``doc/singlehtml``
directory.


Setting Up the Development Environment
======================================

The following steps install Stork and its dependencies natively,
i.e. on the host machine, rather than using Docker images.

First, PostgreSQL must be installed. This is OS-specific, so please
follow the instructions from the :ref:`installation` chapter.


Once the database environment is set up, the next step is to build all
the tools. Note the first command below downloads some missing dependencies
and installs them in a local directory. This is done only once
and is not needed for future rebuilds, although it is safe to rerun
the command.

.. code-block:: console

    $ rake build_backend
    $ rake build_ui

The environment should be ready to run! Open three consoles and run
the following three commands, one in each console:

.. code-block:: console

    $ rake run_server

.. code-block:: console

    $ rake serve_ui

.. code-block:: console

    $ rake run_agent

Once all three processes are running, connect to http://localhost:8080
via a web browser. See :ref:`usage` for initial password information
or for adding new machines to the server.

The `run_agent` runs the agent directly on the current operating
system, natively; the exposed port of the agent is 8888.

There are other Rake tasks for running preconfigured agents in Docker
containers. They are exposed to the host on specific ports.

When these agents are added as machines in the ``Stork Server`` UI,
both a localhost address and a port specific to a given container must
be specified. This is a list of containers can be found in
:ref:`docker_containers_for_development` section.

Installing Git Hooks
--------------------

There is a simple git hook that inserts the issue number in the commit
message automatically; to use it, go to the ``utils`` directory and
run the ``git-hooks-install`` script. It will copy the necessary file
to the ``.git/hooks`` directory.


Agent API
=========

The connection between the server and the agents is established using
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
state, the following command can be used:

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

ReST API
========

The primary user of the ReST API is the Stork UI in a web browser. The
definition of the ReST API is located in the api folder and is
described in Swagger 2.0 format.

The description in Swagger is split into multiple files. Two files
comprise a tag group:

* \*-paths.yaml - defines URLs
* \*-defs.yaml - contains entity definitions

All these files are combined by the ``yamlinc`` tool into a single
Swagger file ``swagger.yaml``.  Then, ``swagger.yaml`` generates code
for:

* the UI fronted by swagger-codegen
* the backend in Go lang by go-swagger

All these steps are accomplished by Rakefile.

Backend Unit Tests
==================

There are unit tests for backend part (agent and server) written in Go.
They can be run using Rake:

.. code:: console

          $ rake unittest_backend

This requires preparing a database in PostgreSQL. One way to avoid
doing this manually is by using a docker container with PostgreSQL
which is automatically created when running the following Rake task:

.. code:: console

          $ rake unittest_backend_db

This one task spawns a container with PostgreSQL in the background and
then it runs unit tests. When the tests are completed the database is
shutdown and removed.

Unit Tests Database
-------------------

When docker container with a database is not used for unit tests, the
PostgreSQL server must be stared and the following role must be
created:

.. code-block:: psql

    postgres=# CREATE USER storktest WITH PASSWORD 'storktest';
    CREATE ROLE
    postgres=# ALTER ROLE storktest SUPERUSER;
    ALTER ROLE

To point unit tests to our specific database set ``POSTGRES_ADDR``
environment variable, e.g.:

.. code:: console

          $ rake unittest_backend POSTGRES_ADDR=host:port

By default it points to ``localhost:5432``.

Similarly, if the db setup requires a password other than the default
``storktest``, it's convenient to set up PGPASSWORD variable accordingly. This
can be done the following way:

.. code:: console

          $ rake unittest_backend PGPASSWORD=secret123

Note there's no need to create the storktest database itself. It is created
and destroyed by the Rakefile task.

Unit Tests Coverage
-------------------

At the end of tests execution there is coverage report presented. If
coverage of any module is below a threshold of 35% then an error is
raised.

System Tests
============

System tests for `Stork` are designed to test `Stork` in distributted environment.
They allow for testing several `Stork` servers and agents running at the same time
in one test case. They are run inside ``LXD`` containers. It is possible to set up
`Kea` or `BIND 9` services along `Stork` agents. The framework enables tinkering
in containers so custom `Kea` configs can be deployed or specific `Kea` daemons
can be stopped.

The tests can use:

- Stork server ReST API directly or
- Stork web UI via Selenium.

Dependencies
------------
System tests require:

- Linux operating system (prefered Ubuntu or Fedora)
- Python 3
- ``LXD`` containers (https://linuxcontainers.org/lxd/introduction)

LXD Installation
----------------

The easiest way to install ``LXD`` is to use ``snap``. So first let's install ``snap``.

On Fedora:

.. code-block:: console

                $ sudo dnf install snapd

On Ubuntu:

.. code-block:: console

                $ sudo apt install snapd


Then install ``LXD``:

.. code-block:: console

                $ sudo snap install lxd

And then add your user to ``lxd`` group:

.. code-block:: console

                $ sudo usermod -a -G lxd $USER

Now you need to relogin to make your presence in ``lxd`` group visible in the shell session.

After installing ``LXD`` it requires initialization. Run:

.. code-block:: console

                $ lxd init

and then for each question press **Enter** i.e. use default values::

   Would you like to use LXD clustering? (yes/no) [default=no]: **Enter**
   Do you want to configure a new storage pool? (yes/no) [default=yes]: **Enter**
   Name of the new storage pool [default=default]: **Enter**
   Name of the storage backend to use (dir, btrfs) [default=btrfs]: **Enter**
   Would you like to create a new btrfs subvolume under /var/snap/lxd/common/lxd? (yes/no) [default=yes]: **Enter**
   Would you like to connect to a MAAS server? (yes/no) [default=no]:  **Enter**
   Would you like to create a new local network bridge? (yes/no) [default=yes]:  **Enter**
   What should the new bridge be called? [default=lxdbr0]:  **Enter**
   What IPv4 address should be used? (CIDR subnet notation, "auto" or "none") [default=auto]:  **Enter**
   What IPv6 address should be used? (CIDR subnet notation, "auto" or "none") [default=auto]:  **Enter**
   Would you like LXD to be available over the network? (yes/no) [default=no]:  **Enter**
   Would you like stale cached images to be updated automatically? (yes/no) [default=yes]  **Enter**
   Would you like a YAML "lxd init" preseed to be printed? (yes/no) [default=no]:  **Enter**

More details can be found on: https://linuxcontainers.org/lxd/getting-started-cli/

One of the questions was about a subvolume. It is stored in /var/snap/lxd/common/lxd.
This subvolume is used to store images and containers. If the space is exhausted then
it is not possible to create new containers. This is not connected with you total disk
space but the space in this subvolume. To free space you may remove some stale images
or stopped containers. Basic usage of ``LXD`` is presented on:
https://linuxcontainers.org/lxd/getting-started-cli/#lxd-client

Running System Tests
--------------------

After preparing all dependencies now it is possible to start tests.
But first RPM and deb Stork packages need to be prepared. This can
be done with this Rake task:

.. code-block:: console

                $ rake build_pkgs_in_docker

When we have packages then the tests can be invoked by the following Rake task:

.. code-block:: console

                $ rake system_tests

This command beside running the tests first prepares Python virtual environment (``venv``)
where ``pytest`` and other Python dependencies are installed. ``pytest`` is a Python testing
framework that is used in Stork system tests.

At the bottom of logs there are listed test cases with their result status.

The tests can be invoked directly using ``pytest`` but first we need to change directory
to ``tests/system``:

.. code-block:: console

                $ cd tests/system
                $ ./venv/bin/pytest --tb=long -l -r ap -s tests.py

The switches passed to ``pytest`` are:

- ``--tb=long`` in case of failures present long format of traceback
- ``-l`` show values of local variables in tracebacks
- ``-r ap`` at the end of execution print report that includes (p)assed and (a)ll except passed (p)

To run particular test case add it just after ``test.py``:

.. code-block:: console

                $ ./venv/bin/pytest --tb=long -l -r ap -s tests.py::test_users_management[centos/7-ubuntu/18.04]

To get a list of tests without actually running them, the following command can be used:

.. code-block:: console

    $ ./venv/bin/pytest --collect-only tests.py

The test names of available tests will be printed as `<Function name_of_the_test>`.

Developing System Tests
-----------------------

System tests are defined in tests.py and other files that start from `test_`.
There are two other files that are defining framework for Stork system tests:

- conftest.py - it defines hooks for ``pytests``
- containers.py - it handles LXD containers: starting/stopping, communication like
  invoking commands, uploading/downloading files, installing and preparing Stork
  Agent and Server, and Kea, and other dependencies that they requires.

Most of tests are constructed as follow:

.. code-block:: python

    @pytest.mark.parametrize("agent, server", SUPPORTED_DISTROS)
    def test_machines(agent, server):
        # login to stork server
        r = server.api_post('/sessions',
                            json=dict(useremail='admin', userpassword='admin'),
                            expected_status=200)
        assert r.json()['login'] == 'admin'

        # add machine
        machine = dict(
            address=agent.mgmt_ip,
            agentPort=8080)
        r = server.api_post('/machines', json=machine, expected_status=200)
        assert r.json()['address'] == agent.mgmt_ip

        # wait for application discovery by Stork Agent
        for i in range(20):
            r = server.api_get('/machines')
            data = r.json()
            if len(data['items']) == 1 and \
               len(data['items'][0]['apps'][0]['details']['daemons']) > 1:
                break
            time.sleep(2)

        # check discovered application by Stork Agent
        m = data['items'][0]
        assert m['apps'][0]['version'] == '1.7.3'

Let's dissect this code and explain each part.


.. code-block:: python

    @pytest.mark.parametrize("agent, server", SUPPORTED_DISTROS)

This indicates that we are parametrizing the test and there will be one or more
instances of this test in execution for each set of parameters.

The constant ``SUPPORTED_DISTROS`` defines two sets of operating systems
for testing:

.. code-block:: python

    SUPPORTED_DISTROS = [
        ('ubuntu/18.04', 'centos/7'),
        ('centos/7', 'ubuntu/18.04')
    ]

The first set indicates that for Stork agent a ``Ubuntu 18.04`` should be used
in LXD container and for Stork server ``Centos 7``. The second sets is the opposite
of the first one.

The next line:

.. code-block:: python

    def test_machines(agent, server):

defines the test function. Normally agent and server argument would get text values
``'ubuntu/18.04'`` and ``'centos/7'`` but there is prepared a hook in ``conftest.py``,
in ``pytest_pyfunc_call()`` function that intercepts these arguments and based
on them it spins up LXD containers with indicated operating systems. This hook
also at the end of the test collects Stork logs from these containers and stores
them in ``test-results`` folder for later user analysis if needed.

So instead text values the hook replaces the arguments with references
to actual LXC container objects. Then the test may interact directly with them.
Beside substituting ``agent`` and ``server`` arguments the hook intercepts
any argument that starts with ``agent`` and ``server``. This way we may have
several agents in the test, e.g. ``agent1`` or ``agent_kea``, ``agent_bind9``.

Then we are logging into the Stork server using its ReST API:

.. code-block:: python

        # login to stork server
        r = server.api_post('/sessions',
                            json=dict(useremail='admin', userpassword='admin'),
                            expected_status=200)
        assert r.json()['login'] == 'admin'

And then we are adding a machine with Stork agent to Stork server:

.. code-block:: python

        # add machine
        machine = dict(
            address=agent.mgmt_ip,
            agentPort=8080)
        r = server.api_post('/machines', json=machine, expected_status=200)
        assert r.json()['address'] == agent.mgmt_ip

Here we have a check that verifies returned address of the machine.

Then we need to wait until Stork agent detects Kea application and reports it
to Stork server. We are pulling periodically the server if it received
information about Kea app.

.. code-block:: python

        # wait for application discovery by Stork Agent
        for i in range(20):
            r = server.api_get('/machines')
            data = r.json()
            if len(data['items']) == 1 and \
               len(data['items'][0]['apps'][0]['details']['daemons']) > 1:
                break
            time.sleep(2)

At the end we are verifying returned data about Kea application:

.. code-block:: python

        # check discovered application by Stork Agent
        m = data['items'][0]
        assert m['apps'][0]['version'] == '1.7.3'


.. _docker_containers_for_development:

Docker Containers for Development
=================================

To ease developemnt, there are several Docker containers available.
These containers and several more are used in Stork Demo that is
described in :ref:`Demo` chapter. The full description of each
container can found in that chapter.

The following ``Rake`` tasks are starting these containers.

.. table:: Rake tasks for managing development containers.
   :class: longtable
   :widths: 25 75

   +------------------------------------+------------------------------------------------------------+
   | Rake task                          | Description                                                |
   +====================================+============================================================+
   | ``rake build_kea_container``       | Build a container `agent-kea` with Stork Agent             |
   |                                    | and Kea with DHCPv4.                                       |
   +------------------------------------+------------------------------------------------------------+
   | ``rake run_kea_container``         | Start `agent-kea` container. Published port is 8888.       |
   +------------------------------------+------------------------------------------------------------+
   | ``rake build_kea6_container``      | Build a `agent-kea6` container with Stork Agent            |
   |                                    | and Kea with DHCPv6.                                       |
   +------------------------------------+------------------------------------------------------------+
   | ``rake run_kea6_container``        | Start `agent-kea6` container. Published port is 8886.      |
   +------------------------------------+------------------------------------------------------------+
   | ``rake build_kea_ha_containers``   | Build two containers, `agent-kea-ha1` and `agent-kea-ha2`, |
   |                                    | with Stork Agent and Kea with DHCPv4 that are configured   |
   |                                    | to work together in `High Availability` mode.              |
   +------------------------------------+------------------------------------------------------------+
   | ``rake run_kea_ha_containers``     | Start `agent-kea-ha1` and `agent-kea-ha2` containers.      |
   |                                    | Published ports are 8881 and 8882.                         |
   +------------------------------------+------------------------------------------------------------+
   | ``rake build_kea_hosts_container`` | Build a `agent-kea-hosts` container with Stork Agent       |
   |                                    | and Kea with DHCPv4 with host reservations stored in       |
   |                                    | a database. It requires **premium** features.              |
   +------------------------------------+------------------------------------------------------------+
   | ``rake run_kea_hosts_container``   | Start `agent-kea-hosts` container. It requires **premium** |
   |                                    | features.                                                  |
   +------------------------------------+------------------------------------------------------------+
   | ``rake build_bind9_container``     | Build a `agent-bind9` container with Stork Agent           |
   |                                    | and BIND 9.                                                |
   +------------------------------------+------------------------------------------------------------+
   | ``rake run_bind9_container``       | Start `agent-bind9` container. Published port is 9999.     |
   +------------------------------------+------------------------------------------------------------+


Packaging
=========

There are scripts for packaging the binary form of Stork. There are
two supported formats:

- RPM
- deb

The RPM package is built on the latest CentOS version. The deb package
is built on the latest Ubuntu LTS.

There are two packages built for each system: a server and an agent.

There are Rake tasks that perform the entire build procedure in a
Docker container: `build_rpms_in_docker` and
`build_debs_in_docker`. It is also possible to build packages directly
in the current operating system; this is provided by the `deb_agent`,
`rpm_agent`, `deb_server`, and `rpm_server` Rake tasks.

Internally, these packages are built by FPM
(https://fpm.readthedocs.io/). The containers that are used to build
packages are prebuilt with all dependencies required, using the
`build_fpm_containers` Rake task. The definitions
of these containers are placed in `docker/pkgs/centos-8.txt` and
`docker/pkgs/ubuntu-18-04.txt`.
