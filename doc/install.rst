.. _installation:

******************
Stork Installation
******************

Stork is in its very early stages of development. As such, it is currently only
supported on Ubuntu 18.04. It is likely that the code would work on many other
systems, but for the time being we want to focus on the core development, rather
than portability issues.

Backend installation
====================

.. code-block:: console

    # Set up dependencies
    go get github.com/gin-gonic/gin
    
    # Get the stork code
    go get gitlab.isc.org/isc-projects/stork
    
    # Optional: If testing a branch, do the following:
    cd $GOPATH/src/gitlab.isc.org/isc-projects/stork
    git checkout name-of-the-branch
    
    # run stork server
    go run gitlab.isc.org/isc-projects/stork/backend

