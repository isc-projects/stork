.. _glossary:

Glossary
========

   +-----------------------+----------------------------------------------------------------+
   | Term                  | Definition                                                     |
   +=======================+================================================================+
   | app                   | A program monitored by the Stork server via Stork agent.       |
   |                       | Typically, it is one of the integrated servers (e.g., Kea      |
   |                       | DHCP or BIND 9 DNS). An app may comprise multiple daemons.     |
   |                       | For example, Kea DHCP app contains DHCPv4 and DHCPv6 daemons.  |
   +-----------------------+----------------------------------------------------------------+
   | app ID                | A unique identifier of a monitored app in the Stork server     |
   |                       | database. This identifier is often displayed in the UI and can |
   |                       | be, for example, used for filtering apps.                      |
   +-----------------------+----------------------------------------------------------------+
   | authorized machine    | A machine running Stork agent that requested registration in   |
   |                       | the Stork server, and this request was approved by the system  |
   |                       | administrator in the Stork UI.                                 |
   +-----------------------+----------------------------------------------------------------+
   | configured subnet ID  | The same as Kea subnet ID.                                     |
   +-----------------------+----------------------------------------------------------------+
   | daemon                | One of the programs belonging to an app. For example: a DHCPv4 |
   |                       | or D2 daemon in the Kea app.                                   |
   +-----------------------+----------------------------------------------------------------+
   | global configuration  | Parts of the Kea configuration that are not specific to        |
   |                       | subnets, shared networks or hosts. They include global         |
   |                       | parameters and global DHCP options. They affect all subnets,   |
   |                       | shared networks and host reservations in a given Kea server,   |
   |                       | unless overridden at lower (non-global) configuration levels.  |
   +-----------------------+----------------------------------------------------------------+
   | global parameter      | A part of the global Kea configuration other than DHCP option. |
   +-----------------------+----------------------------------------------------------------+
   | global DHCP option    | A DHCP option specified within a global configuration of a     |
   |                       | Kea instance.                                                  |
   +-----------------------+----------------------------------------------------------------+
   | high availability (HA)| A failure resiliency mechanism implemented in Kea that         |
   |                       | relies on a presence of multiple cooperating DHCP servers. The |
   |                       | servers are capable of taking over responsibility for          |
   |                       | responding to the DHCP client traffic normally handled by the  |
   |                       | partner server, in case of a partner server failure. Stork     |
   |                       | server can monitor the failures of the servers participating   |
   |                       | in the HA setup, and present the detailed information about    |
   |                       | the state of these servers.                                    |
   +-----------------------+----------------------------------------------------------------+
   | hook library          | A software library (plugin) that extends Kea server's          |
   |                       | capabilities and can be used by Stork to provide more          |
   |                       | detailed diagnostics of the Kea server or gain more control    |
   |                       | over its operation. For example: subnet_cmds hook library      |
   |                       | allows for subnet management in the Kea instance. If this hook |
   |                       | library is not enabled, Stork cannot manage the subnet         |
   |                       | configuration in that instance.                                |
   +-----------------------+----------------------------------------------------------------+
   | host reservation      | A part of the Kea server configuration, associating certain    |
   |                       | resources (e.g., IP address assignment) with a given DHCP      |
   |                       | client. Stork allows for creating host reservations in the Kea |
   |                       | instances.                                                     |
   +-----------------------+----------------------------------------------------------------+
   | Kea server            | Kea DHCP daemon or app (depending on the context).             |
   +-----------------------+----------------------------------------------------------------+
   | Kea subnet ID         | A subnet identifier specified in the Kea configuration file.   |
   |                       | It is sometimes called "configured subnet ID". It is different |
   |                       | identifier than "subnet ID" described in this glossary.        |
   +-----------------------+----------------------------------------------------------------+
   | machine               | A physical or virtual system running Stork agent software and  |
   |                       | communicating with the Stork server. It typically also runs    |
   |                       | one or more apps (e.g., Kea or BIND 9 app) that the Stork      |
   |                       | server monitors.                                               |
   +-----------------------+----------------------------------------------------------------+
   | machine ID            | A unique identifier of an authorized or unauthorized machine   |
   |                       | in the Stork database. This identifier is often displayed in   |
   |                       | the UI and can be, for example, used for filtering machines.   |
   +-----------------------+----------------------------------------------------------------+
   | puller                | A mechanism in the Stork server periodically fetching certain  |
   |                       | kind of data from the Stork agents or the apps behind them.    |
   |                       | The puller interval is configurable, allowing for specifying   |
   |                       | different data refresh intervals for different data types.     |
   +-----------------------+----------------------------------------------------------------+
   | shared network        | A group of subnets sharing configuration parameters and        |
   |                       | constituting a single address space from which clients         |
   |                       | in the same network segment can be allocated a DHCP lease.     |
   +-----------------------+----------------------------------------------------------------+
   | Stork agent           | One of the Stork programs which is installed on the same       |
   |                       | machine as monitored apps. It detects apps running on the      |
   |                       | machine, registers the machine in the Stork server, and        |
   |                       | serves as an intermediary between the Stork server and the     |
   |                       | apps. There may be many Stork agents in a Stork deployment.    |
   +-----------------------+----------------------------------------------------------------+
   | Stork hook            | A library (plugin) that can be attached to the Stork server,   |
   |                       | extending its capabilities (e.g., authorization with LDAP).    |
   +-----------------------+----------------------------------------------------------------+
   | Stork server          | One of the Stork programs, a central instance of the Stork     |
   |                       | deployment. It coordinates monitoring of the connected         |
   |                       | machines and exposes the grahical user interface to the users. |
   |                       | It also pulls and maintains the information from the machines, |
   |                       | and stores the information in the local database.              |
   +-----------------------+----------------------------------------------------------------+
   | subnet                | A part of the configuration in Kea that defines address space  |
   |                       | and other configuration assigned to the DHCP clients in a      |
   |                       | given network segment. Several servers may include             |
   |                       | configuration of the same subnet (e.g., in the high            |
   |                       | availability setup).                                           |
   +-----------------------+----------------------------------------------------------------+
   | subnet ID             | A unique identifier of a subnet in the Stork database.         |
   |                       | This term is often confusing because Kea instances also use    |
   |                       | this term for the identifiers within their subnet              |
   |                       | configurations (see "configured subnet ID" and                 |
   |                       | "Kea subnet ID"). Stork server stores the subnet information   |
   |                       | from one or more Kea instances, combines this information into |
   |                       | a single subnet entry within the Stork database. This entry    |
   |                       | comes with a unique ID ("subnet ID"). Kea subnet IDs for that  |
   |                       | subnet are also stored in the Stork database but they are      |
   |                       | different than the "subnet ID". In fact, each Kea server may   |
   |                       | use different Kea subnet ID for the same subnet. Hence a       |
   |                       | distinct, unique identifier is required in Stork.              |
   +-----------------------+----------------------------------------------------------------+
   | unauthorized machine  | A machine running Stork agent that requested registration in   |
   |                       | the Stork server, and this request has not been yet approved   |
   |                       | by the system administrator in the Stork UI. The machine       |
   |                       | unauthorized until the request is approved.                    |
   +-----------------------+----------------------------------------------------------------+