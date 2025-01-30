.. _hook:

*************
Hooks's Guide
*************

The hook is an additional file (plugin) that extends the standard Stork
functionalities. It contains functions that are called during handling of
various operations and can change the typical flow or run in parallel.
Independent developers may create the hooks and enhance the Stork applications
with new, optional features.

The basis of the Stork hook solution has been discussed in
`this design <https://gitlab.isc.org/isc-projects/stork/-/wikis/designs/Hooks>`_.

How to use the hooks?
=====================

The hooks are distributed as binary files with the ``.so`` extension. These
files must be placed in the hook directory. The default location is
``/usr/lib/stork-agent/hooks`` for Stork agent and
``/usr/lib/stork-server/hooks`` for Stork server. You can change it using
the ``--hook-directory`` CLI option or setting the
``STORK_AGENT_HOOK_DIRECTORY`` or ``STORK_SERVER_HOOK_DIRECTORY`` environment
variable.

All the hooks must be compiled for the used Stork application (agent or server)
and its exact version. If the hook directory contains non-hook files or
out-of-date hooks, then Stork will not run.

Glossary
========

plugin
    Golang binary compiled with the ``plugin`` flag. It provides a variety of
    symbols (constants, interfaces, structs, variables, functions, objects) that
    may be extracted at runtime. The plugin dependencies are static-linked
    (built-in into the binary). If the plugin and the main application share the
    same dependency, then its version must be the same in both projects. They
    must be compiled using the same Golang version too. The plugin doesn't need
    to implement any specific interface.

library
    The compatible plugin. It was compiled using the same Golang version as the
    target application, and all common dependencies match. The library doesn't
    need to implement any specific interface, but it's available to lookup for
    symbols.

hook (file)
    The library that provides symbols required by the hook specification - the
    ``Load`` and ``Version`` functions. The ``Load`` function is used to create
    the carrier object. The hook shouldn't use any global variables (except
    constants). It should be possible to call the ``Load`` and close the payload
    object multiple times without side effects. The hooks are loaded in the
    lexicographic order. Only the hooks with the compatible application name
    and Stork version returned by the ``Version`` function are loaded.

core application
    The application that loads and uses the hooks.

callout specification (interface)
    The interface that defines the callout(s) for a given hook point. The
    specification interface allows sharing of the callout signatures between
    the core application and the hooks. The single hook may implement one or
    more specification interfaces. All specification interfaces supported by
    the given application are implemented by the hook manager and registered in
    the hook executor.

callout carrier (structure)
    The structure defined in the hook implements the callout specification
    interfaces.  The structure isn't directly available for the core
    application, but the instance of it is created by the ``Load`` hook
    function.

callout carrier (object)
    The instance of the callout payload structure created by the ``Load`` hook function.
    It allows calling the callout points implementations. The instance
    shouldn't be created before the ``Load`` call. At shutdown, the ``Close``
    method of the object is called. It should free all used resources.  The 
    hook payload may contain other functions for internal purposes, not only
    callouts.

callout (function)
    A single function defined by the callout specification interface. It is
    dedicated to being called at a specific callout point. Due to technical
    reasons, every callout should return a non-void value.
    
callout point
    The point in the code at which a call callout is made. In a single callout
    point multiple callouts from various hooks may be executed by the hook
    executor. The hook manager specifies the exact order of calling the
    callouts from different hooks.

hook executor
    It is responsible for managing callout carrier instances and executing the
    callouts.

hook manager
    The facade for calling the callouts. The specialized structs are
    created in the core applications by implementing the callout specifications.
    It is responsible for defining the execution order of the callouts
    from the loaded hooks by calling specific methods of the hook executor.

library manager
    The wrapper for the library that allows calling the standard hook
    functions. The library manager instance may be created from any compatible
    plugin (library).

Hook structure
==============

Stork hook is a Go plugin that contains following symbols:

- ``Load`` function that accepts a settings object as an argument and returns
  the callout carrier or error. The settings object is created by the
  ``CreateCLIFlags`` function and its members are set by the CLI handler of the
  application. If the ``CreateCLIFlags`` is not defined, then the settings
  object is ``nil``.
- ``GetVersion`` function that accepts no arguments and returns the target 
  application name and version string.
- ``CreateCLIFlags`` function that accepts no arguments and returns a settings
  object. The object must define the CLI flags in its members' tags, following
  the `jessevdk/go-flags <https://github.com/jessevdk/go-flags>`_ library
  requirements.


The callout object must implement the ``io.Closer`` interface and should
implement one or more callout interfaces.

Hook development
================

This section describes tools and good practices helpful in hook development.

Missing parts
-------------

The hook support is still under development. The list of implemented and
planned features is available on
`the Wiki page
<https://gitlab.isc.org/isc-projects/stork/-/wikis/Hook-To-Do-List>`_

Initialization
--------------

Stork provides the ``hook:init`` Rake task. It creates a directory with the
hook project, implementations of required hook functions, a stub of the
callout structure, and initializes the git repository. It includes the Rakefile
with some basic tasks (it isn't mandatory to use them but recommended).

.. code-block:: shell

    $ rake hook:init

Repository
----------

We recommend keeping each hook in a separate git repository. The ``go.mod`` file
stored in a public repository should define Stork core dependency using tag
(explicit version) or commit hash. It shouldn't use the relative path, except
when the git submodule with Stork core is used.

Build
-----

The standard Go plugin can be compiled using the below command executed in the
main directory (the directory containing the ``go.mod`` file):

.. code-block:: shell

    $ go build -buildmode=plugin

or

.. code-block:: shell

    $ rake build

Golang requires the plugins to be built with the same flags as the core
application. Stork doesn't use any custom flags, but it may be compiled in
debug mode. The standard DLV flag set is used in this case:
``-gcflags "all=-N -l"``. The command to compile the plugins in debug mode is:

.. code-block:: shell

    $ go build -buildmode=plugin -gcflags "all=-N -l"

or

.. code-block:: shell

    $ rake build DEBUG=true

The ``build`` command validates and adjusts the ``go.mod`` file.  
Extending the build command for complex hooks may be necessary to support
additional build steps.

Lint & test
-----------

The default Rakefile contains the tasks for linting and unit testing the hook
source code for a more straightforward start of development.

.. code-block:: shell

    $ rake lint
    $ rake unittest

There are no mandatory quality checkers to use. The hook maintainer is free to
choose the tools that will be most helpful.

Remap core dependency version
-----------------------------

The Go supports three ways to specify the dependency revision. It may use a
version tag (most popular and recommended), commit hash, or relative
path to sources.

The version tag is the best option for sharing the code. But it has some
limitations. Developing a hook for a core revision that isn't already merged
(exists only on the feature branch) is impossible. The core dependency version
should be specified using the commit hash in this case. Sometimes, sharing the
core changes with the hook codebase by the repository is inconvenient. It may
be necessary to work with live Stork core sources, for example, during a new
callout point development or changing the hook framework. In this case, the
hook should use updated core sources without committing the changes to the
repository. A developer may achieve this behavior by specifying the relative
path to the core dependency instead of the version string.

Below are presented three forms of defining dependencies for Stork hook:

.. code-block:: go

    replace isc.org/stork => gitlab.isc.org/isc-projects/stork/backend v1.7.0

    replace isc.org/stork => gitlab.isc.org/isc-projects/stork/backend d7be54ae623fb07bafd4c9f819425b18b55cacce
    replace isc.org/stork => gitlab.isc.org/isc-projects/stork/backend v1.7.1-0.20221024100457-d7be54ae623f

    replace isc.org/stork => ../../backend

Note that the commit hash version has two forms. The first uses the complete
commit hash, and the second uses the short commit hash with the version tag and
timestamp. The first form is converted to the second one during the ``go.mod``
validation.

The Stork core provides the ``hook:remap_core`` Rake task to switch the core
dependency version in the ``go.mod`` files of hooks.

Use the ``TAG`` argument to specify the core version using a tag. If no value
is provided, the current Stork version is used.

.. code-block:: shell

    $ rake hook:remap_core TAG=
    $ rake hook:remap_core TAG=v1.7.0

Use the ``COMMIT`` argument to specify the core version using a commit hash. If
no value is provided, the hash of current commit is used.

.. code-block:: shell

    $ rake hook:remap_core COMMIT=
    $ rake hook:remap_core COMMIT=d7be54ae623fb07bafd4c9f819425b18b55cacce

Use the remap command without ``TAG`` and ``COMMIT`` arguments to specify
the core version using the relative path.

.. code-block:: shell

    $ rake hook:remap_core

Size & dependencies
-------------------

The Go plugins, as all Go binaries, are static linked. It means that any used
dependency will be built-in into the output file. It is essential to define
the callout interfaces to minimize the number of dependencies. Primarily, we
should avoid using external, third-party types in the callout point signatures.
Another good practice is placing the callout interfaces in separate packages.
The unnecessary dependencies may drastically increase the size of the output
plugin.

Stork provides a Rake task to list the dependencies of a given package (single
callout interface):

.. code-block:: shell

    $ rake hook:list_callout_deps KIND=agent CALLOUT=authenticationcallouts

The ``KIND`` means a target application of callout (``agent`` or ``server``).
The ``CALLOUT`` specifies name of the callout package.

Hook inspector
--------------

Some basic information (target application and version) can be listed using
the ``hook-inspect`` command of the Stork tool.

.. code-block:: shell

    $ stork-tool hook-inspect -p /usr/lib/stork-server/hooks

The ``-p`` or ``--path`` flag indicates the path to the hook directory or
single hook file.

Working with Hooks
------------------------

All Stork hooks must be built using the same Stork core version, compilation
flags, and directory structure. The Stork build system provides some tasks
to help with this.

The ``hook:sync`` Rake task clones all official Stork hooks into the ``hooks``
directory and checks out them on proper commits. Any third-party hooks should be
placed in the ``hooks`` directory manually.

The ``hook:lint`` Rake task runs the linter on all hooks using the rules of the
Stork core project. The ``hook:unittest`` task runs the unit tests of all hooks
by executing their internal ``unittest`` tasks.

The ``hook:build`` task compiles all hooks from the repositories located in the
hook directory using the current Stork core codebase. The compiled plugins
are copied to the hook directory.
The ``hook:build_pkg`` task creates a package with all compiled hooks.

For development purposes, the ``run:server_hooks`` task builds all hooks and
runs the Stork server with them.

Documentation
~~~~~~~~~~~~~

The documentation of the hooks from the ``hooks`` directory is embedded in the
Stork ARM when the ``build:doc`` task is executed. The documentation of the
hook must be placed in the ``doc`` directory of the hook project and be written
in the reStructuredText format. The entry point of the documentation should be
the ``index.rst`` file.

The man pages must be placed in the ``man`` directory of the hook project and
be written in the reStructuredText format. The build system expects a single
``man.8.rst`` file here. The compiled man pages are included in the Stork ARM
and the hook package.

Steps to implement hook
=======================

1. Look for needed callout specification in the hook module:

    .. code-block:: go

        type Foo interface {
            int Foo(x int)
        }

2. Prepare a structure that will implement the callouts:

    .. code-block:: go

        type calloutCarrier struct {}

3. Write interface checks to ensure that the callouts will have a correct signature. It would cause compilation errors if the callout changed.

    .. code-block:: go

        var _ hooks.Foo = (*calloutCarrier)(nil)

4. Implement callout function:

    .. code-block:: go

        func (c *calloutCarrier) Foo(x int) int {
            return 42
        }

5. Prepare top-level version function using the constants from the shared module:

    .. code-block:: go

        func GetVersion() (string, string) {
            return hooks.AgentName, hooks.CurrentVersion
        }

6. Prepare top-level load function:

    .. code-block:: go

        func Load(any) (hooks.CalloutCarrier, error) {
            return &callouts{}, nil
        }

7. Prepare callout close function:

    .. code-block:: go

        func (c *calloutCarrier) Close() error {
            return nil
        }

8. Compile to a plugin file:

    .. code-block:: console
    
        $ go build -buildmode=plugin -o foo-hook.so

9. Copy the plugin file to the hook directory:

    .. code-block:: console

        $ cp foo-hook.so /usr/lib/stork-server/hooks

10. Run the Stork. Enjoy!
