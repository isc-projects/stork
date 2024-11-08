.. _glossary:

Glossary
========

   +-----------------------+----------------------------------------------------------------+
   | Term                  | Definition                                                     |
   +=======================+================================================================+
   | app                   | A program monitored by the Stork server via the Stork agent.   |
   |                       | Typically, it is one of the integrated servers (e.g., Kea      |
   |                       | DHCP or BIND 9 DNS). An app may comprise multiple daemons.     |
   |                       | For example, the Kea DHCP app contains DHCPv4 and DHCPv6       |
   |                       | daemons.                                                       |
   +-----------------------+----------------------------------------------------------------+
   | app ID                | The unique identifier of a monitored app in the Stork server   |
   |                       | database. This identifier is often displayed in the UI and can |
   |                       | be used, for example, for filtering apps.                      |
   +-----------------------+----------------------------------------------------------------+
   | authorized machine    | A machine running the Stork agent that has requested           |
   |                       | registration on the Stork server, and whose request has been   |
   |                       | approved by the system administrator in the Stork UI.          |
   +-----------------------+----------------------------------------------------------------+
   | configured subnet ID  | See "Kea subnet ID."                                           |
   +-----------------------+----------------------------------------------------------------+
   | daemon                | One of the programs belonging to an app. For example: a DHCPv4 |
   |                       | or D2 daemon in the Kea app.                                   |
   +-----------------------+----------------------------------------------------------------+
   | global configuration  | The set of parameters and DHCP options of a Kea configuration  |
   |                       | that apply to all subnets, shared networks, or host            |
   |                       | reservations on a given Kea server, unless overridden at lower |
   |                       | (non-global) configuration levels.                             |
   +-----------------------+----------------------------------------------------------------+
   | global parameter      | A parameter of the global configuration other than DHCP        |
   |                       | options.                                                       |
   +-----------------------+----------------------------------------------------------------+
   | global DHCP option    | A DHCP option specified within a global configuration of a     |
   |                       | Kea instance.                                                  |
   +-----------------------+----------------------------------------------------------------+
   | high availability (HA)| A failure resiliency mechanism implemented in Kea that         |
   |                       | relies on the presence of multiple cooperating DHCP servers.   |
   |                       | In the event of a server failure on one of the partners,       |
   |                       | another server is able to respond to the DHCP client traffic   |
   |                       | normally handled by the partner server. The Stork              |
   |                       | server can monitor failures on the servers participating       |
   |                       | in an HA setup and present detailed information about          |
   |                       | the states of these servers.                                   |
   +-----------------------+----------------------------------------------------------------+
   | hook library          | A software library (plugin) that extends the Kea server's      |
   |                       | capabilities and can be used by Stork to provide more          |
   |                       | detailed diagnostics of the Kea server or to gain more control |
   |                       | over its operation. For example, the ``subnet_cmds`` hook      |
   |                       | library allows Stork to manage subnets in a Kea instance.      |
   +-----------------------+----------------------------------------------------------------+
   | host reservation      | A part of the Kea server configuration that associates certain |
   |                       | resources (e.g., IP addresses) with a given DHCP               |
   |                       | client. Stork allows for creating host reservations in the Kea |
   |                       | instances.                                                     |
   +-----------------------+----------------------------------------------------------------+
   | Kea server            | The Kea DHCP daemon or app, depending on the context.          |
   +-----------------------+----------------------------------------------------------------+
   | Kea subnet ID         | A subnet identifier specified in the Kea configuration file,   |
   |                       | sometimes called the "configured subnet ID." Distinct from     |
   |                       | the "subnet ID."                                               |
   +-----------------------+----------------------------------------------------------------+
   | machine               | A physical or virtual system running the Stork agent software  |
   |                       | and communicating with the Stork server. It typically also runs|
   |                       | one or more apps (e.g., the Kea or BIND 9 app) that the Stork  |
   |                       | server monitors.                                               |
   +-----------------------+----------------------------------------------------------------+
   | machine ID            | The unique identifier of an authorized or unauthorized machine |
   |                       | in the Stork database. This identifier is often displayed in   |
   |                       | the UI and can be used for, for example, filtering machines.   |
   +-----------------------+----------------------------------------------------------------+
   | puller                | A mechanism in the Stork server that periodically fetches      |
   |                       | data from the Stork agents or the apps behind them.            |
   |                       | The puller interval is configurable, allowing different data   |
   |                       | refresh intervals to be specified for different data types.    |
   +-----------------------+----------------------------------------------------------------+
   | service               | One of the functions provided by one or more monitored apps.   |
   |                       | For example: Kea provides a DHCP service to the DHCP clients.  |
   |                       | BIND 9 provides DNS service. Multiple Kea servers can          |
   |                       | provide High Availability service to DHCP clients.             |
   +-----------------------+----------------------------------------------------------------+
   | shared network        | A group of subnets sharing configuration parameters and        |
   |                       | constituting a single address space from which clients         |
   |                       | in the same network segment can be allocated a DHCP lease.     |
   +-----------------------+----------------------------------------------------------------+
   | Stork agent           | One of the Stork programs, installed on the same               |
   |                       | machine as monitored apps. It detects apps running on the      |
   |                       | machine, registers the machine in the Stork server, and        |
   |                       | serves as an intermediary between the Stork server and the     |
   |                       | apps. There may be many Stork agents in a Stork deployment.    |
   +-----------------------+----------------------------------------------------------------+
   | Stork hook            | A library (plugin) that can be attached to the Stork server,   |
   |                       | extending its capabilities (e.g. authorization with LDAP).     |
   +-----------------------+----------------------------------------------------------------+
   | Stork server          | One of the Stork programs and a central instance of the Stork  |
   |                       | deployment. It coordinates monitoring of the connected         |
   |                       | machines and exposes the grahical user interface to users.     |
   |                       | It also pulls and maintains the information from the machines, |
   |                       | and stores the information in the local database.              |
   +-----------------------+----------------------------------------------------------------+
   | subnet                | A part of the configuration in Kea that defines address space  |
   |                       | and other parameters assigned to the DHCP clients in a         |
   |                       | given network segment. Several servers may include             |
   |                       | configuration of the same subnet (e.g., in the high            |
   |                       | availability setup).                                           |
   +-----------------------+----------------------------------------------------------------+
   | subnet ID             | The unique identifier of a subnet in the Stork database.       |
   |                       | This term is often confusing, because Kea instances also use   |
   |                       | this term for the identifiers within their subnet              |
   |                       | configurations (see "configured subnet ID" and                 |
   |                       | "Kea subnet ID"). The Stork server stores the subnet           |
   |                       | information from one or more Kea instances and processes it    |
   |                       | into a single subnet entry within the Stork database. This     |
   |                       | entry comes with a unique ID, the "subnet ID." Kea subnet IDs  |
   |                       | for that subnet are also stored in the Stork database, but     |
   |                       | they are distinct from the "subnet ID". In fact, each Kea      |
   |                       | server may use a different Kea subnet ID for the same subnet;  |
   |                       | hence a distinct, unique identifier is required in Stork.      |
   +-----------------------+----------------------------------------------------------------+
   | unauthorized machine  | A machine running the Stork agent that has requested           |
   |                       | registration in the Stork server, but this request has not     |
   |                       | yet been approved by the system administrator in the Stork UI. |
   |                       | The machine is unauthorized until the request is approved.     |
   +-----------------------+----------------------------------------------------------------+