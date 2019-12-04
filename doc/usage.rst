.. _usage:

***********
Using Stork
***********

This section describes how to use features available in stork. To connect to Stork, use your
web browser and connect to port 4200. If Stork is running on your localhost, you can navigate
to http://localhost:4200.

Managing users
==============

Currently, the default administrator's account is created and can be used to sign in to the system
via the web UI. Please use the login ``admin`` and password ``admin`` to sign in to the system.

To manage users, click on the ``Configuration`` menu and choose ``Users``. You will see a list of
existing users. At the very least, there will be user ``admin``.

To add new user, click ``Create User Account``. A new tab will opened that will let you specify the
new account parameters. Some fields have specific restrictions. Username can consist of only
letters, numbers and underscore. E-mail field is optional. However, if specified, it must be a well
formed e-mail. First and lastname fields are mandatory. Password must only contain letters, digits,
@, ., !, +, - and must be at least 8 characters long. Once all requirements are met, the ``Save``
button will become active and you will be able to add new account.

.. note::

    As of Stork 0.2.0 release, the role-based access control is not implemented yet. Every user
    is considered a super-admin and has full control over the system.


Deploying Stork Agent
=====================

Stork system uses agents to monitor services. Stork Agent (`STAG` or simply `agent`) is a
daemon that is expected to be deployed and run on each machine to be monitored. As of Stork 0.2.0
release there are no automated deployment routines and STAG has to be copied and run manually.
This can be done in a variety of ways. Here is one of them.

Assuming you want to monitor services running on machine with IP 192.0.2.1, you can do the following
on the Stork server command line:

.. code:: console

    cd <stork-dir>
    scp backend/cmd/stork-agent login@192.0.2.1:/path

On the machine to be monitored, you need to start the agent. In the basic case, you can simply
run it:

.. code:: console

    ./stork-agent

You can optionally pass ``--host=`` or set the ``STORK_AGENT_ADDRESS`` environment variable to
specify which address the agent will listen on. You can pass ``--port`` or set the ``STORK_AGENT_PORT``
environment variable to specify which TCP port the agent will listen on.

.. note::

   Unless explicitly specified, the agent will listen on all addresses on port 8080. There are no
   authentication mechanisms implemented in the agent yet. Use with care!

Registering New Machine
=======================

Once the agent is deployed and running on the machine to be monitored, you should instruct Stork
server to start monitoring it. You can do so by going to Services menu and choosing Machines.
You will be presented with a list of currently registered machines.

To add a new machine, click ``Add New Machine``. You need to specify the machine address or hostname
and a port. If Stork agent is running in a container, you may specify the container name. This is
particularly useful, if you built stork using ``rake docker_up`` command and the agent is running in
a container. In such case, you can use kea-agent as your hostname. If you run agent by using ``rake
run_agent``, the agent will listen on port 8888.

Once you click Add, the server will attempt to establish gRPC over http/2 connection to the agent.
Make sure that any firewalls in between will allow incoming connections to the TCP port specified.

Once a machine is added, number of parameters, such as hostname, address, agent version, number
of CPU cores, CPU load, available total memory, current memory utilization, uptime, OS, platform
family, platform name, OS version, kernel, virtualization details (if any), host ID and other
information will be provided. More information will become available in the future Stork versions.

Monitoring Machines
===================

To monitor registered machines, go to Services menu and click Machines. A list of currently
registered machines will be displayed. Pagination mechanism is available to display larger
number of machines.

There is a filtering mechanism that acts as an omnibox. The string typed is searched for an address,
agent version, hostname, OS, platform, OS version, kernel version, kernel architecture,
virtualization system, host-id fields. The filtering happens once you hit ENTER.

You can inspect the state of a machine by clicking its hostname. A new tab will open with machine
details. Multiple tabs can be open at the same time. You can click Refresh state to get updated
information.

The machine state can also be refreshed using Action menu. On the machines list, each machine has
its own menu. Click on the triple lines button at the right side and choose the Refresh option.

Deleting Machines
=================

To stop monitoring a machine, you can go to the Machines list, find the machine you want to stop
monitoring, click on the triple lines button at the right side and choose Delete. Note this will
terminate the connection between Stork server and the agent running on the machine and the server
will no longer monitor it. However, the Stork agent process will continue running. If you want to
completely shut it down, you need to do so manually, e.g. by connecting to the machine using ssh and
stopping the agent there. One way to achieve that is to issue ``killall stork-agent`` command.
