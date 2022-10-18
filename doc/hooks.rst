.. _hooks:

*************
Hooks's Guide
*************

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
