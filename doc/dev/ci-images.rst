.. _ci-images:

*********
CI images
*********


List of CI images
=================

Currently available images:

    - ``debian.Dockerfile`` - Debian-based image, it is a default base for CI task.
    Available for AMD64 and ARM64 architectures from the ``1`` tag. Stored in the
    registry as the ``ci-base`` image.
    - ``redhat-ubi8.Dockerfile`` - RedHat-based image
    Available for AMD64 and ARM64 architectures from the ``1`` tag. Stored in the
    registry as the ``pkgs-redhat-ubi8`` image.
    - ``compose.Dockerfile`` - Allows using Docker-in-Docker in CI pipelines
    Available only for AMD64 architecture. Stored in the registry as the
    ``pkgs-compose`` image.
    - ``cloudsmith.Dockerfile`` - Image for Cloudsmith CLI for the release purposes
    Available only for AMD64 architecture. Stored in the registry as the
    ``pkgs-cloudsmith`` image.
    - ``alpine.Dockerfile`` - Alpine-based image. Available for AMD64 and ARM64
    architectures. Stored in the registry as the ``pkgs-alpine`` image.

Removed images:

    - ``ubuntu-18.04.Dockerfile`` - Ubuntu-based image. It was initially used as a
    default base for CI task, but it was replaced with ``debian.Dockerfile``.

Deprecated images:

    The Dockerfiles for the images are missing but they are stored in the
    registry. Intended for removal.

    - ``ci-danger`` - Image for Danger CI tool. It was replaced with
      ``debian.Dockerfile`` (``ci-base``) image.
    - ``ci-postgresql`` - Image for PostgreSQL database. It was used to perform
        the backend unit tests in the CI pipeline. It was replaced with the
        official Postgres image (based on Alpine).

Changelog
=========

Below is the list of changes introduced in the CI images for particular tags.
The image names are the file names of their Dockerfiles.

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
        - Frozen all dependency versions
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
