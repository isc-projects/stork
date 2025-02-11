.. _demo:

Demo
====

A sample installation of Stork can be used to demonstrate its
capabilities, and can also be used for its development.

The demo installation uses Docker and Docker Compose to set up all
Stork services. It contains:

- Stork Server
- Stork Server LDAP Hook
- Stork Agent with Kea DHCPv4
- Stork Agent with Kea DHCPv6
- Stork Agent with Kea HA-1 (High Availability Server 1)
- Stork Agent with Kea HA-2 (High Availability Server 2)
- Stork Agent with Kea HA-3 (High Availability Server 3)
- Stork Agent with Kea Using Many Subnets
- Stork Agent with BIND 9
- Stork Agent with BIND 9-2
- Stork Agent with PowerDNS
- Stork Environment Simulator
- PostgreSQL Database
- Prometheus & Grafana
- OpenLDAP server

The demo setup can optionally launch additional containers:

- Stork Agent with Kea Premium Features-1
- Stork Agent with Kea Premium Features-2

In the sections below, we explain how to start the demo setup
and how to launch the containers with additional Kea features, provided by hooks.
These hooks are required to manage hosts, subnets, and shared networks from within Stork.

The demo launch instructions and its features are also described on the
`ISC GitLab Wiki <https://gitlab.isc.org/isc-projects/stork/-/wikis/Demo>`_.

ISC hosts a public Stork demo available at https://demo.stork.isc.org. This demo
instance can be freely used to start experimenting with Stork. It contains Kea instances
with the additional hooks installed, so the demo can be used to try managing hosts, subnets, and shared
networks with Stork.

.. note::

   The demo setup at https://demo.stork.isc.org is restarted every hour on the hour.
   All changes in the demo over the previous hour (e.g., machine authorizations,
   Kea configuration changes, Stork settings changes, etc.) are lost and the defaults
   are restored. It may take several minutes to restart the demo, causing its
   unavailability around the top of each hour.

Requirements
------------

Running the Stork demo requires the same dependencies as building
Stork, which are described in the :ref:`installation_sources` chapter.
Besides these dependencies, the demo requires
`Docker Compose <https://docs.docker.com/compose/>`_. See the `Development Environment
<https://gitlab.isc.org/isc-projects/stork/-/wikis/Processes/development-Environment>`_
page on the Stork Wiki for more details.

Setup Steps
-----------

The following command retrieves all required software (Go, go-swagger,
Node.js, Angular dependencies, etc.) to the local directory. No root
password is necessary. It then prepares Docker images and starts them.

.. code-block:: console

   $ rake demo:up

Once the build process finishes, the Stork UI is available at
http://localhost:8080/. Use any browser to connect.

Alternatively, the ``stork-demo.sh`` script can be used to start the demo without
the Ruby toolkit:

.. code-block:: console

   $ ./stork-demo.sh

Additional Features
~~~~~~~~~~~~~~~~~~~

It is possible to run the demo with additional features enabled in the Kea
app; this requires the demo to be started with an access token for the Kea premium
and/or subscriber hook repositories. Access tokens are provided to ISC's paid support customers;
format-specific setup instructions can be found on
https://kb.isc.org/docs/isc-kea-packages#private-repositories-for-commercially-licensed-hooks. ISC paid support
customers should feel free to open a ticket for assistance if needed.

.. code-block:: console

   $ rake demo:up CS_REPO_ACCESS_TOKEN=<access token>

Detached Mode
~~~~~~~~~~~~~

It is possible to start the demo in detached mode. In that case, it
does not depend on the terminal and runs in the background until the
``rake demo:down`` call. To enable detached mode, specify the
DETACH variable set to ``true``.

.. code-block:: console

   $ rake demo:up DETACH=true

Demo Containers
---------------

The setup procedure creates several Docker containers. Their definition
is stored in the ``docker-compose.yaml`` file in the Stork source code repository.

These containers have Stork production services and components:

server
   This container is essential. It runs ``stork-server``,
   which interacts with all the agents and the database and exposes the
   API. Without it, Stork is not able to function.
webui
   This container is essential in most circumstances. It
   provides the front-end web interface. It is potentially unnecessary with
   the custom development of a Stork API client. The content is served by NGINX.
webui-apache
   This container is similar to the previous one, except Apache serves it, and
   the web UI is available under the ``/stork`` sub-directory, port 8081.
agent-bind9
   This container runs a BIND 9 server. With this container, the agent
   can be added as a machine and Stork begins monitoring its BIND
   9 service.
agent-bind9-2
   This container also runs a BIND 9 server, for the purpose of
   experimenting with two different DNS servers.
agent-pdns
   This container runs a PowerDNS server. With this container, the agent
   can be added as a machine and Stork begins monitoring its PowerDNS
   service.
agent-kea
   This container runs a Kea DHCPv4 server. With this container, the
   agent can be added as a machine and Stork begins monitoring its
   Kea DHCPv4 service.
agent-kea6
   This container runs a Kea DHCPv6 server.
agent-kea-ha1 and agent-kea-ha2
   These two containers should, in general, be run together. They each
   have a Kea DHCPv4 server instance configured in a High Availability (HA) pair. With
   both instances running and registered as machines in Stork, users can observe
   certain HA mechanisms, such as one partner taking over the traffic if the
   other partner becomes unavailable.
agent-kea-many-subnets
   This container runs an agent with a Kea DHCPv4 server that has many (nearly
   7000) subnets defined in its configuration.
agent-kea-premium-one and agent-kea-premium-two
   These containers run agents with Kea DHCPv4 and DHCPv6 servers connected
   to a MySQL database containing host reservations. They are only available when
   additional features have been enabled during the demo build.

These are containers with third-party services that are required by Stork:

postgres
   This container is essential. It runs the PostgreSQL database that
   is used by ``stork-server`` and the Kea containers. Without it,
   ``stork-server`` produces error messages about an unavailable database.
prometheus
   Prometheus, a monitoring solution (https://prometheus.io/), uses this
   container to monitor applications. It is preconfigured
   to monitor the Kea and BIND 9 containers.
grafana
   This is a container with Grafana (https://grafana.com/), a
   dashboard for Prometheus. It is preconfigured to pull data from a
   Prometheus container and show Stork dashboards.
mariadb
   This container is essential. It runs the MariaDB database that
   is used by the Kea containers.
openldap
   This container provides an LDAP server - an authentication service
   (https://www.openldap.org/). It is preconfigured with a set of users:
   ``admin`` (password: ``admin``), ``user1``, ``user2``, ``user3`` (passwords:
   ``password1``, ``password2``,  ``password3``) and groups: ``stork-admins``,
   ``stork-super-admins``.

There is also a supporting container:

simulator
   Stork Environment Simulator is a web application that can run DHCP
   traffic using ``perfdhcp`` (useful to observe non-zero statistics
   coming from Kea), run DNS traffic using ``dig`` and ``flamethrower``
   (useful to observe non-zero statistics coming from BIND 9), and
   start and stop any service in any other container (useful to
   simulate, for example, a Kea crash).
dns-proxy-server
   This is used only when the Stork agent from a container connects to a locally running
   server. The Kea/BIND 9 containers use internal Docker hostnames that the host
   cannot resolve. To translate Docker hostnames to valid IP addresses, the demo
   launches a DNS proxy in the background.

.. note::

   The containers running the Kea and BIND 9 applications are for demonstration
   purposes only. They allow users to quickly start experimenting with
   Stork without having to manually deploy Kea and/or BIND 9
   instances.

The PostgreSQL database schema is automatically migrated to the latest
version required by the ``stork-server`` process.

The setup procedure assumes those images are fully under Stork's
control. Any existing images are overwritten.

Initialization
--------------

``stork-server`` requires some initial information:

#. Go to http://localhost:8080/machines/unauthorized.
#. Authorize the machines by clicking ``Action -> Authorize`` for the desired machines.

Stork Environment Simulator
---------------------------

The Stork Environment Simulator demonstrates how Stork:

- sends DHCP traffic to Kea applications
- sends DNS requests to BIND 9 applications
- stops and starts Stork agents and the Kea and BIND 9 daemons

The Stork Environment Simulator allows DHCP traffic to be sent to selected
subnets pre-configured in Kea instances, with a limitation: it is
possible to send traffic to only one subnet from a given shared
network.

The Stork Environment Simulator also allows demonstration DNS traffic to
be sent to selected DNS servers.

The Stork Environment Simulator can add all the machines available in the
demo setup. It can stop and start selected Stork agents and the Kea and
BIND 9 applications. This is useful to simulate communication problems
between applications, Stork agents, and the Stork server.

The Stork Environment Simulator can be found at port 5010 when the demo is
running.

Prometheus
----------

The Prometheus instance is preconfigured in the Stork demo and pulls statistics from:

- the node exporters: ``agent-kea:9100``, ``agent-bind9:9100``, ``agent-bind9:9100``
- the Kea exporters embedded in ``stork-agent``: ``agent-kea:9547``,
  ``agent-kea6:9547``, ``agent-kea-ha1:9547``, ``agent-kea-ha2:9547``
- the BIND exporters embedded in ``stork-agent``: ``agent-bind9:9119``,
  ``agent-bind9-2:9119``

The Prometheus web page can be found at: http://localhost:9090/ .

Grafana
-------

The Grafana instance is also preconfigured in the Stork demo. It pulls data from
Prometheus and loads dashboards from the Stork repository, in the
Grafana folder.

The Grafana web page can be found at: http://localhost:3000/ .

Login Page Welcome Message
--------------------------

The :ref:`configuring-deployment-specific-views` section describes how to set up
a custom welcome message on the login page. These instructions can be adapted
to deploy a welcome message in the Stork server demo container, but the
copied HTML file is automatically removed from the container when the demo is
restarted each hour. Therefore, a better approach is to create the ``login-page-welcome.html``
file in the Stork source tree (i.e., ``webui/src/assets/static-page-content/login-page-welcome.html``).
This file will be automatically copied to the Stork server container when the
demo is started.
