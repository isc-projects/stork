* 17 [func] marcin

    Implemented a mechanism by which it is possible to send a command
    from the Stork server to Kea via Stork Agent and Kea Control
    Agent.
    (Gitlab #109)

Stork 0.2.0 released on 2019-12-04.

* 16 [bug] marcin

    Fixed an issue with closing a tab on the user management page.
    (Gitlab #100)

* 15 [doc] tomek

    Users and machines management is now documented in the Stork ARM.
    (Gitlab #99)

* 14 [doc] sgoldlust

    Introduced new Stork logo in the documentation.
    (Gitlab #95)

* 13 [build] tomek

    Extended the build system to be able to run on MacOS. Also updated
    installation instructions regarding how to build and run Stork natively.
    (Gitlab #87)

* 12 [func] marcin

    Enabled creation and editing of Stork user accounts in the UI.
    (Gitlab #25)

* 11 [func] marcin

    Stork server automatically migrates the database schema to the latest
    version upon startup.
    (Gitlab #33)

Stork 0.1.0 released on 2019-11-06.

* 10 [doc] marcin

    Updated ARM with a description how to sign in to the system using the
    default administrator account.
    (Gitlab #84)

* 9 [doc] tomek

    Initial ARM version added.
    (Gitlab #27)

* 8 [func] marcin

    Enabled sign-in/sign-out mechanism with HTTP sessions based on cookies.
    The default admin account has been created with default credentials.
    (Gitlab #22)

* 7 [func] godfryd

    Added initial implementation of the page which allows for adding new machines
    and listing them. The missing part of this implementation is the actual storage
    of the machines in the database. In addition, the agent has been extended to
    return a state of the machine.
    (Gitlab #23)

* 6 [func] godfryd

    Added initial implementation of Stork Agent. Implemented basic communication
    between Stork Agent and Stork Server using gRPC (Server initiates connection
    to Agent).
    (Gitlab #26)

* 5 [func] marcin

    Added stork-db-migrate tool to be used for migrating the database
    schema between versions and returning the current schema version
    number. Also, added basic schema with SQL tables holding system
    users and session information.
    (Gitlab #20)

* 4 [doc] tomek

    Added several text files: AUTHORS (lists project authors and contributors), ChangeLog.md
    (contains all new user visible changes) and CONTRIBUTING.md (Contributor's guide, explains how
    to get your patches accepted in Stork project in a seamless and easy way.
    (Gitlab #17)

* 3 [func] godfryd

   Added Swagger-based API for defining ReST API to Stork server.
   Added initial Web UI based on Angular and PrimeNG. Added Rakefile
   for building whole solution. Removed gin-gonic dependency.
   (Gitlab #19)

* 2 [build] godfryd

   Added initial framework for backend, using go and gin-gonic.
   (Gitlab #missing)

* 1 [func] franek

   Added initial proposal for Grafana dashboard.
   (Gitlab #6)


For complete code revision history, see
	http://gitlab.isc.org/isc-projects/stork

LEGEND
* [bug]   General bug fix.  This is generally a backward compatible change,
          unless it's deemed to be impossible or very hard to keep
	      compatibility to fix the bug.
* [build] Compilation and installation infrastructure change.
* [doc]   Update to documentation. This shouldn't change run time behavior.
* [func]  new feature.  In some cases this may be a backward incompatible
	      change, which would require a bump of major version.
* [sec]   Security hole fix. This is no different than a general bug
          fix except that it will be handled as confidential and will cause
 	      security patch releases.
* [perf]  Performance related change.

*: Backward incompatible or operational change.
