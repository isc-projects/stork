.. _ci-images:

*********
CI Images
*********

The GitLab CI in the Stork project is extensively used to test, build, and
release the software on multiple operating systems and architectures. Each GitLab CI
pipeline runs a new Docker container. To avoid repeatedly fetching the dependencies
required to build and test Stork (i.e. Java, Python, and Ruby), our developers have
prepared custom Docker images that include these dependencies. This limits the amount
of transferred data and speeds up the execution of the CI pipelines.

.. warning::

    The images tagged as ``latest`` are legacy. They should not be overwritten
    because their Dockerfiles were lost; they were not prepared from
    the images stored in the repository.
    Use explicit tags instead of the legacy images.

The Dockerfiles of CI images are located in the ``docker/images/ci``
directory. Rake tasks related to the CI images are defined in the
``rakelib/45_docker_registry.rake`` file.

List of CI Images
=================

Currently available images:

    - ``debian.Dockerfile`` - Debian-based image; a default base for CI tasks.
      Available for AMD64 and ARM64 architectures from the ``1`` tag. Stored in the
      registry as the ``ci-base`` image.
    - ``redhat-ubi.Dockerfile`` (old name: ``redhat-ubi8.Dockerfile`` - RedHat-based
      image. Available for AMD64 and ARM64
      architectures from the ``1`` tag. Stored in the
      registry as the ``pkgs-redhat-ubi`` image (prior tag ``5``: ````pkgs-redhat-ubi8``).
    - ``compose.Dockerfile`` - Allows using Docker-in-Docker in CI pipelines.
      Available only for AMD64 architecture. Stored in the registry as the
      ``pkgs-compose`` image.
    - ``cloudsmith.Dockerfile`` - Image for Cloudsmith CLI for release purposes.
      Available only for AMD64 architecture. Stored in the registry as the
      ``pkgs-cloudsmith`` image.
    - ``alpine.Dockerfile`` - Alpine-based image. Available for AMD64 and ARM64
      architectures. Stored in the registry as the ``pkgs-alpine`` image.

Removed images:

    - ``ubuntu-18.04.Dockerfile`` - Ubuntu-based image. It was initially used as a
      default base for CI tasks, but it was replaced by ``debian.Dockerfile``.

Deprecated images:

    The Dockerfiles for the images are missing but they are stored in the
    registry. Intended for removal.

    - ``ci-danger`` - Image for the Danger CI tool. It was replaced with the
      ``debian.Dockerfile`` (``ci-base``) image.
    - ``ci-postgres`` - Image for PostgreSQL database. It was used to perform
      backend unit tests in the CI pipeline. It was replaced by the
      official Postgres image (based on Alpine).

Update the Docker CI Images
===========================

To update the Docker CI images, follow these steps:

1. Edit the specific Dockerfile.
2. Check the next free tag number in the GitLab registry. Specify it in the
   ``TAG`` variable. Do not override existing tags (always keep the previous
   version around), and do not use the ``latest``  keyword unless absolutely
   confident. Use incremented tags.
   The tags should be consistent across all images. Assign
   the same tag to all images that are updated in the same ticket, meaning
   pick the highest tag number from the registry and increment it by
   one.

   For example: the registry contains two images A and B. The image A has the
   tag ``2`` and the image B has the tag ``1`` (because there were no changes
   in the last update). When updating the A and B images, assign
   the tag ``3`` to both of them.
   
3. Run the specific Rake task with the ``DRY_RUN`` set to ``true``:

    See below for the full list of available commands.

    .. code-block:: console

        $ rake push:debian TAG=42 DRY_RUN=true
        $ rake push:rhel TAG=42 DRY_RUN=true

4. Check whether the build was successful.
5. If the ``DRY_RUN`` was set to ``true``, the image is available locally. Call
   the below command to run the container and attach to it:

    .. code-block:: console

        $ docker run -it IMAGE_NAME:TAG
        # Example:
        $ docker run -it registry.gitlab.isc.org/isc-projects/stork/ci-base:42

6. Check if the container is working as expected.
7. If everything is OK, log into the registry.

    1. Create a new access token for the registry.

        Open `the Access Token GitLab page <https://gitlab.isc.org/-/profile/personal_access_tokens>`_
        and add a new token with a 1-day validity (recommended) and the
        ``read_registry`` and ``write_registry`` scopes. Copy the token value.

    2. Login to the registry.

        .. code-block:: bash

            docker login registry.gitlab.isc.org/isc-projects/stork
            # 1. Provide your GitLab login.
            # 2. Provide the access token from the previous step.

7. If everything is OK, set the ``DRY_RUN`` to ``false`` and run the task again.

    .. code-block:: console

        $ rake push:debian TAG=42 DRY_RUN=false
        $ rake push:rhel TAG=42 DRY_RUN=false

The newly pushed image is available in the GitLab registry.

.. note::

    An exclamation mark may appear near the image tag with the hint
    message (visible on hover) - ``Invalid tag: missing manifest digest``.
    It is caused by
    `a bug in the GitLab UI <https://gitlab.com/groups/gitlab-org/-/epics/10434>`_.

The following Rake tasks are available:

- ``rake push:debian`` - builds and pushes the image based on Debian.
- ``rake push:rhel`` - builds and pushes the image based on RHEL (RH UBI).
- ``rake push:alpine`` - builds and pushes the image based on Alpine.
- ``rake push:compose`` - builds and pushes the image based on official
  Docker image (includes docker-compose).
- ``rake push:cloudsmith`` - builds and pushes the image with the Cloudsmith tools

Changelog
=========

Below is the list of changes of CI images for particular tags.
The image names are the file names of their Dockerfiles.

**Tag: 6**

    - ``alpine.Dockerfile``:

        Introduced in the #1676 ticket to upgrade Go to 1.23.5. Python has
        been upgraded to 3.12 because the 3.11 version is no longer available
        in the Alpine repository.

**Tag: 5**

    - ``alpine.Dockerfile``:

        Introduced in the #1512 ticket to upgrade overall dependencies.
        Upgraded Go to 1.23.1, NodeJS 20, and Protoc to 24.4. Removed the FPM
        dependencies, i.e. tar.

    - ``redhat-ubi.Dockerfile``:

        Introduced in the #1512 ticket to upgrade overall dependencies.
        Upgraded Universal Base Image 9.4 and Ruby 3.

**Tag: 4**

    - ``compose.Dockerfile``:

        Introduced in #1328 ticket to add the missing ``protoc`` dependency.

        - Added: protoc 24
        - Updated: NodeJS 20 and NPM 10

    - ``alpine.Dockerfile``:

        Introduced in the #1353 ticket to provide the new Alpine 3.18 image,
        which includes the updated Go 1.22.2 package.

        - Base: ``golang:1.22-alpine3.18``
        - Froze all dependency versions
        - Updated to Ruby 3.2, Python 3.11, NPM 9.6, Make 4.4, Binutils-gold 2.40

    Other images were not changed.

**Tag: 3**

    Introduced in the #1178 ticket to add the missing ``ssh`` dependency.

    - ``debian.Dockerfile``:

        - Added: ssh

    Other images were not changed.

**Tag: 2**

    Introduced in the #689 ticket. The images were updated, including Python and
    Ruby. Introduced more images to avoid installing dependencies in the CI
    pipelines completely.

    - ``ubuntu-18-04.Dockerfile``:

        - Replaced with ``debian.Dockerfile``

    - ``debian.Dockerfile``:

        - Base: ``debian:12.1-slim``
        - Froze all dependency versions
        - Updated to Python 3.11, OpenJDK 17, Postgres client 15, Chromium 117,
          build essentials 12
        - Added Ruby 3.1

    - ``redhat-ubi8.Dockerfile``:

        - Base updated: ``redhat/ubi8:8.8``
        - Updated to Python 3.11, OpenJDK 17
        - Added: GCC 8.5
        - Set /tmp to be world-writable (``chmod +t``)

    - ``compose.Dockerfile``:

        - Base: ``docker:24`` (Alpine)
        - Added Python 3.11, OpenJDK 17, Rake 13, NodeJS 18.17, NPM 9, OpenSSL 3.1

    - ``cloudsmith.Dockerfile``:

        - Base updated: ``ubuntu:22.04``
        - Updated to Cloudsmith CLI 1.1.1, Python 11 (not frozen), Rake 13

    - ``alpine.Dockerfile``:

        - Base: ``golang:1.21-alpine3.17``
        - Added Python 3.10, OpenJDK 17, Rake 13, Ruby 3.1, NodeJS 18, GCC 12, Protoc 3.21

**Tag: 1**

    Introduced in the #893 ticket. The primary purpose of this tag was to include
    more dependencies in the images to avoid installing them by CI in every new
    pipeline. It allowed the execution to speed up and limit the amount of
    transferred data.

    - ``ubuntu-18-04.Dockerfile``:

        - Base: ``ubuntu:18.04``
        - Added Python 3.8, man, make, Postgres client, wget, chromium
        - Removed Docker, fpm
        - Refactored to single RUN directive

    - ``redhat-ubi8.Dockerfile``:

        - Base: ``redhat/ubi8:8.6``
        - Added Python 3.8, man

    - ``cloudsmith.Dockerfile``:

        - No changes

**Tag: latest**

    The legacy image based on Ubuntu 18.04. It is no longer used. It is kept in the
    registry to prevent the CI pipelines from breaking in old merge requests. The
    exact Dockerfile used to prepare the image available in the registry was never
    committed, and it is lost.

    - ``ubuntu-18-04.Dockerfile``:

        - Base ``ubuntu:18.04``

    - ``redhat-ubi8.Dockerfile``:

        - Base: ``redhat/ubi8:8.6``

    - ``cloudsmith.Dockerfile``:

        - Base: ``ubuntu:18.04``
