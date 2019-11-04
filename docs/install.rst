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

The following command will retrieve all required software (go, goswagger, nodejs, Angular
dependencies, etc.) to local directory. No root password necessary.

.. code-block:: console

    # Prepare docker images and start them up
    rake docker_up

Once the build process finishes, Stork UI will be available at http://localhost:8080/. Use
any browser to connect.

.. note::

   The installation procedure will create 3 Docker images: `stork_webui`, `stork_server` and `postgres`.
   If you run unit-tests, also `stork-ui-pgsql` image will be created. The installation
   procedure assumes those images are fully under Stork control. If there are existing images,
   they will be overwritten.

There are several other rake targets. For a complete list of available tasks, use `rake -T`.
Also see `wiki <https://gitlab.isc.org/isc-projects/stork/wikis/Development-Environment#building-testing-and-running-stork>`_
for detailed build instructions.
