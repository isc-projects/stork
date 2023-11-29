.. _troubleshooting:

***************
Troubleshooting
***************

``stork-agent``
===============

This section describes the solutions for some common issues with the Stork agent.

--------------

:Issue:       A machine is authorized in the Stork server successfully, but there are no applications.
:Description: The user installed and started ``stork-server`` and ``stork-agent`` and authorized
              the machine. The "Last Refreshed" column has a value on the Machines page, the
              "Error" column value shows no errors, but the "Daemons" column is still blank.
              The "Application" section on the specific Machine page is also blank.
:Solution:    Make sure that the daemons are running:

              - Kea Control Agent, Kea DHCPv4 server, and/or Kea DHCPv6 server
              - BIND 9

              Stork looks for the processes named ``kea-ctrl-agent`` (for Kea) or ``named`` (for BIND 9). Make sure
              those processes are running and are named appropriately. You may use the ``ps aux`` (or similar) command
              to debug if the processes are running. Currently Stork does not support detecting off-line services. If
              BIND 9 is located in an uncommon location and Stork agent is unable to detect it, there are two steps that
              may be helpful. You may enable DEBUG logging level, so the agent will print more detailed information
              about locations being checked.

              For BIND9, the detection process consists of four steps. The next
              step is only performed if the previous one failed. The steps are:

              1. Try to parse -c parameter of the running process;
              2. Use STORK_BIND9_CONFIG environment variable;
              3. Try to parse output of the named -V command;
              4. Try to find named.conf in the default locations.

              You may define ``STORK_BIND9_CONFIG`` environment variable to specify
              exact location of the BIND 9 configuration file.

              For BIND 9, make sure that the rndc channel is enabled. By
              default, it is enabled, even if the ``controls`` clause is
              missing. Stork is able to detect default values, so typically
              there is no administrative action required, unless the rndc channel
              was explicitly disabled. Make sure the rndc key is readable by
              Stork agent.

              Also, make sure that BIND 9 has statistics channel enabled. That
              is done by adding ``statistics-channels`` entry. Typically, this
              looks like the following:

              .. code-block:: console

                statistics-channels {
                    inet 127.0.0.1 port 8053 allow { 127.0.0.1; };
                };

              but it may vary greatly, depending on your setup. Please consult
              BIND 9 ARM for details.

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
:Solution 1.: ``stork-server`` is not running. Start the Stork server first and restart the ``stork-agent`` daemon.
:Solution 2.: The configured server URL in ``stork-agent`` is invalid. Correct the URL and restart the agent.

--------------

:Issue:       After starting ``stork-agent``, it keeps printing the following messages:
              ``loaded server cert: /var/lib/stork-agent/certs/cert.pem and key: /var/lib/stork-agent/certs/key.pem``
:Description: ``stork-agent`` runs correctly, and its registration is successful.
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

:Issue:       A connection problem to the DHCP daemon(s).
:Description: The agent prints the message ``problem with connecting to dhcp daemon: unable to forward command to
              the dhcp6 service: No such file or directory. The server is likely to be offline``.
:Solution 1:  Try to start the Kea service: ``systemctl start kea-dhcp4 kea-dhcp6``
:Solution 2:  Ensure that the ``control-socket`` entry is specified in the Kea DHCP configuration file (``kea-dhcp4.conf``
              or ``kea-dhcp6.conf``)
:Explanation: The ``kea-dhcp4.service`` or ``kea-dhcp6.service`` (depending on the service type in the message) may be not running.
              If the DHCP daemon is running and operational (it allocates the leases), but the problem is still occurring,
              inspect the DHCP daemon configuration file (``kea-dhcp4.conf`` or ``kea-dhcp6.conf``). The file must
              contain the top-level ``control-socket`` property with valid content. See the Kea
              `DHCPv4 <https://kea.readthedocs.io/en/latest/arm/dhcp4-srv.html#management-api-for-the-dhcpv4-server>`_ or
              `DHCPv6 <https://kea.readthedocs.io/en/latest/arm/dhcp6-srv.html#management-api-for-the-dhcpv6-server>`_
              ARM for details. This property is missing by default if you install Kea from the Debian/Ubuntu repository.
              To avoid this and similar problems, we recommend to use our official packages available on
              `CloudSmith <https://cloudsmith.io/~isc/repos>`_.

--------------

:Issue:       ``stork-agent`` receives a ``remote error: tls: certificate required`` message from the Kea Control Agent.
:Description: The Stork agent and the Kea Control Agent are running, but they cannot establish a connection.
              The ``stork-agent`` log contains the error message mentioned above.
:Solution:    Install the valid TLS certificates in ``stork-agent`` or set the ``cert-required`` value in ``/etc/kea/kea-ctrl-agent.conf`` to ``false``.
:Explanation: By default, ``stork-agent`` does not use TLS when it connects to Kea. If the Kea Control Agent configuration
              includes the ``cert-required`` value set to ``true``, it requires the Stork agent to use secure connections
              with valid, trusted TLS certificates. It can be turned off by setting the ``cert-required`` value to
              ``false`` when using self-signed certificates, or the Stork agent TLS credentials
              can be replaced with trusted ones.

--------------

:Issue:       Kea Control Agent returns a ``Kea error response - status: 401, message: Unauthorized`` message.
:Description: The Stork agent and the Kea Control Agent are running, but they cannot connect.
              The ``stork-agent`` logs contain similar messages: ``failed to parse responses from Kea:
              { "result": 401, "text": "Unauthorized" }`` or ``Kea error response - status: 401, message: Unauthorized``.
:Solution:    Update the ``/etc/stork/agent-credentials.json`` file with the valid user/password credentials.
:Explanation: The Kea Control Agent can be configured to use Basic Authentication. If it is enabled,
              valid credentials must be provided in the ``stork-agent`` configuration. Verify that this file exists
              and contains a valid username, password, and IP address.

--------------

:Issue:       During the registration process, ``stork-agent`` prints a ``problem with registering machine:
              cannot parse address`` message.
:Description: Stork is configured to use an IPv6 link-local address. The agent prints the
              ``try to register agent in Stork server`` message and then the above error. The agent exists
              with a fatal status.
:Solution:    Use a global IPv6 or an IPv4 address.
:Explanation: IPv6 link-local addresses are not supported by ``stork-server``.

--------------

:Issue:       A protocol problem occurs during the agent registration.
:Description: During the registration process, ``stork-agent`` prints a ``problem with registering machine:
              Post "/api/machines": unsupported protocol scheme ""`` message.
:Solution:    The ``--server-url`` argument is provided in the wrong format; it must be a canonical URL.
              It should begin with the protocol (``http://`` or ``https://``), contain the host (DNS name or
              IP address; for IPv6 escape them with square brackets), and end with the port
              (delimited from the host by a colon). For example: ``http://storkserver:8080``.

---------------

:Issue:       The values in ``/etc/stork/agent.env`` or ``/etc/stork/agent-credentials.json`` were changed,
              but ``stork-agent`` does not noticed the changes.
:Solution 1.: Restart the daemon.
:Solution 2.: Send the SIGHUP signal to the ``stork-agent`` process.
:Explanation: ``stork-agent`` reads configurations at startup or after receiving the SIGHUP signal.

--------------

:Issue:       The values in ``/etc/stork/agent.env`` were changed and the Stork agent was restarted, but
              it still uses the default values.
:Description: The agent is running using the ``stork-agent`` command. It uses the parameters passed
              from the command line but ignores the ``/etc/stork/agent.env`` file entries.
              If the agent is running as the systemd daemon, it uses the expected values.
:Solution 1.: Load the environment variables from the ``/etc/stork/agent.env`` file before running Stork agent.
              For example, run ``. /etc/stork/agent.env``.
:Solution 2.: Run the Stork agent with the ``--use-env-file`` switch.
:Explanation: The ``/etc/stork/agent.env`` file contains the environment variables, but ``stork-agent`` does not automatically
              load them, unless you use ``--use-env-file flag``; the file must be loaded manually. The default systemd service
              unit is configured to load this file before starting the agent.

--------------

:Issue:       Stork shows only Kea Control Agent tab on the application page. It detects no Kea DHCP servers,
              although the DHCP daemons are running and allocating leases.
:Description: There are only a single tab titled "CA" on the Kea application page but no data about any DHCP daemon or
              DDNS. The Kea Control Agent and Kea DHCPv4 or Kea DHCPv6 daemon are running and serve leases. The Stork
              agent logs comprises the ``The Kea application has no DHCP daemons configured`` message.
:Solution:    The ``kea-ctrl-agent.conf`` file misses the ``control-sockets`` property.
:Explanation: Stork detects Kea components using the control socket list from the Kea Control Agent configuration file.
              The list must be configured properly to allow Stork to send commands to Kea daemons. See
              `Kea ARM <https://kea.readthedocs.io/en/latest/arm/agent.html#configuration>` for details.
              This property is missing by default if you install Kea from the Debian/Ubuntu repository.
              To avoid this and similar problems, we recommend to use our official packages available on
              `CloudSmith <https://cloudsmith.io/~isc/repos>`_.
:Issue:       Stork agent doesn't start with the following error:
              ``failed to load hooks from directory: '[HOOK DIRECTORY]': plugin.Open("[HOOK DIRECTORY]/[FILENAME]"): [HOOK DIRECTORY]/[FILENAME]: file too short`` or
              ``failed to load hooks from directory: '[HOOK DIRECTORY]': plugin.Open("[HOOK DIRECTORY]/[FILENAME]"): [HOOK DIRECTORY]/[FILENAME]: invalid ELF header``
:Solution:    Remove the given file from the hook directory.
:Explanation: The file under a given path is not valid Stork hook.

--------------

:Issue:       Stork agent doesn't start with the following error:
              ``Cannot start the Stork Agent: incompatible hook version: 1.0.0``
:Solution:    Update the given hook.
:Explanation: The hook is out-of-date. It's incompatible with the Stork core
              application.

--------------

:Issue:       Stork agent doesn't start with the following error:
              ``Cannot start the Stork Agent: plugin: symbol Version not found in plugin``
:Solution:    Remove or fix the given file.
:Explanation: Hook directory contains Go plugin but that is not a hook; Hook
              doesn't contain required symbol.

--------------

:Issue:       Stork agent doesn't start with the following error:
              ``Cannot start the Stork Agent: hook library dedicated for another program: Stork Server``
:Solution:    Move the incompatible hooks to a separate directory.

--------------

:Issue:       Stork agent starts but the hooks aren't loaded. The logs comprise
              the following message:
              ``Cannot find plugin paths in: /var/lib/stork-agent/hooks: cannot list hook directory: /var/lib/stork-agent/hooks: open /var/lib/stork-agent/hooks: no such file or directory``
:Solution:    Create the hook directory or change the path in the configuration.
:Explanation: Hook directory doesn't exist.

--------------

:Issue:       Stork agent doesn't start with the following error:
              ``Cannot start the Stork Agent: open [HOOK DIRECTORY]: permission denied cannot list hook directory``
:Solution:    Grant the right for read the hook directory for the Stork user.
:Explanation: The hook directory is not readable.

--------------

:Issue:       Stork agent doesn't start with the following error:
              ``Cannot start the Stork Agent: readdirent [HOOK DIRECTORY]/[FILENAME]: not a directory cannot list hook directory``
:Solution:    Change the hook directory path.
:Explanation: Directory is a file.

``stork-server``
================

This section describes the solutions for some common issues with the Stork server.

---------------

:Issue:       The values in ``/etc/stork/server.env`` were changed,
              but ``stork-server`` does not noticed the changes.
:Solution 1.: Restart the daemon.
:Solution 2.: Send the SIGHUP signal to the ``stork-server`` process.
:Explanation: ``stork-server`` reads configurations at startup or after receiving the SIGHUP signal.

--------------

:Issue:       The values in ``/etc/stork/server.env`` were changed and the Stork server was restarted, but
              it still uses the default values.
:Description: The server is running using the ``stork-server`` command. It uses the parameters passed
              from the command line but ignores the ``/etc/stork/server.env`` file entries.
              If the server is running as the systemd daemon, it uses the expected values.
:Solution 1.: Load the environment variables from the ``/etc/stork/server.env`` file before running Stork server.
              For example, run ``. /etc/stork/server.env``.
:Solution 2.: Run the Stork server with the ``--use-env-file`` switch.
:Explanation: The ``/etc/stork/server.env`` file contains the environment variables, but ``stork-server`` does not automatically
              load them, unless you use ``--use-env-file`` flag; the file must be loaded manually. The default systemd service
              unit is configured to load this file before starting the agent.

--------------

:Issue:       The server is running but rejects the HTTP requests due to the TLS handshake error.
:Description: The HTTP requests sent via an Internet browser or tools like ``curl`` are rejected. The clients show a
              message similar to: ``OpenSSL SSL_write: Broken pipe, errno 32``. The Stork  server logs contain the
              ``TLS handshake error`` entry with the ``tls: client didn't provide a certificate`` description.
:Solution 1.: Leave the ``STORK_REST_TLS_CA_CERTIFICATE`` environment variable and the ``--rest-tls-ca`` flag empty.
:Solution 2.: Configure the Internet browser or HTTP tool to use the valid and trusted TLS client certificate.
              The client certificate must be signed by the authority whose CA certificate was provided in the server
              configuration.
:Explanation: Providing the ``STORK_REST_TLS_CA_CERTIFICATE`` environment variable or the ``--rest-tls-ca`` flag turns
              on the TLS client certificate verification. The HTTP requests must be assigned with the valid and trusted
              HTTP certificate signed by the authority whose CA certificate was provided in the server configuration.
              Otherwise, the request will be rejected. This option is dedicated to improving server security by limiting
              access to only trusted users. You shouldn't use it if you don't have a CA configured or want to allow to
              login to the Stork server from any computer without prior setup.

--------------

:Issue:       Server doesn't start and prints the ``permission denied for schema public`` message.
:Description: The fresh installation of the Stork server is made, and the database is empty. The Stork server doesn't
              start, and the Stork tool returns an error on the database migration. The logs reveal the denied access to
              the schema public.
:Solution 1.: Execute the ``GRANT ALL ON DATABASE stork_db TO stork_user;`` on the Stork database (replace ``stork_db``
              and ``stork_user`` with the proper names).
:Solution 2.: Perform migration using Stork tool with the maintenance (e.g., superuser) database credentials.
:Explanation: In some Postgres installations (by default in Postgres 15 and above), the ``CREATE`` permission is not
              initially granted to all users except the database owner. The stork server needs this permission to
              perform the database migration on startup. You can grant this permission or use the Stork tool to migrate
              the schema as the maintenance database user (e.g., superuser).

--------------

:Issue:       Stork server doesn't start with the following error:
              ``Cannot start the Stork Server: failed to load hooks from directory: '[HOOK DIRECTORY]': plugin.Open("[HOOK DIRECTORY]/[FILENAME]"): [HOOK DIRECTORY]/[FILENAME]: file too short`` or
              ``Cannot start the Stork Server: failed to load hooks from directory: '[HOOK DIRECTORY]': plugin.Open("[HOOK DIRECTORY]/[FILENAME]"): [HOOK DIRECTORY]/[FILENAME]: invalid ELF header``
:Solution:    Remove the given file from the hook directory.
:Explanation: The file under a given path is not valid Stork hook.

--------------

:Issue:       Stork server doesn't start with the following error:
              ``Cannot start the Stork Server: incompatible hook version: 1.0.0``
:Solution:    Update the given hook.
:Explanation: The hook is out-of-date. It's incompatible with the Stork core
              application.

--------------

:Issue:       Stork server doesn't start with the following error:
              ``Cannot start the Stork Server: plugin: symbol Version not found in plugin``
:Solution:    Remove or fix the given file.
:Explanation: Hook directory contains Go plugin but that is not a hook; Hook
              doesn't contain required symbol.

--------------

:Issue:       Stork server doesn't start with the following error:
              ``Cannot start the Stork Server: hook library dedicated for another program: Stork Agent``
:Solution:    Move the incompatible hooks to a separate directory.

--------------

:Issue:       Stork server starts but the hooks aren't loaded. The logs comprise
              the following message:
              ``Cannot find plugin paths in: /var/lib/stork-server/hooks: cannot list hook directory: /var/lib/stork-server/hooks: open /var/lib/stork-server/hooks: no such file or directory``
:Solution:    Create the hook directory or change the path in the configuration.
:Explanation: Hook directory doesn't exist.

--------------

:Issue:       Stork server doesn't start with the following error:
              ``Cannot start the Stork Server: open [HOOK DIRECTORY]: permission denied cannot list hook directory``
:Solution:    Grant the right for read the hook directory for the Stork user.
:Explanation: The hook directory is not readable.

--------------

:Issue:       Stork server doesn't start with the following error:
              ``Cannot start the Stork Server: readdirent [HOOK DIRECTORY]/[FILENAME]: not a directory cannot list hook directory``
:Solution:    Change the hook directory path.
:Explanation: Directory is a file.
