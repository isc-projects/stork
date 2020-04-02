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

Optional step: to initialize the database directly, the migrations
tool must be built and used to initialize and upgrade the database to the
latest schema. However, this is completely optional, as the database
migration is triggered automatically upon server startup.  This is
only useful if for some reason it is desirable to set up the database
but not yet run the server. In most cases this step can be skipped.

.. code-block:: console

    $ rake build_migrations
    $ backend/cmd/stork-db-migrate/stork-db-migrate init
    $ backend/cmd/stork-db-migrate/stork-db-migrate up

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

Once all three processes are running, connect to http://localhost:4200
via a web browser. See :ref:`usage` for initial password information
or for adding new machines to the server.

The `run_agent` runs the agent directly on the current operating
system, natively; the exposed port of the agent is 8888.

There are other Rake tasks for running preconfigured agents in Docker
containers. They are exposed to the host on specific ports.

When these agents are added as machines in the ``Stork Server`` UI,
both a localhost address and a port specific to a given container
must be specified. This is a list of ports for particular Rake tasks
and containers:

- `rake run_kea_container`: Kea with DHCPv4, port 8888
- `rake run_kea6_container`: Kea with DHCPv6, port 8886
- `rake run_kea_ha_containers` (2 containers): Kea 1 and 2 with
  preconfigured HA, ports 8881 and 8882
- `rake run_bind9_container`: port 9999

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

Installing Git Hooks
====================

There is a simple git hook that inserts the issue number in the commit
message automatically; to use it, go to the ``utils`` directory and
run the ``git-hooks-install`` script. It will copy the necessary file
to the ``.git/hooks`` directory.


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

Docker Containers
=================

To ease testing, there are several Docker containers available,
although not all of them are necessary.

* ``server`` - This container is essential. It runs the Stork server,
  which interacts with all the agents and the database and exposes the
  API. Without it, Stork will not be able to function.
* ``postgres`` - This container is essential. It runs the PostgreSQL
  database that is used by the Stork server. Without it, the Stork
  server will produce error messages about an unavailable database.
* ``webui`` - This container is essential in most circumstances. It
  provides the front-end web interface. It is potentially unnecessary with
  the custom development of a Stork API client.

There are also several containers provided that are used as
samples. They are not needed in a production network; however, they
are very useful to demonstrate existing Stork capabilities. They
simulate certain services that Stork is able to handle, including:

* ``agent-bind9`` - This container runs a BIND 9 server. With this
  container, the agent can be added as a machine and Stork will begin
  monitoring its BIND 9 service.

* ``agent-kea`` - This container runs a Kea DHCPv4 server. With this
  container, the agent can be added as a machine and Stork will begin
  monitoring its Kea DHCPv4 service.

* ``agent-kea-ha1`` and ``agent-kea-ha2`` - These two containers
  should, in general, be run together. They each have a Kea DHCPv4
  server instance configured in a HA pair. With both running and
  registered as machines in Stork, users can observe certain HA
  mechanisms, such as one taking over the traffic if the partner
  becomes unavailable.

* ``traffic-dhcp`` - This container is optional. If started, it will
  transmit DHCP packets to ``agent-kea``. It may be useful to observe
  non-zero statistics coming from Kea. When running Stork in Docker,
  ``rake start_traffic_dhcp`` can be used to conveniently control
  traffic.

* ``prometheus`` - This is a container with Prometheus for monitoring
  applications.  It is preconfigured to monitor Kea and BIND 9
  containers.

* ``grafana`` - This is a container with Grafana, a dashboard for
  Prometheus. It is preconfigured to pull data from a Prometheus
  container and show Stork dashboards.

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
