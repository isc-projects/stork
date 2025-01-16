.. _troubleshooting:

***************
Troubleshooting
***************

``stork-agent``
===============

This section describes the solutions for some common issues with the Stork agent.

--------------

:Issue:       A machine is authorized in the Stork server successfully, but there are no applications.
:Description: The user has installed and started ``stork-server`` and ``stork-agent`` and authorized
              the machine. The "Last Refreshed" column has a value on the Machines page and the
              "Error" column value shows no errors, but the "Daemons" column is still blank.
              The "Application" section on the specific Machine page is also blank.
:Solution:    Make sure that the daemons are running:

              - Kea Control Agent, Kea DHCPv4 server, and/or Kea DHCPv6 server
              - BIND 9

              Stork looks for the processes named ``kea-ctrl-agent`` (for Kea) or ``named`` (for BIND 9). Make sure
              those processes are running and are named appropriately. Use the ``ps aux`` (or similar) command
              to determine whether the processes are running. Currently, Stork does not support detecting off-line services. If
              BIND 9 is located in an uncommon location and the Stork agent is unable to detect it, there are several steps that
              may be helpful. First, enable the DEBUG logging level, so the agent will print more detailed information
              about locations being checked.

              Stork attempts the next four actions, in order:

              1. Try to parse the ``-c`` parameter of the running process;
              2. Use the STORK_AGENT_BIND9_CONFIG environment variable;
              3. Try to parse the output of the ``named -V`` command;
              4. Try to find the named.conf file in the default locations.

              The ``STORK_AGENT_BIND9_CONFIG`` environment variable may be defined to specify
              the exact location of the BIND 9 configuration file.

              For BIND 9, make sure that the rndc channel is enabled. By
              default it is enabled, even if the ``controls`` clause is
              missing. Stork is able to detect default values, so typically
              there is no administrative action required, unless the rndc channel
              was explicitly disabled. Make sure the rndc key is readable by
              the Stork agent.

              Also, make sure that BIND 9 has the statistics channel enabled,
              by adding a ``statistics-channels`` entry. Typically, this
              looks like the following:

              .. code-block:: console

                statistics-channels {
                    inet 127.0.0.1 port 8053 allow { 127.0.0.1; };
                };

              but it may vary greatly, depending on a given setup. Please consult
              the BIND 9 Administrator Reference Manual (ARM) for details.

:Explanation: If the "Last Refreshed" column has a value, and the "Error" column value has no errors,
              the communication between ``stork-server`` and ``stork-agent`` works correctly, which implies that
              the cause of the problem is between the Stork agent and the daemons. The most likely issue is that none of
              the Kea/BIND 9 daemons are running. ``stork-agent`` communicates with the BIND 9 daemon
              directly; however, it communicates with the Kea DHCPv4 and Kea DHCPv6 servers via the
              Kea Control Agent. If only the "CA" daemon is displayed in the Stork interface, the Kea Control Agent
              is running, but the DHCP daemons are not.

--------------

:Issue:       After starting the Stork agent, it gets stuck in an infinite "sleeping" loop.
:Description: ``stork-agent`` is running with server support (the ``--listen-prometheus-only`` flag is unused).
              The ``try to register agent in Stork server`` message is displayed initially, but the agent only
              prints the recurring ``sleeping for 10 seconds before next registration attempt`` message.
:Solution 1:  ``stork-server`` is not running. Start the Stork server first and restart the ``stork-agent`` daemon.
:Solution 2:  The configured server URL in ``stork-agent`` is invalid. Correct the URL and restart the agent.

--------------

:Issue:       ``stork-agent`` starts but returns the
              ``loaded server cert: /var/lib/stork-agent/certs/cert.pem and key: /var/lib/stork-agent/certs/key.pem`` message.
:Description: ``stork-agent`` runs correctly and its registration is successful.
              After the ``started serving Stork Agent`` message, the agent prints the recurring message about loading server certs.
              The network traffic analysis to the server reveals that it rejects all packets from the agent
              (TLS HELLO handshake failed).
:Solution:    Re-register the agent to regenerate the certificates, using the ``stork-agent register`` command.
:Explanation: The ``/var/lib/stork-agent/certs/ca.pem`` file is missing or corrupted.
              The re-registration removes old files and creates new ones.

--------------

:Issue:       The cert PEM file is not loaded.
:Description: The agent fails to start and prints an ``open /var/lib/stork-agent/certs/cert.pem: no such file or directory
              could not load cert PEM file: /var/lib/stork-agent/certs/cert.pem`` error message.
:Solution:    Re-register the agent to regenerate the certificates, using the ``stork-agent register`` command.

--------------

:Issue:       A connection problem to the DHCP daemon(s) is occurring.
:Description: The agent prints the message ``problem with connecting to dhcp daemon: unable to forward command to
              the dhcp6 service: No such file or directory. The server is likely to be offline``.
:Solution 1:  Try to start the Kea service: ``systemctl start kea-dhcp4 kea-dhcp6``.
:Solution 2:  Ensure that the ``control-socket`` entry is specified in the Kea DHCP configuration file (``kea-dhcp4.conf``
              or ``kea-dhcp6.conf``).
:Explanation: The ``kea-dhcp4.service`` or ``kea-dhcp6.service`` (depending on the service type in the message) may not be running.
              If the DHCP daemon is running and operational (it allocates the leases), but the problem is still occurring,
              inspect the DHCP daemon configuration file (``kea-dhcp4.conf`` or ``kea-dhcp6.conf``). The file must
              contain the top-level ``control-socket`` property with valid content. See the
              `DHCPv4 <https://kea.readthedocs.io/en/latest/arm/dhcp4-srv.html#management-api-for-the-dhcpv4-server>`_ or
              `DHCPv6 <https://kea.readthedocs.io/en/latest/arm/dhcp6-srv.html#management-api-for-the-dhcpv6-server>`_ section of
              the Kea ARM for details. This property is missing by default if Kea is installed from the Debian/Ubuntu repository.
              To avoid this and similar problems, we recommend using ISC's official packages on
              `Cloudsmith <https://cloudsmith.io/~isc/repos>`_.

--------------

:Issue:       ``stork-agent`` receives a ``remote error: tls: certificate required`` message from the Kea Control Agent.
:Description: The Stork agent and the Kea Control Agent are running, but they cannot establish a connection.
              The ``stork-agent`` log contains the error message mentioned above.
:Solution:    Install valid TLS certificates in ``stork-agent`` or set the ``cert-required`` value in ``/etc/kea/kea-ctrl-agent.conf`` to ``false``.
:Explanation: By default, ``stork-agent`` does not use TLS when it connects to Kea. If the Kea Control Agent configuration
              includes the ``cert-required`` value set to ``true``, it requires the Stork agent to use secure connections
              with valid, trusted TLS certificates. It can be turned off by setting the ``cert-required`` value to
              ``false`` when using self-signed certificates, or the Stork agent TLS credentials
              can be replaced with trusted ones.

--------------

:Issue:       The Kea Control Agent returns a ``Kea error response - status: 401, message: Unauthorized`` message.
:Description: The Stork agent and the Kea Control Agent are running, but they cannot connect.
              The ``stork-agent`` logs contain similar messages: ``failed to parse responses from Kea:
              { "result": 401, "text": "Unauthorized" }`` or ``Kea error response - status: 401, message: Unauthorized``.
:Solution:    Check if there are any clients specified in the Kea Control Agent configuration file in the
              ``authentication`` node.
:Explanation: The Kea Control Agent can be configured to use Basic Authentication. If it is enabled, Stork agent will
              read the credentials from the Kea CA configuration file and use them to authenticate. The Stork agent
              chooses credentials with user name beginning with ``stork``. If there is no such user, the agent will use
              the first user from the list.

--------------

:Issue:       During the registration process, ``stork-agent`` returns a 
              ``problem with registering machine: cannot parse address`` message.
:Description: Stork is configured to use an IPv6 link-local address. The agent prints the
              ``try to register agent in Stork server`` message and then the above error. The agent exists
              with a fatal status.
:Solution:    Use a global IPv6 or an IPv4 address.
:Explanation: IPv6 link-local addresses are not supported by ``stork-server``.

--------------

:Issue:       A protocol problem occurs during the agent registration.
:Description: During the registration process, ``stork-agent`` prints a
              ``problem with registering machine: Post "/api/machines": unsupported protocol scheme ""`` message.
:Solution:    The ``--server-url`` argument is provided in the wrong format; it must be a canonical URL.
              It should begin with the protocol (``http://`` or ``https://``), contain the host (DNS name or
              IP address; for IPv6 escape them with square brackets), and end with the port
              (delimited from the host by a colon). For example: ``http://storkserver:8080``.

---------------

:Issue:       The values in ``/etc/stork/agent.env``  were changed, but ``stork-agent`` does not notice the changes.
:Solution 1:  Restart the daemon.
:Solution 2:  Send the SIGHUP signal to the ``stork-agent`` process.
:Explanation: ``stork-agent`` reads configurations at startup or after receiving the SIGHUP signal.

--------------

:Issue:       The values in ``/etc/stork/agent.env`` were changed and the Stork agent was restarted, but
              it still uses the default values.
:Description: The agent is running using the ``stork-agent`` command. It uses the parameters passed
              from the command line but ignores the ``/etc/stork/agent.env`` file entries.
              If the agent is running as the systemd daemon, it uses the expected values.
:Solution 1:  Load the environment variables from the ``/etc/stork/agent.env`` file before running Stork agent.
              For example, run ``. /etc/stork/agent.env``.
:Solution 2:  Run the Stork agent with the ``--use-env-file`` switch.
:Explanation: The ``/etc/stork/agent.env`` file contains the environment variables, but ``stork-agent`` does not automatically
              load them unless the ``--use-env-file flag`` is set; the file must be loaded manually. The default ``systemd`` service
              unit is configured to load this file before starting the agent.

--------------

:Issue:       Stork shows only the Kea Control Agent tab on the Apps page. It detects no Kea DHCP servers,
              although the DHCP daemons are running and allocating leases.
:Description: There is only a single tab titled "CA" on the Kea Apps page, but no data about any DHCP daemon or
              DDNS. The Kea Control Agent and Kea DHCPv4 or Kea DHCPv6 daemon are running and serve leases. The Stork
              agent log includes the ``The Kea application has no DHCP daemons configured`` message.
:Solution:    The ``kea-ctrl-agent.conf`` file is missing the ``control-sockets`` property.
:Explanation: Stork detects Kea components using the control socket list from the Kea Control Agent configuration file.
              The list must be configured properly to allow Stork to send commands to Kea daemons. See
              `the Kea ARM <https://kea.readthedocs.io/en/latest/arm/agent.html#configuration>` for details.
              This property is missing by default if Kea is installed from the Debian/Ubuntu repository.
              To avoid this and similar problems, we recommend using ISC's official packages on
              `Cloudsmith <https://cloudsmith.io/~isc/repos>`_.

--------------

:Issue:       The Stork agent fails to start and returns the following error:
              ``failed to load hooks from directory: '[HOOK DIRECTORY]': plugin.Open("[HOOK DIRECTORY]/[FILENAME]"): [HOOK DIRECTORY]/[FILENAME]: file too short`` or
              ``failed to load hooks from directory: '[HOOK DIRECTORY]': plugin.Open("[HOOK DIRECTORY]/[FILENAME]"): [HOOK DIRECTORY]/[FILENAME]: invalid ELF header``.
:Solution:    Remove the given file from the hook directory.
:Explanation: The file under a given path is not a valid Stork hook.

--------------

:Issue:       The Stork agent fails to start and returns the following error:
              ``Cannot start the Stork Agent: incompatible hook version: 1.0.0``.
:Solution:    Update the given hook.
:Explanation: The hook is out-of-date and is incompatible with the Stork core
              application.

--------------

:Issue:       The Stork agent fails to start and returns the following error:
              ``Cannot start the Stork Agent: plugin: symbol Version not found in plugin``.
:Solution:    Remove or fix the given file.
:Explanation: The hook directory contains the Go plugin, but not the hook; the Go hook
              does not contain a required symbol.

--------------

:Issue:       The Stork agent fails to start and returns the following error:
              ``Cannot start the Stork Agent: hook library dedicated for another program: Stork Server``.
:Solution:    Move the incompatible hooks to a separate directory.
:Explanation: The Stork agent requires the hook directory to contain only agent
              hooks. The error message indicates that the hook directory
              contains hooks dedicated to the Stork server.

--------------

:Issue:       The Stork agent starts but the hooks are not loaded. The logs include
              the following message:
              ``Cannot find plugin paths in: /usr/lib/stork-agent/hooks: cannot list hook directory: /usr/lib/stork-agent/hooks: open /usr/lib/stork-agent/hooks: no such file or directory``.
:Solution:    Create the hook directory or change the path in the configuration.
:Explanation: The hook directory does not exist.

--------------

:Issue:       The Stork agent fails to start and returns the following error:
              ``Cannot start the Stork Agent: open [HOOK DIRECTORY]: permission denied cannot list hook directory``.
:Solution:    Grant read access to the hook directory to the ``stork-agent`` user.
:Explanation: The hook directory is not readable.

--------------

:Issue:       The Stork agent fails to start and returns the following error:
              ``Cannot start the Stork Agent: readdirent [HOOK DIRECTORY]/[FILENAME]: not a directory cannot list hook directory``.
:Solution:    Change the hook directory path.
:Explanation: A file was found instead of a directory under the given hook directory path.

``stork-server``
================

This section describes the solutions for some common issues with the Stork server.

---------------

:Issue:       The values in ``/etc/stork/server.env`` were changed,
              but ``stork-server`` does not notice the changes.
:Solution 1:  Restart the daemon.
:Solution 2:  Send the SIGHUP signal to the ``stork-server`` process.
:Explanation: ``stork-server`` reads configurations at startup or after receiving the SIGHUP signal.

--------------

:Issue:       The values in ``/etc/stork/server.env`` were changed and the Stork server was restarted, but
              it still uses the default values.
:Description: The server is running using the ``stork-server`` command. It uses the parameters passed
              from the command line but ignores the ``/etc/stork/server.env`` file entries.
              If the server is running as the ``systemd`` daemon, it uses the expected values.
:Solution 1:  Load the environment variables from the ``/etc/stork/server.env`` file before running the Stork server.
              For example, run ``. /etc/stork/server.env``.
:Solution 2:  Run the Stork server with the ``--use-env-file`` switch.
:Explanation: The ``/etc/stork/server.env`` file contains the environment variables, but ``stork-server`` does not automatically
              load them unless the ``--use-env-file`` flag is set; the file must be loaded manually. The default ``systemd`` service
              unit is configured to load this file before starting the agent.

--------------

:Issue:       The server is running but rejects HTTP requests due to a TLS handshake error.
:Description: HTTP requests sent via an Internet browser or tools like ``curl`` are rejected. The clients show a
              message similar to: ``OpenSSL SSL_write: Broken pipe, errno 32``. The Stork server logs contain a
              ``TLS handshake error`` entry with the ``tls: client didn't provide a certificate`` description.
:Solution 1:  Leave the ``STORK_REST_TLS_CA_CERTIFICATE`` environment variable and the ``--rest-tls-ca`` flag empty.
:Solution 2:  Configure the Internet browser or HTTP tool to use a valid and trusted TLS client certificate.
              The client certificate must be signed by the authority whose CA certificate was provided in the server
              configuration.
:Explanation: Providing the ``STORK_REST_TLS_CA_CERTIFICATE`` environment variable or the ``--rest-tls-ca`` flag turns
              on TLS client certificate verification. HTTP requests must be assigned with a valid and trusted
              HTTP certificate, signed by the authority whose CA certificate was provided in the server configuration;
              otherwise, the request is rejected. This option improves server security by limiting
              access to only trusted users; it should not be used if there is no CA configured, or if it is desirable to allow
              login to the Stork server from any computer without prior setup.

--------------

:Issue:       The Stork server fails to start and returns the following error: ``permission denied for schema public``.
:Description: A fresh installation of the Stork server is made and the database is empty. However, the Stork server does not
              start and the Stork tool returns an error on the database migration. The logs reveal denied access to
              the schema public.
:Solution 1:  Execute ``GRANT ALL ON DATABASE stork_db TO stork_user;`` on the Stork database (replace ``stork_db``
              and ``stork_user`` with the proper names).
:Solution 2:  Perform the migration using the Stork tool with the maintenance (e.g., super-admin) database credentials.
:Explanation: In some Postgres installations (by default in Postgres 15 and above), the ``CREATE`` permission is only
              initially granted to the database owner. The Stork server needs this permission to
              perform the database migration on startup. This permission can be granted manually, or the Stork tool can be used to migrate
              the schema as the maintenance database user (e.g., super-admin).

--------------

:Issue:       The Stork server fails to start and returns the following error:
              ``Cannot start the Stork Server: failed to load hooks from directory: '[HOOK DIRECTORY]': plugin.Open("[HOOK DIRECTORY]/[FILENAME]"): [HOOK DIRECTORY]/[FILENAME]: file too short`` or
              ``Cannot start the Stork Server: failed to load hooks from directory: '[HOOK DIRECTORY]': plugin.Open("[HOOK DIRECTORY]/[FILENAME]"): [HOOK DIRECTORY]/[FILENAME]: invalid ELF header``.
:Solution:    Remove the given file from the hook directory.
:Explanation: The file under the given path is not a valid Stork hook.

--------------

:Issue:       The Stork server fails to start and returns the following error:
              ``Cannot start the Stork Server: incompatible hook version: 1.0.0``.
:Solution:    Update the given hook.
:Explanation: The hook is out-of-date and is incompatible with the Stork core
              application.

--------------

:Issue:       The Stork server fails to start and returns the following error:
              ``Cannot start the Stork Server: plugin: symbol Version not found in plugin``.
:Solution:    Remove or fix the given file.
:Explanation: The hook directory contains the Go plugin but not the hook; the Go hook
              does not contain a required symbol.

--------------

:Issue:       The Stork server fails to start and returns the following error:
              ``Cannot start the Stork Server: hook library dedicated for another program: Stork Agent``.
:Solution:    Move the incompatible hooks to a separate directory.
:Explanation: The Stork server requires the hook directory to contain only server
              hooks. The error message indicates that the hook directory
              contains hooks dedicated to the Stork agent.

--------------

:Issue:       The Stork server starts but the hooks are not loaded. The logs include
              the following message:
              ``Cannot find plugin paths in: /usr/lib/stork-server/hooks: cannot list hook directory: /usr/lib/stork-server/hooks: open /usr/lib/stork-server/hooks: no such file or directory``.
:Solution:    Create the hook directory or change the path in the configuration.
:Explanation: The hook directory does not exist.

--------------

:Issue:       The Stork server fails to start and returns the following error:
              ``Cannot start the Stork Server: open [HOOK DIRECTORY]: permission denied cannot list hook directory``.
:Solution:    Grant read access to the hook directory to the ``stork-server`` user.
:Explanation: The hook directory is not readable.

--------------

:Issue:       The Stork server fails to start and returns the following error:
              ``Cannot start the Stork Server: readdirent [HOOK DIRECTORY]/[FILENAME]: not a directory cannot list hook directory``.
:Solution:    Change the hook directory path.
:Explanation: A file was found instead of a directory under the given hook directory path.


High Virtual Memory Usage
=========================

Stork processes allocate a large amount of virtual memory, which is a common
situation for applications written in Golang. The Go runtime uses virtual
memory to manage memory efficiently. Virtual memory is not the same as
physical memory. The size of the reserved virtual memory depends on the
internal implementation details of the Go memory allocator. A high value of
virtual memory usage is not alarming, as long as real memory usage is low.

Virtual  and physical memory usage can be examined using the ``ps aux`` command.
Virtual memory usage is displayed in the ``VSZ`` column; the
``RSS`` column shows physical memory usage.

The usual virtual memory usage of the Stork agent on a machine with 16GB RAM,
Go 1.22.4, and Ubuntu 22.04 is about 2.5-3GB.
The real memory usage is relatively low, about 10-40MB for Kea deployments with
dozens of subnets and host reservations and 40-80MB for deployments with
thousands of subnets and host reservations.

References:

- `Official Golang FAQ - Why does my Go process use so much virtual memory? <https://go.dev/doc/faq#Why_does_my_Go_process_use_so_much_virtual_memory>`_
- `Go memory management <https://povilasv.me/go-memory-management/>`_
