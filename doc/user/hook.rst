.. _hook:

#####
Hooks
#####

Hooks are library files (plugins) extending core Stork functionalities.

Using Hooks
=====================

The hooks are distributed as binary files with the ``.so`` extension. These
files must be placed in a specific directory to be loaded by Stork. The
default locations are:
``/usr/lib/stork-agent/hooks`` for Stork agent and
``/usr/lib/stork-server/hooks`` for Stork server. They can be modified using
the ``--hook-directory`` CLI option or by setting the
``STORK_AGENT_HOOK_DIRECTORY`` or ``STORK_SERVER_HOOK_DIRECTORY`` environment
variable.

Hooks are compiled for and can be used with one of the Stork applications (e.g., agent or server). Hooks can only be used with the exact Stork version they were compiled for. Also, if the directory from which the hooks are loaded contains any other files (not actual hook libraries), Stork will not run.

List of the Official Hooks
==========================

.. toctree::
   :glob:
   :maxdepth: 1

   hooks/**/index