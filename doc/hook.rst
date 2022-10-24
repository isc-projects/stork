.. _hook:

*************
Hooks's Guide
*************

The hook is an additional file (plugin) that extends the standard Stork
functionalities. It contains functions that are called during handling of
various operations and can change the typical flow or run parallel. Independent
developers may create the hooks and enhance the Stork applications with new,
optional features.

The basis of the Stork hook solution has been discussed in
`this design <https://gitlab.isc.org/isc-projects/stork/-/wikis/designs/Hooks>`_.

How to use the hooks?
=====================

The hooks are distributed as binary files with the ``.so`` extension. These
files must be placed in the hook directory. The default location is
``/var/lib/stork-agent/hooks`` for Stork agent and
``/var/lib/stork-server/hooks`` for Stork server. You can change it using
the ``--hook-directory`` CLI option or setting the
``STORK_AGENT_HOOK_DIRECTORY`` or ``STORK_Server_HOOK_DIRECTORY`` environment
variable.

All the hooks must be compiled for the used Stork application (agent or server)
and its exact version. If the hook directory contains non-hook files or
out-of-date hooks, then Stork will not run.

End-user troubleshooting
========================

1. Directory contains non-hook files
2. Directory contains out-of-date hook (wrong version)
3. Directory contains Go plugin but no hook
4. Directory contains hook for another application
5. Directory doesn't exist
6. Directory is not readable
7. Hook doesn't contain required symbol
8. Hook was compiled with different interfaces than core
9. Directory is a file

Hook To Do list
===============

The list of hook-related features that should be implemented:

- ☑ Version check
- ☑ Application check
- ☑ Static type checking of callout points
- ☑ Calling the callout points through a proxy to hide the number of registered hooks from core
- ☑ Loading and validating the hooks
- ☑ Customizable hook location
- ☑ Calling hooks sequentially
- ☑ Calling single hook
- ☑ Separating callouts to minimize the dependencies footprint
- ☑ Possibility to use Stork-specific data types
- ☑ Exchange data via context
- ☑ Receive output from hook
- ☑ Task to initialize new hook
- ☑ Task to list dependencies of a given callout
- ☑ Task to change Stork core dependency location in go.mod
- ☑ Develop example hook
- ☐ Documentation
- ☐ Handle hook CLI parameters
- ☐ Handle hook environment variables by dedicated component
- ☐ Exchange data between hooks (excluding context)
- ☐ Demo with hooks
- ☐ Unify the hook and core toolkits and standards (for ISC hooks, e.g., linting, unit testing)
- ☐ Support UI hooks
- ☐ Tool for inspecting hooks
    - ☑ Check if file is Go plugin
    - ☑ Check if file has the mandatory functions
    - ☐ Check if file returns a proper callout object
- ☐ Specify good practices to minimize output size of binary
- ☐ Configure CI for hooks
- ☐ Distributing the compiled hooks in release
- ☐ Reload and unload hooks
- ☐ Measure hook execution time, CPU and memory usage
- ☐ Hook monitoring
- ☐ Allow storing hook settings in the database
- ☐ Hook RESTApi endpoint
- ☐ Test hooks on various operating systems

and adding more and more callout points.

Glossary
========

plugin
    Golang binary compiled with the ``plugin`` flag. It provides a variety of
    symbols (constants, interfaces, structs, variables, functions, objects) that
    may be extracted in the runtime. The plugin dependencies are static-linked
    (built-in into the binary). If the plugin and the main application share the
    same dependency, then its version must be the same in both projects. They
    must be compiled using the same Golang version too. The plugin doesn't need
    to implement any specific interface.

library
    The compatible plugin. It was compiled using the same Golang version as the
    target application, and all common dependencies match. The library doesn't
    need to implement any specific interface, but it's available to lookup for
    symbols.

hook
    The library that provides symbols required by the hook specification - the
    ``Load`` and ``Version`` functions. The ``Load`` function is used to create
    the callout object. The hook shouldn't use any global variables (except
    constants). It should be possible to call the ``Load`` and close the callout
    object multiple times without side effects. The hooks are loaded in the
    lexicographic order. Only the hooks with the compatible application name
    and Stork version returned by the ``Version`` function are loaded.

core application
    The application that loads and uses the hooks.

callout (interface)
    The interface that specifies the callout points for a given hook. The
    callout interface allows sharing of the callout points signatures between
    the core application and the hooks. The single hook may implement one or
    more callout interfaces. All callout interfaces supported by the given
    application are implemented by the hook manager and registered in the hook
    executor.

callout (structure)
    The structure defined in the hook implements the callout interfaces. 
    The structure isn't directly available for the core application, but the
    instance of it is created by the ``Load`` hook function.

callout (object)
    The instance of the callout structure created by the ``Load`` hook function.
    It allows calling the callout points implementations. The instance
    shouldn't be created before the ``Load`` call. At shutdown, the ``Close``
    method of the object is called. It should free all used resources.

callouts
    Multiple callout objects.

callout point
    A single function defined by the callout interface. It is dedicated to
    being called at a specific moment of the Stork execution. The hook manager
    specifies the exact order of calling the callout points from different
    hooks. The hook executor calls the callout points. A single callout
    interface may define one or many callout points. Due to technical reasons,
    every callout point should return a non-void value. The callout structure
    may contain other functions for internal purposes, not only callout points.

hook executor
    It is responsible for manage callout instances and execute the callout
    points.

hook manager
    The facade for calling the callout points. The specialized structs are
    created in the core applications by implementing the callout interfaces.
    It is responsible for defining the execution order of the callout points
    from the loaded hooks by calling specific methods of the hook executor.

library manager
    The wrapper for the library allows calling the functions defined by the
    hook specification. The library manager instance may be created from any
    compatible plugin (library).

Hook structure
==============

Stork hook is a Go plugin that contains fallowing symbols:

- ``Load`` function that accepts no arguments (yet?) and returns the callout
  object or error.
- ``Version`` function that accepts no arguments and returns the target 
  application name and version string.

The callout object must implement the ``io.Closer`` interface and should
implement one or more callout interfaces.

Hook development
================

1. Init
2. Repository
3. Build
4. Lint&test
5. Remap
6. Size&dependencies
7. Other tools

Steps to implement hook
=======================

1. Look for needed callout points in the hook module

    .. code-block:: go

        type Foo interface {
            int Foo(x int)
        }

2. Prepare a structure that will implement the callouts

    .. code-block:: go

        type callouts struct {}

3. Write interface checks to ensure that the callouts will have a correct signature. It would cause compilation errors if the callout point changed.

    .. code-block:: go

        var _ hooks.Foo = (*callouts)(nil)

4. Implement callout function

    .. code-block:: go

        func (c *callouts) Foo(x int) int {
            return 42
        }

5. Prepare top-level version function using the constants from the shared module

    .. code-block:: go

        func Version() (string, string) {
            return hooks.AgentName, hooks.CurrentVersion
        }

6. Prepare top-level load function

    .. code-block:: go

        func Load() (hooks.Callout, error) {
            return &callouts{}, nil
        }

7. Prepare callout close function

    .. code-block:: go

        func (c *callout) Close() error {
            return nil
        }

8. Compile to a plugin file

    .. code-block:: console
    
        $ go build -buildmode=plugin -o foo-hook.so

9. Copy the plugin file to the hook directory

    .. code-block:: console

        $ cp foo-hook.so /var/lib/stork-server/hooks

10. Run the Stork. Enjoy!
