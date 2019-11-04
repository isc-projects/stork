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

   The installation procedure will create 3 Docker images: stork_webui, stork_server and postgres.
   If you run unit-tests, also `stork-ui-pgsql` image will be created. The installation
   procedure assumes those images are fully under Stork control. If there are existing images,
   they will be overwritten.

There are several other rake targets. To get an up to date list, use `rake --list`. You will get
a list similar to the following:

.. code-block:: console

    $ rake --tasks
    rake build_agent          # Compile agent part
    rake build_backend        # Compile whole backend: server, migrations and agent
    rake build_migrations     # Compile database migrations tool
    rake build_server         # Compile server part
    rake build_ui             # Build angular application
    rake clean                # Remove tools and other build or generated files
    rake docker_down          # Shut down all containers
    rake docker_up            # Build containers with everything and statup all services using docker-compose
    rake docs                 # Builds Stork documentation, using Sphinx
    rake gen_agent            # Generate API sources from agent.proto
    rake gen_client           # Generate client part of REST API using swagger_codegen based on swagger.yml
    rake gen_server           # Generate server part of REST API using goswagger based on swagger.yml
    rake lint_go              # Check backend source code
    rake lint_ui              # Check frontend source code
    rake prepare_env          # Download all dependencies
    rake run_server           # Run server
    rake serve_ui             # Serve angular app
    rake unittest_backend     # Run backend unit tests
    rake unittest_backend_db  # Run backend unit tests with local postgres docker container
