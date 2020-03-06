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
 - Docker and Docker Compose (if installing using Docker)

For details, please see Stork wiki
https://gitlab.isc.org/isc-projects/stork/wikis/Development-Environment .
Note the Stork project is in very early stages and its building
instructions change frequently. Please refer to the wiki page in case
of problems.

Java is currently a build-time dependency, because one of the tools used to generate API
bindings, goswagger, is written in Java. However, Java is not needed to run Stork. In the future
Stork versions, this dependency will be optional,  only necessary for developers who
want to implement new or change existing API interfaces.

For ease of deployment, Stork uses Rake to automate compilation and installation.
It facilitates installation both using Docker and without Docker (see the
following sections).

Installation Steps
==================

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

Optional step: if you want to initialize the database on your own, you need to build the migrations
and use it to initialize and upgrade the DB to the latest schema. However, this is completely
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
browser.  See  :ref:`usage` for initial password information.
