.. _hook:

#####
Hooks
#####

The hook is an additional file (plugin) that extends the standard Stork
functionalities. It contains functions that are called during handling of
various operations and can change the typical flow.

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

.. toctree::
   :glob:
   :caption: List of the Official Hooks
   :maxdepth: 1

   hooks/**/index