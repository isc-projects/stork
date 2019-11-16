.. _installation:

******************
Stork Installation
******************

Stork is in its very early stages of development. As such, it is currently only supported on Ubuntu
18.04. It is likely that the code would work on many other systems, but for the time being we want
to focus on the core development, rather than portability issues.

There are several dependencies that needs to be installed:

 - rake
 - Java Runtime Environment
 - Docker and Docker Compose

For details, please see Stork wiki
https://gitlab.isc.org/isc-projects/stork/wikis/Development-Environment .
Note the Stork project is in very early stages and its building
instructions change frequently. Please refer to the wiki page in case
of problems.

For ease of deployment, Stork uses Rake to automate compilation and installation. It currently
requires Docker, but soon it will be optional. Technically, you can see how all Stork elements are
built and conduct all of those steps manually (without using docker).

Installation using Docker
=========================

The following command will retrieve all required software (go, goswagger, nodejs, Angular
dependencies, etc.) to local directory. No root password necessary.

.. code-block:: console

    # Prepare docker images and start them up
    rake docker_up

Once the build process finishes, Stork UI will be available at http://localhost:8080/. Use
any browser to connect.

.. note::

   The installation procedure will create 3 Docker images: `stork_webui`, `stork_server` and `postgres`.
   The PostgreSQL database schema will be automatically migrated to the latest version required
   by the Stork server process.

   If you run unit-tests, also `stork-ui-pgsql` image will be created. The installation
   procedure assumes those images are fully under Stork control. If there are existing images,
   they will be overwritten.

There are several other rake targets. For a complete list of available tasks, use `rake -T`.
Also see `wiki <https://gitlab.isc.org/isc-projects/stork/wikis/Development-Environment#building-testing-and-running-stork>`_
for detailed build instructions.

Native Installation
===================

The following steps will install Stork and its dependencies natively, i.e. on the host machine
rather than using Docker images.

First, you need to install PostgreSQL. This is OS specific. Please follow up the instructions for your
system.

.. code-block:: console

    $ psql postgres
    psql (11.5)
    Type "help" for help.

    postgres=# CREATE USER stork WITH PASSWORD 'stork';
    CREATE ROLE
    postgres=# CREATE DATABASE stork;
    CREATE DATABASE
    postgres=# GRANT ALL PRIVILEGES ON DATABASE stork TO stork;
    GRANT
    postgres=# \c stork
    You are now connected to database "stork" as user "thomson".
    stork=# create extension pgcrypto;
    CREATE EXTENSION

Now you need to build the migrations and use it to initialize and upgrade the DB to the latest schema:

.. code-block:: console

    $ rake build_migrations
    $ backend/cmd/stork-db-migrate/stork-db-migrate init
    $ backend/cmd/stork-db-migrate/stork-db-migrate up

Now that you have the database environment set up, the next step is to build all the tools. Note the first
command will download some missing dependencies needed and will install it in a local directory. This is
done only once and is not needed for future rebuilds. However, it's safe to rerun the command.

.. code-block:: console

    $ rake prepare_env
    $ rake build_agent
    $ rake build_backend
    $ rake build_server
    $ rake build_ui

The environment should be ready to run! Open 3 consoles, and run the following 3 commands, one in each
console:

.. code-block:: console

    $ rake run_server
    $ rake serve_ui
    $ rake run_agent

Once all three processes are running, go ahead and connect to http://localhost:4200 with your web browser.
