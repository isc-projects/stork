# stork

Stork is a new project proposed by ISC with the aim of delivering BIND and Kea dashboard.
It is going to be a spiritual successor of earlier attempts - Kittiwake and Antherius.

It is currently in very early stages of planning. More information will become publicly
available in October 2019. Stay tuned!

# Starting

Run:

```console
rake docker_up
```

It spins up two containers using docker-compose. One with server in Go and the other
with Nginx with UI in Angular/PrimeNG.

Access the service: http://localhost:8080/
