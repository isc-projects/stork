.. _demo:

Demo
====

Demo installation of Stork can be used to demonstrate Stork capabilities but can be used
for its development as well.

Demo installation is using Docker Compose for setting up all Stork services.
It contains:

- Stork Server
- Stork Agent with Kea DHCPv4
- Stork Agent with Kea DHCPv6
- Stork Agent with Kea HA-1 (high availability server 1)
- Stork Agent with Kea HA-2 (high availability server 2)
- Stork Agent with BIND 9
- Stork DHCP Traffic Simulator
- PostgreSQL database
- Prometheus & Grafana

Running all these services allows presenting many features of Stork.

Installation steps
------------------

The following command will retrieve all required software (go, goswagger, nodejs, Angular
dependencies, etc.) to your local directory. No root password necessary.

.. code-block:: console

    # Prepare docker images and start them up
    rake docker_up

Once the build process finishes, Stork UI will be available at http://localhost:8080/. Use
any browser to connect.

The installation procedure will create several Docker images:

- `stork_webui`: exposing web UI interface,
- `stork_server`: a server backend,
- `postgres`: a PostgreSQL database used by the server,
- `stork_agent-bind9`: agent with BIND 9,
- `stork_agent-kea`: agent with Kea DHCPv4 server,
- `stork_agent-kea6`: agent with Kea DHCPv6 server,
- `stork_agent-kea-ha1`: the primary Kea DHCPv4 server in High Availability mode,
- `stork_agent-kea-ha2`: the secondary Kea DHCPv4 server in High Availability mode
- `traffic-dhcp`: a web application that can run DHCP traffic using perfdhcp
- `prometheus`: Prometheus, a monitoring solution (https://prometheus.io/)
- `grafana`: Grafana, a dashboard for Prometheus (https://grafana.com/)

.. note::

   The containers running Kea and BIND 9 applications are for demo purposes only. They
   allow the users to quickly start playing with Stork without having to manually
   deploy Kea and/or BIND 9 instances.

The PostgreSQL database schema will be automatically migrated to the latest version required
by the Stork server process.

The installation procedure assumes those images are fully under Stork control. If there are
existing images, they will be overwritten.

Initialization
--------------

At the beginning some initial information needs to be added in Stork Server:

#. Go to http://localhost:8080/machines/all
#. Add new machines (leave default port):

   #. agent-kea
   #. agent-kea6
   #. agent-kea-ha1
   #. agent-kea-ha2
   #. agent-bind9

DHCP Traffic Simulator
----------------------
Traffic simulator allows sending DHCP traffic to selected subnets pre-configured
in Kea instances. There is a limitation, it is possible to send traffic to one subnet
from given shared network.

Traffic simulator can be found at: http://localhost:5000/

Prometheus
----------

Prometheus instance is preconfigured and pulls stats from:

- node exporters: agent-kea:9100, agent-bind9:9100
- kea exporters embedded in stork agent: agent-kea:9547, agent-kea6:9547, agent-kea-ha1:9547, agent-kea-ha2:9547
- bind9 exporter: agent-bind9:9119

Prometheus web page can be found at: http://localhost:9090/

Grafana
-------

Grafana instance is preconfigured as well. It pulls data from Prometheus and loads dashboards from stork repository,
from grafana folder.

Grafana web page can be found at: http://localhost:3000/
