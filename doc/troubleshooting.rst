.. _troubleshooting:

***************
Troubleshooting
***************

Stork Agent
===========

This section describes the solutions for some common issues with the Stork Agent.

--------------

:Issue:       A machine is authorized in the Stork Server successfully, but it has no application.
:Description: The user installed and started the Stork Server and the Stork Agent and authorized
              the machine. The "Last Refreshed" column value is set on the Machines page, the
              "Error" column value shows no error, but the "Daemons" column is still blank. The
              "Application" section on the specific Machine page is blank too.
:Solution:    Mqke sure that the daemons are running:

              - Kea Control Agent, Kea DHCPv4 server, or/and Kea DHCPv6 server
              - BIND9
:Explanation: If the "Last Refreshed" column value is set and the "Error" column value has no error,
              the communication between Stork Server and Stork Agent works correctly. It implies that
              the cause of the problem is between the Stork Agent and the daemons. Probably none of
              the Kea/BIND9 daemons are running. Stork Agent communicates with the BIND9 daemon
              directly. However, it communicates with the Kea DHCPv4 and Kea DHCPv6 servers via the
              Kea Control Agent. If you see only "CA" daemon in the Stork UI, the Kea Control Agent
              is running, but the DHCP daemons aren't.

--------------

:Issue:       After starting the Stork Agent, gets stuck in an infinite "sleeping" loop.
:Description: The Stork Agent is running with the server support (the ``--listen-prometheus-only` flag is unused).
              The ``try to register agent in Stork server`` message is displayed initially. Next, the agent only
              prints the recurring ``sleeping for 10 seconds before next registration attempt`` message.
:Solution 1.: The Stork Server isn't running. Start the Stork Server first and restart the Stork Agent daemon.
:Solution 2.: The configured server URL in the Stork Agent is invalid. Correct the URL and restart the agent.

--------------

:Issue:       After starting the Stork Agent, it keeps printing the following messages: "loaded server cert:
              /var/lib/stork-agent/certs/cert.pem and key: /var/lib/stork-agent/certs/key.pem"
:Description: The Stork Agent runs correctly, and its registration is successful. After the "started serving
              Stork Agent" message, the agent prints the recurring message about loading server certs.
              The network traffic analysis to the server reveals that it rejects all packets from the agent
              (TLS HELLO handshake failed).
:Solution:    Re-register the agent to regenerate the certificates. You can use the ``stork-agent register`` command. 
:Explanation: The /var/lib/stork-agent/certs/ca.pem file is missing or corrupted. The re-registration
              removes old files and creates the new ones.


--------------

:Issue:       The agent fails to start and prints ``open /var/lib/stork-agent/certs/cert.pem: no such file or directory
              could not load cert PEM file: /var/lib/stork-agent/certs/cert.pem`` error message.
:Solution:    Re-register the agent to regenerate the certificates. You can use the ``stork-agent register`` command.

--------------

:Issue:       The agent prints the message ``problem with connecting to dhcp daemon: unable to forward command to
              the dhcp6 service: No such file or directory. The server is likely to be offline``.
:Solution:    Try to start the Kea service: ``systemctl start kea-dhcp4 kea-dhcp6``
:Explanation: The ``kea-dhcp4.service`` or ``kea-dhcp6.serive`` (depending on the service type in the message) is not running.
              If the above commands don't resolve the problem, check the Kea ARM for troubleshooting.

--------------

:Issue:       The Stork Agent receives the "remote error: tls: certificate required" message from the Kea Control Agent.
:Description: The Stork Agent and the Kea Control Agent are running, but they cannot establish the connection.
              The Stork Agent log contains the error messages mentioned above.
:Solution:    Install the valid TLS certificates in the Stork Agent or set "cert-required" value in ``/etc/kea/kea-ctrl-agent.conf`` to "false".
:Explanation: By default, the Stork Agent does not use TLS when it connects to Kea. If the Kea Control Agent configuration
              includes the "cert-required" value set to "true", it requires the Stork Agent to use secure connections
              with valid, trusted TLS certificates. You can turn it off by setting the "cert-required" value to
              "false" when you use the self-signed certificates or replace the Stork Agent TLS credentials with
              the trusted ones.

--------------

:Issue:       Kea Control Agent returns ``Kea error response - status: 401, message: Unauthorized`` message.
:Description: The Stork Agent and the Kea Control Agent are running, but they can't connect.
              The Stork Agent logs contain similar messages: ``failed to parse responses from Kea:
              { "result": 401, "text": "Unauthorized" }`` or ``Kea error response - status: 401, message: Unauthorized``.
:Solution:    Update the ``/etc/stork/agent-credentials.json`` file with the valid user/password credentials.
:Explanation: The Kea Control Agent can be configured to use the Basic Authentication. If it is enabled,
              valid credentials must be provided in the Stork Agent configuration. Check if this file exists
              and contains a valid user, password, and IP address.

--------------

:Issue:       During the registration process, the Stork Agent prints the ``problem with registering machine:
              cannot parse address`` message.
:Description: The Stork is configured to use an IPv6 Link Local address. The agent prints the
              ``try to register agent in Stork server`` message and then the above error. The agent exists
              with the fatal status.
:Solution:    Use a global IPv6 or an IPv4 address.
:Explanation: The IPv6 Link Local addresses aren't supported by the Stork Server.

--------------

:Issue:       During the registration process, the Stork Agent prints the ``problem with registering machine:
              Post "/api/machines": unsupported protocol scheme ""`` message.
:Solution:    The ``--server-url`` argument is provided in the wrong format. It must be a canonical URL.
              It should begin with the protocol (``http://`` or ``https://``), contain the host (DNS name or
              IP address - for IPv6 escape them with the square brackets), and end with the port
              (delimited from the host by a colon). For example: ``http://storkserver:8080``.

---------------

:Issue:       The values in the ``/etc/stork/agent.env`` or ``/etc/stork/agent-credentials.json`` were changed
              but it didn't affect the Stork Agent.
:Solution:    Restart the daemon.
:Explanation: The Stork Agent reads configurations only at startup.

--------------

:Issue:       The values in the ``/etc/stork/agent.env`` were changed, and the daemon was restarted, but
              the agent still uses the default values.
:Description: The agent is running using the ``stork-agent`` command. It uses the parameters passed
              from the command-line but ignores the ``/etc/stork/agent.env`` file entries.
              If the agent is running as the SystemD daemon it uses expected values.
:Solution:    Load the environment variables from the ``/etc/stork/agent.env`` file before running the CLI tool.
              For example, you can run ``. /etc/stork/agent.env``.
:Explanation: The ``/etc/stork/agent.env`` contains the environment variables. They aren't automatically
              loaded by the Stork Agent and it must be loaded manually. The default SystemD service unit is
              configured to load this file before starting the agent.
