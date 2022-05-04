#### Example of how to build and push image to GitLab

```console
$ docker login registry.gitlab.isc.org
$ docker build --no-cache -f ./docker-ci-base.txt -t registry.gitlab.isc.org/isc-projects/stork/ci-base .
$ docker push registry.gitlab.isc.org/isc-projects/stork/ci-base
```
