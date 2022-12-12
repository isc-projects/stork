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
    more specifications interfaces. All specifications interfaces supported by
    the given application are implemented by the hook manager and registered in
    the hook executor.

callout carrier (structure)
    The structure defined in the hook implements the callout specification
    interfaces.  The structure isn't directly available for the core
    application, but the instance of it is created by the ``Load`` hook
    function.

callout carrier (object)
    The instance of the callout payload structure created by the ``Load`` hook function.
    It allows calling the callout implementations. The instance
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
    It is responsible for manage callout carrier instances and execute the
    callouts.

hook manager
    The facade for calling the callouts. The specialized structs are
    created in the core applications by implementing the callout specifications.
    It is responsible for defining the execution order of the callouts
    from the loaded hooks by calling specific methods of the hook executor.

library manager
    The wrapper for the library allows calling the standard hook functions. The
    library manager instance may be created from any compatible plugin
    (library).
