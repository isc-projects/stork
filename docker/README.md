#### Example of how to build and push image to GitLab

Automatic method:

Use ``rake push:*`` tasks. See ``ci-images.rst`` for reference.

Manual method:

```console
$ docker login registry.gitlab.isc.org
$ docker build --no-cache -f ./docker-ci-base.txt -t registry.gitlab.isc.org/isc-projects/stork/ci-base .
$ docker push registry.gitlab.isc.org/isc-projects/stork/ci-base
```
