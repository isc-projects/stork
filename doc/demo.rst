.. _demo:

Demo
====

A demo installation of ``Stork`` can be used to demonstrate ``Stork`` capabilities but can be used
for its development as well.

The demo installation uses `Docker` and `Docker Compose` to set up all `Stork` services.
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

These services allow observation of many Stork features.

Requirements
------------

Running the ``Stork Demo`` requires the same dependencies as building Stork,
which is described in the :ref:`installation_sources` chapter.

Besides the standard dependencies, the ``Stork Demo`` requires:

- Docker
- Docker Compose

For details, please see the Stork wiki
https://gitlab.isc.org/isc-projects/stork/wikis/Development-Environment.

Installation Steps
------------------

The following command retrieves all required software (go, goswagger, nodejs, Angular
dependencies, etc.) to the local directory. No root password is necessary.

.. code-block:: console

    # Prepare docker images and start them up
    rake docker_up

Once the build process finishes, the Stork UI is available at http://localhost:8080/. Use
any browser to connect.

The installation procedure creates several Docker images:

- `stork_webui`: a web UI interface,
- `stork_server`: a server backend,
- `postgres`: a PostgreSQL database used by the server,
- `stork_agent-bind9`: an agent with BIND 9,
- `stork_agent-kea`: an agent with a Kea DHCPv4 server,
- `stork_agent-kea6`: an agent with a Kea DHCPv6 server,
- `stork_agent-kea-ha1`: the primary Kea DHCPv4 server in High Availability mode,
- `stork_agent-kea-ha2`: the secondary Kea DHCPv4 server in High Availability mode,
- `traffic-dhcp`: a web application that can run DHCP traffic using perfdhcp,
- `prometheus`: Prometheus, a monitoring solution (https://prometheus.io/),
- `grafana`: Grafana, a dashboard for Prometheus (https://grafana.com/)

.. note::

   The containers running the Kea and BIND 9 applications are for demo purposes only. They
   allow users to quickly start experimenting with Stork without having to manually
   deploy Kea and/or BIND 9 instances.

The PostgreSQL database schema is automatically migrated to the latest version required
by the Stork server process.

The installation procedure assumes those images are fully under Stork control. If there are
existing images, they will be overwritten.

Premium Features
~~~~~~~~~~~~~~~~

It is possible to run demo with premium features enabled in Kea
apps. It requires starting demo with access token to Kea premium
repositories. Access token can be found on
https://cloudsmith.io/~isc/repos/kea-1-7-prv/setup/#formats-deb. The
token can be found inside this URL on that page:
``https://dl.cloudsmith.io/<access token>/isc/kea-1-7-prv/cfg/setup/bash.deb.sh``.
This web page and the token is available only to ISC employees and ISC customers.

.. code-block:: console

   $ rake docker_up cs_repo_access_token=<access token>


Initialization
--------------

At the beginning some initial information needs to be added in the ``Stork Server``:

#. Go to http://localhost:8080/machines/all
#. Add new machines (leave the default port):

   #. agent-kea
   #. agent-kea6
   #. agent-kea-ha1
   #. agent-kea-ha2
   #. agent-bind9

DHCP Traffic Simulator
----------------------
The traffic simulator allows DHCP traffic to be sent to selected subnets pre-configured
in Kea instances. There is a limitation: it is possible to send traffic to only one subnet
from a given shared network.

The traffic simulator can be found at: http://localhost:5000/

Prometheus
----------

The Prometheus instance is preconfigured and pulls statistics from:

- node exporters: agent-kea:9100, agent-bind9:9100
- kea exporters embedded in stork-agent: agent-kea:9547, agent-kea6:9547, agent-kea-ha1:9547, agent-kea-ha2:9547
- bind9 exporter: agent-bind9:9119

The Prometheus web page can be found at: http://localhost:9090/

Grafana
-------

The Grafana instance is preconfigured as well. It pulls data from Prometheus and loads dashboards from the Stork repository,
in the Grafana folder.

The Grafana web page can be found at: http://localhost:3000/
