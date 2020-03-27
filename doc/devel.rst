.. _devel:

*****************
Developer's Guide
*****************

.. note::

   We acknowledge that users and developers are two different groups of people, so the documents
   should eventually be separated. However, since these are still very early days of the project,
   this section is kept in the Stork ARM for convenience only.

Rakefile
========

Rakefile is a script for performing many development tasks like building source code, running linters,
running unit tests, running Stork services directly or in Docker containers.

There are several other rake targets. For a complete list of available tasks, use `rake -T`.
Also see `wiki <https://gitlab.isc.org/isc-projects/stork/wikis/Development-Environment#building-testing-and-running-stork>`_
for detailed instructions.


Generating Documentation
========================

To generate documentation, simply type ``rake doc``. You need to have `Sphinx <http://www.sphinx-doc.org>`_
and `rtd-theme <https://github.com/readthedocs/sphinx_rtd_theme>`_ installed. The generated
documentation will be available in the ``doc/singlehtml`` directory.


Setting up Development Environment
==================================

The following steps will install Stork and its dependencies natively, i.e. on the host machine
rather than using Docker images.

First, you need to install PostgreSQL. This is OS specific. Please follow the instructions
from :ref:`installation` chapter.

Optional step: if you want to initialize the database on your own, you need to build the migrations
tool and use it to initialize and upgrade the DB to the latest schema. However, this is completely
optional as the database migration will be triggered automatically upon the server startup.
This is only useful if for some reason you want to set up the database, but don't want to run
the server yet. In most cases this step can be skipped.

.. code-block:: console

    $ rake build_migrations
    $ backend/cmd/stork-db-migrate/stork-db-migrate init
    $ backend/cmd/stork-db-migrate/stork-db-migrate up

Now that you have the database environment set up, the next step is to build all the tools. Note the first
command will download some missing dependencies needed and will install it in a local directory. This is
done only once and is not needed for future rebuilds. However, it's safe to rerun the command.

.. code-block:: console

    $ rake build_backend
    $ rake build_ui

The environment should be ready to run! Open 3 consoles, and run the following 3 commands, one in each
console:

.. code-block:: console

    $ rake run_server
    $ rake serve_ui
    $ rake run_agent

Once all three processes are running, go ahead and connect to http://localhost:4200 with your web
browser.  See  :ref:`usage` for initial password information or for adding new machines to the server.

The `run_agent` runs the agent directly on current operating system, natively,
and exposed port of the agent is 8888.

There are other Rake tasks for running preconfigured agents in Docker containers.
They are exposed to the host on specific ports.

So when they are added as machines in ``Stork Server`` UI a localhost as addresss should be specified and
port specific to given container has to be specified. This is a list of ports for particular Rake tasks
and containers:

- `rake run_kea_container`: Kea with DHCPv4, port 8888
- `rake run_kea6_container`: Kea with DHCPv6, port 8886
- `rake run_kea_ha_containers`: 2 containers: Kea 1 and 2 with preconfigured HA, ports:  8881 and 8882
- `rake run_bind9_container`: port 9999

Agent API
=========

The connection between the server and the agents is established using gRPC over http/2. The agent API
definition is kept in the ``backend/api/agent.proto`` file. For debugging purposes, it is possible
to connect to the agent using `grpcurl <https://github.com/fullstorydev/grpcurl>`_ tool. For example,
you can retrieve a list of currently provided gRPC calls by using this command:

.. code:: console

    $ grpcurl -plaintext -proto backend/api/agent.proto localhost:8888 describe
    agentapi.Agent is a service:
    service Agent {
      rpc detectServices ( .agentapi.DetectServicesReq ) returns ( .agentapi.DetectServicesRsp );
      rpc getState ( .agentapi.GetStateReq ) returns ( .agentapi.GetStateRsp );
      rpc restartKea ( .agentapi.RestartKeaReq ) returns ( .agentapi.RestartKeaRsp );
    }

You can also make specific gRPC calls. For example, to get the machine state, the following command
can be used:

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

There's a simple git hook that inserts the issue number in the commit message automatically. If you
want to use it, go to ``utils`` directory and run ``git-hooks-install`` script. It will copy the
necessary file to ``.git/hooks`` directory.


ReST API
========

The primary user of ReST API is Stork UI in web browser. The definition of ReST API is located
in api folder and is described in Swagger 2.0 format.

The description in Swagger is split into multiple files. 2 files comprise a tag group:

* \*-paths.yaml - defines URLs
* \*-defs.yaml - contains entity definitions

All these files are combined by ``yamlinc`` tool into signle swagger file ``swagger.yaml``.
Then from ``swagger.yaml`` there are generated code for:

* UI fronted by swagger-codegen
* backend in Go lang by go-swagger

All these steps are realized by Rakefile.

Docker Containers
=================

To ease testing, there are several docker containers available. Not all of them are necessary.

* ``server`` - This container is essential. It runs the Stork server, which interacts with all the
  agents, the database and exposes API. Without it, Stork will not be able to function.
* ``postgres`` - This container is essential. It runs the PostgreSQL database that is used by the
  Stork server. Without it, Stork server will will only able to produce error messages about
  unavailable database.
* ``webui`` - This container is essential in most circumstances. It provides the front web
  interface. You could possibly not run it, if you are developing your own Stork API client.

There are also several containers provided that are used to samples. Those will not be needed in a
production network, however they're very useful to demonstrate existing Stork capabilities. They
simulate certain services that Stork is able to handle:

* ``agent-bind9`` - This container runs BIND 9 server. If you run it, you can add it as a machine
  and Stork will begin monitoring its BIND 9 service.

* ``agent-kea`` - This container runs Kea DHCPv4 server. If you run it, you can add it as a machine
  and Stork will begin monitoring its BIND 9 service.

* ``agent-kea-ha1`` and ``agent-kea-ha2`` - Those two containers should in general be run
  together. They have each a Kea DHCPv4 server instance configured in a HA pair. If you run both of
  them and register them as machines in Stork, you will be able to observe certain HA mechanisms,
  such as one taking over the traffic if the partner becomes unavailable.

* ``traffic-dhcp`` - This container is optional. If stated, it will start transmitting DHCP packets
  towards agent-kea. It may be useful to observe non-zero statistics coming from Kea. If you're
  running Stork in docker, you can coveniently control that using ``rake start_traffic_dhcp`` and

* ``prometheus`` - This is a container with Prometheus for monitoring applications.
  It is preconfigured to monitor Kea and BIND 9 containers.

* ``grafana`` - This is a container with Grafana - a dashboard for Prometheus. It is preconfigured
  to pull data from Prometheus container and show Stork dashboards.

Packaging
=========

There are scripts for packaging binary form of Stork. There are supported two formats:

- RPM
- deb

RPM package is being built on latest CentOS. deb package is being built on latest Ubuntu LTS.

There are built 2 packages for each system: a server and an agent.

There are Rake tasks that perform whole build procedure in a docker containers:
`build_rpms_in_docker` and `build_debs_in_docker`. It is also possible to build packages directly
in current operating system - this is provided by `deb_agent`, `rpm_agent`, `deb_server`, `rpm_server`
Rake tasks.

Internally these packages are being built by FPM (https://fpm.readthedocs.io/). The containers
that are being used to built packages are prebuilt with all dependencies required.
This is accomplished by `build_fpm_containers` Rake tasks. The definitions of these containers
are placed in `docker/pkgs/centos-8.txt` and `docker/pkgs/ubuntu-18-04.txt`.
