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
