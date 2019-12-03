.. _devel:

*****************
Developer's Guide
*****************

.. note::

   We acknowledge that users and developers are two different groups of people, so the documents
   should eventually be separated. However, since these are still very early days of the project,
   this section is kept in the Stork ARM for convenience only.

Generating Documentation
========================

To generate documentation, simply type ``rake doc``. You need to have Sphinx and rtd-theme installed.
The documentation will be available in the ``doc/singlehtml`` directory.

Agent API
=========

The connection between the server and the agents is done using gRPC over http/2. The agent API
definition is kept in the ``backend/api/agent.proto`` file. For debugging purposes, it is possible
to connect to the agent using `grpcurl <https://github.com/fullstorydev/grpcurl>`_ tool. For example,
you can retrieve a list of currently provided gRPC calls by using this command:

.. code:: console

    $ grpcurl -plaintext -proto backend/api/agent.proto localhost:8888 describe
    agentapi.Agent is a service:
    service Agent {
      rpc detectServices ( .agentapi.DetectServicesReq ) returns ( .agentapi.DetectServicesRsp );
      rpc getState ( .agentapi.GetStateReq ) returns ( .agentapi.GetStateRsp );
      rpc restartKea ( .agentapi.RestartKeaReq ) returns ( .agentapi.RestartKeaRsp );
    }

You can also call specific gRPC calls. For example, to get the state, the following command can be
used:

.. code:: console

    $ grpcurl -plaintext -proto backend/api/agent.proto localhost:8888 agentapi.Agent.getState
    {
      "agentVersion": "0.1.0",
      "hostname": "copernicus",
      "cpus": "8",
      "cpusLoad": "1.68 1.46 1.28",
      "memory": "16",
      "usedMemory": "59",
      "uptime": "2",
      "os": "darwin",
      "platform": "darwin",
      "platformFamily": "Standalone Workstation",
      "platformVersion": "10.14.6",
      "kernelVersion": "18.7.0",
      "kernelArch": "x86_64",
      "hostID": "c41337a1-0ec3-3896-a954-a1f85e849d53"
    }

