.. _troubleshooting:

***************
Troubleshooting
***************

Stork Agent
===========

There are described the solutions for some popular issues with the Stork Agent.

--------------

:Issue:       The machine is authorized in the Stork Server and shows no error, but has no application.
:Description: User installed and ran the Stork Server and next the Stork Agent.
              Next, it authorized the machine. Some time has passed, on the "Machines"
              page the "Last Refreshed" column value is actual and "Error" column value
              is empty, but "Daemons" colum is still blank. The "Application" section
              on the specific machine page is empty too.
:Solution:    Check that:
              - Kea Conrol Agent, Kea DHCP4, and/or Kea DHCP6
              - BIND9
              daemons are running.
:Explanation: If the "Last Refreshed" column value is actual and the "Error" column value
              has no error then the communication between Stork Server
              and Stork Agent works correctly. It means that the cause of the problem
              is between the Stork Agent and the daemons. But the Stork Agent doesn't report
              any connection problem. Probably none of the Kea/BIND9 daemons are running.
              BIND9 daemon communicates with Stork Agent directly. Kea DHCP4 and Kea DHCP6
              connect via Kea Control Agent. If you see in the Stork UI only "CA" daemon
              then it means that the Kea Control Agent is running, but the DHCP daemons aren't.

--------------

:Issue:       After start the Stork Agent stucks in infinite "sleeping" loop
:Description: The Stork Agent is running with the server support (the `--listen-prometheus-only`
              flag isn't used). On the `try to register agent in Stork server` message in the standard output
              follows only the infinite loop of the `sleeping for 10 seconds before next registration
              attempt` messages.
:Solution:    The Stork Server isn't running. First start the server service and next the Stork Agent daemon.

--------------

:Issue:       Incorrect agent token or certs

--------------

:Issue: Missing agent token or certs

--------------

:Issue: Kea Ctrl Agent uses basic auth, but Stork Agent no

--------------

:Issue: KCA uses BA, but SA has no password

--------------

:Issue: KCA uses BA, but SA has wrong password

--------------

:Issue: KCA requires valid certs, but SA uses these ones from Server

--------------

:Issue: SA uses link-local address

--------------

:Issue: SA has invalid server address

--------------

:Issue: SA has invalid server token

--------------

:Issue: After provide the SA changes nothing happens

--------------

:Issue: SA runned by hand uses wrong configuration
