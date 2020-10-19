.. _demo:

Demo
====

A demo installation of ``Stork`` can be used to demonstrate ``Stork``
capabilities but can be used for its development as well.

The demo installation uses `Docker` and `Docker Compose` to set up all
`Stork` services. It contains:

- Stork Server
- Stork Agent with Kea DHCPv4
- Stork Agent with Kea DHCPv6
- Stork Agent with Kea HA-1 (high availability server 1)
- Stork Agent with Kea HA-2 (high availability server 2)
- Stork Agent with BIND 9
- Stork Environment Simulator
- PostgreSQL database
- Prometheus & Grafana

These services allow observation of many Stork features.

Requirements
------------

Running the ``Stork Demo`` requires the same dependencies as building
Stork, which is described in the :ref:`installation_sources` chapter.

Besides the standard dependencies, the ``Stork Demo`` requires:

- Docker
- Docker Compose

For details, please see the Stork wiki
https://gitlab.isc.org/isc-projects/stork/wikis/Development-Environment.

Setup Steps
-----------

The following command retrieves all required software (go, goswagger,
nodejs, Angular dependencies, etc.) to the local directory. No root
password is necessary. Then it prepares Docker images and starts them
up.

.. code-block:: console

    $ rake docker_up

Once the build process finishes, the Stork UI is available at
http://localhost:8080/. Use any browser to connect.

Premium Features
~~~~~~~~~~~~~~~~

It is possible to run the demo with premium features enabled in Kea
apps. It requires starting the demo with an access token to the Kea premium
repositories. Access tokens can be found on
https://cloudsmith.io/~isc/repos/kea-1-7-prv/setup/#formats-deb. The
token can be found inside this URL on that page:
``https://dl.cloudsmith.io/<access token>/isc/kea-1-7-prv/cfg/setup/bash.deb.sh``.
This web page and the token are available only to ISC employees and paid customers of ISC.

.. code-block:: console

   $ rake docker_up cs_repo_access_token=<access token>

Demo Containers
---------------

The setup procedure creates several Docker containers. Their definition
is stored in docker-compose.yaml file in Stork source code repository.

These containers have Stork production services and components:

server
   This container is essential. It runs the Stork server,
   which interacts with all the agents and the database and exposes the
   API. Without it, Stork will not be able to function.
webui
   This container is essential in most circumstances. It
   provides the front-end web interface. It is potentially unnecessary with
   the custom development of a Stork API client.
agent-bind9
   This container runs a BIND 9 server. With this container, the agent
   can be added as a machine and Stork will begin monitoring its BIND
   9 service.
agent-bind9-2
   This container also runs a BIND 9 server, for the purpose of
   experimenting with two different DNS servers.
agent-kea
   This container runs a Kea DHCPv4 server. With this container, the
   agent can be added as a machine and Stork will begin monitoring its
   Kea DHCPv4 service.
agent-kea6
   An agent with a Kea DHCPv6 server.
agent-kea-ha1 and agent-kea-ha2
   These two containers should, in general, be run together. They each
   have a Kea DHCPv4 server instance configured in a HA pair. With
   both running and registered as machines in Stork, users can observe
   certain HA mechanisms, such as one taking over the traffic if the
   partner becomes unavailable.
agent-kea-many-subnets
   An agent ith a Kew DHCPv4 server that has many subnets defined in
   its config (about 7000)

These are containers with 3rd party services that are required by Stork:

postgres
   This container is essential. It runs the PostgreSQL database that
   is used by the Stork server. Without it, the Stork server will
   produce error messages about an unavailable database.
prometheus
   Prometheus, a monitoring solution (https://prometheus.io/). This
   container is used for monitoring applications.  It is preconfigured
   to monitor Kea and BIND 9 containers.
grafana
   This is a container with Grafana (https://grafana.com/), a
   dashboard for Prometheus. It is preconfigured to pull data from a
   Prometheus container and show Stork dashboards.

These are supporting containers:

simulator
   Stork Environment Simulator, a web application that can run DHCP
   traffic using perfdhcp (useful to observe non-zero statistics
   coming from Kea), run DNS traffic using dig and flamethrower
   (useful to observe non-zero statistics coming from BIND 9), and
   start and stop any service in any other container (useful to
   simulate e.g. crash of Kea by stopping it).

.. note::

   The containers running the Kea and BIND 9 applications are for demo
   purposes only. They allow users to quickly start experimenting with
   Stork without having to manually deploy Kea and/or BIND 9
   instances.

The PostgreSQL database schema is automatically migrated to the latest
version required by the Stork server process.

The setup procedure assumes those images are fully under Stork
control. If there are existing images, they will be overwritten.


Initialization
--------------

``Stork Server`` requires some initial information:

#. Go to http://localhost:8080/machines/all
#. Add new machines (leave the default port):

   #. agent-kea
   #. agent-kea6
   #. agent-kea-ha1
   #. agent-kea-ha2
   #. agent-bind9
   #. agent-bind9-2

Stork Environment Simulator
---------------------------

Stork Environment Simulator allows:

- sending DHCP traffic to Kea applications
- sending DNS requests to BIND 9 applications
- stopping and starting Stork Agents, Kea and BIND 9 daemons

Stork Environment Simulator allows DHCP traffic to be sent to selected
subnets pre-configured in Kea instances, with a limitation: it is
possible to send traffic to only one subnet from a given shared
network.

Stork Environment Simulator also allows sending DNS traffic to
selected DNS servers.

Stork Environment Simulator can add all the machines available in the
demo setup. It can stop and start selected Stork Agents, and Kea and
BIND 9 applications. This is useful to simulate communication problems
between applications, Stork Agents and the Stork Server.

Stork Environment Simulator can be found at: http://localhost:5000/

For development purposes simulator can be started directly by command:

.. code-block:: console

   $ rake run_sim


Prometheus
----------

The Prometheus instance is preconfigured and pulls statistics from:

- node exporters: agent-kea:9100, agent-bind9:9100, agent-bind9:9100
- kea exporters embedded in stork-agent: agent-kea:9547,
  agent-kea6:9547, agent-kea-ha1:9547, agent-kea-ha2:9547
- bind exporters embedded in stork-agent: agent-bind9:9119,
  agent-bind9-2:9119

The Prometheus web page can be found at: http://localhost:9090/

Grafana
-------

The Grafana instance is preconfigured as well. It pulls data from
Prometheus and loads dashboards from the Stork repository, in the
Grafana folder.

The Grafana web page can be found at: http://localhost:3000/
