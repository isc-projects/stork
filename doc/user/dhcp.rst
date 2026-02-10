.. _dhcp:

****
DHCP
****

Kea DHCP Integration with Stork
===============================

Prior to version 3.0.0, Kea exposed the control API via the Kea Control Agent
daemon. Communication with the other Kea daemons was routed through that daemon.
From version 3.0.0 onward, each Kea daemon (DHCP4, DHCP6, D2, and NETCONF)
provides a direct control channel which works without the Kea Control Agent. As
a result, the Kea Control Agent is deprecated as of Kea 3.0.0

The Stork agent can use that direct channel to communicate with each Kea daemon.
Stork will continue to support communication with pre-3.0.0 Kea daemons using
the Kea Control Agent until those Kea versions have reached the end of support.

Kea instance detection begins by looking for the ``kea-ctrl-agent``,
``kea-dhcp4``, ``kea-dhcp6``, and ``kea-d2`` processes, which are expected to
run with the ``-c`` parameter specifying the path to their configuration files.
The Stork agent then reads the following configuration parameters from each file:

- The listening control socket (``http-host`` and ``http-port`` for CA, and
  ``control-socket`` or ``control-sockets`` for other daemons)
- The list of controlled daemons (for CA only -  ``control-sockets``)
- The HTTP socket authorization credentials (``authentication``)

The Stork agent uses the first control socket which it is able to connect to. It
supports both UNIX domain and network sockets.

.. note:: The control socket displayed in the Stork UI may not be the same
   control socket that the agent is using to communicate with Kea.The UI
   presents only the first specified control socket from the Kea configuration.
   If the first socket is not reachable, the agent will try the next one, and
   so on, until a successful connection is made. The active control socket will
   be reflected in the UI in a future release.

Subnets and Networks
====================

IPv4 and IPv6 Subnets per Kea Daemons
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

One of the primary configuration aspects of any network is the layout
of IP addressing. This is represented in Kea with IPv4 and IPv6
subnets. Each subnet represents addresses used on a physical
link. Typically, certain parts of each subnet ("pools") are delegated
to the DHCP server to manage. Stork is able to display this
information.

One way to inspect the subnets and pools within Kea is by looking at
each Kea daemon to get an overview of the configurations a
specific Kea daemon is serving. A list of configured subnets on
that specific Kea daemon is displayed. The following picture
shows a simple view of the Kea DHCPv6 server running with a single
subnet, with three pools configured in it.

.. figure:: ./static/kea-subnets6.png
   :alt: View of subnets assigned to a single Kea daemon

IPv4 and IPv6 Subnets in the Whole Network
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

It is convenient to see a complete overview of all subnets
configured in the network that are being monitored by Stork. Once at least one
machine with the Kea daemon running is added to Stork, click on
the ``DHCP`` menu and choose ``Subnets`` to see all available subnets. The
view shows all IPv4 and IPv6 subnets, with the address pools and links
to the daemons that are providing them. An example view of all
subnets in the network is presented in the figure below.

.. figure:: ./static/kea-subnets-list.png
   :alt: List of all subnets in the network

Stork provides filtering capabilities; it is possible to
choose to see IPv4 only, IPv6 only, or both. There is also an
omnisearch box available where users can type a search string.
For strings of four characters or more, the filtering takes place
automatically, while shorter strings require the user to hit
Enter. In the above example, it is possible to show only
the first (192.0.2.0/24) subnet by searching for the *0.2* string. One
can also search for specific pools, and easily filter the subnet with
a specific pool, by searching for part of the pool range,
e.g. *3.200*. The input box accepts a text string that can be a part of the
subnet or shared network name.

Stork displays pool utilization for each subnet, with
the absolute number of addresses allocated and usage percentage.
There are two thresholds: 80% (warning; the pool utilization
bar turns orange) and 90% (critical; the pool utilization bar
turns red).

Subnet Names
~~~~~~~~~~~~

Kea allows storing any arbitrary data related to a subnet in the ``user-context``
field. This field is a JSON object. It may be used to store some metadata about
the subnet, such as the name of the location where the subnet is used, the name
of the department, name of related service or any other information that is
useful for the network administrator.

Stork displays the subnet's user context on the subnet page. Additionally, the
value of the ``subnet-name`` key is displayed in the subnet list view. This
allows the network administrator to quickly identify the subnet by its name.

The subnet name can be used to filter the subnets on the subnet list page and
in the global search box.

IPv4 and IPv6 Networks
~~~~~~~~~~~~~~~~~~~~~~

Kea uses the concept of a shared network, which is essentially a stack
of subnets deployed on the same physical link. Stork
retrieves information about shared networks and aggregates it across all
configured Kea servers. The ``Shared Networks`` view allows the
inspection of networks and the subnets that belong in them. Pool
utilization is shown for each subnet.

.. _creating-subnets:

Creating Subnets
~~~~~~~~~~~~~~~~

Stork can configure new subnets in Kea instances with the Subnet Commands (``subnet_cmds``)
hook library loaded. Navigate to ``DHCP -> Subnets`` to display the subnets list, and click
the ``New Subnet`` button. The opened form initially contains only an input box where
a subnet prefix must be specified. It can be an IPv4 address (e.g., ``192.0.2.0/24``) or
IPv6 prefix (e.g., ``2001:db8:1::/64``). Click the ``Proceed`` button to expand the
form and enter the remaining subnet configuration information.

The Stork subnet form allows the user to specify a common subnet configuration that
can be instantly populated to multiple DHCP servers. Configuring the same subnet in
multiple Kea instances is specific to the deployments where service redundancy is
required (e.g. deployments using High Availability or with a shared lease database).
When configuring a new subnet it is possible to select multiple DHCP servers
in the ``Assignments`` panel, and the subnet is populated to these servers. Please
note that the list of servers only contains those matching the subnet prefix
(IPv4 or IPv6). Additionally, only servers running the ``subnet_cmds`` hook library
are listed.

The new subnet may be assigned to a shared network in the ``Subnet`` panel. The Shared
Network dropdown list may be empty for two reasons:

- There are no shared networks in the selected Kea instances.
- Some Kea instances selected for the subnet lack a shared-network specification.

If there are no shared networks, simply create one before creating the subnet.
If the shared-network specification is absent, update the shared network and assign it to all servers
to which the subnet will be assigned. As an example, suppose we want to add a new subnet and assign
it to both ``server 1`` and ``server 2``. If this subnet is currently only on the shared
network that is assigned to ``server 1``, we must first edit the shared network and add its
assignment to ``server 2``. Then we can create a new subnet and assign it to both
``server 1`` and ``server 2``, and the shared networks list should now contain our shared network.
Select this shared network from the list in the subnet form.

Once a shared network is selected, subnet assignments cannot be changed. To
change an assignment, first unassign the subnet from the shared network by clicking the
X button to the right of the selected shared network name. Once the shared network
has been removed, the subnet assignments can now be changed.

The subnet usually comes with one or more address pools (both IPv4 and IPv6), and it may
also contain delegated prefix pools (IPv6 only). The DHCP servers assign leases
to the clients from the resources available in these pools. The address pool boundaries
are specified as a pair of addresses (i.e. first and last address). Both addresses
must match the subnet prefix (i.e. must be within this subnet), and the first address must be
lower than or equal to the last address. If the first and last addresses are the same, the
pool contains exactly one address. Empty pools are not allowed.

In some deployments, multiple DHCP servers can share the same subnets but may
include different pools. In this scenario, administrators can avoid the conflict
whereby two servers offer the same address (from overlapping pools) to different
clients. Stork allows the assignment of a pool to a subset
of the DHCP servers assigned to the subnet. If the pool should be included in
all servers, pick all servers in the pool's ``Assignments`` panel. Note that, in addition to
specifying the pool boundaries and assignments, each expandable pool panel also
allows the specification of some pool-level configuration parameters,
such as ``Client Class`` and ``Pool ID``. It is also possible to specify pool-level
DHCP options.

Create more pools as needed using the ``Add Pool`` button. Click ``Delete Pool``
to remove a selected pool from the subnet.

Delegated prefix pools can be added for IPv6 subnets. The delegated prefix pool
boundaries are specified differently than the address pool boundaries; also, the
delegated prefix pool prefix does not have to match (belong to) the subnet prefix.
The delegated prefix pool comprises an actual prefix (e.g. ``3000::/64``) and
a delegated prefix length (e.g. ``96``). The delegated prefix length must be
greater than or equal to the prefix length; in the examples above, ``96 > 64``. If they are
equal, the delegated prefix pool contains exactly one prefix.

`RFC 6603 <https://www.rfc-editor.org/rfc/rfc6603.html>`_ describes the mechanism
to exclude one specific prefix from a delegated prefix set in DHCPv6.
This prefix can be optionally specified as an ``Excluded Prefix`` for a delegated
prefix pool. This prefix must belong to the delegated prefix and its length must be
greater than the delegated prefix length.

The Kea subnet configuration contains ``DHCP Parameters`` which include different
aspects of lease assignment in that subnet. By default, each DHCP server in the
subnet gets the same values of the DHCP parameters. In some cases, however, an
administrator can choose to specify different values for the same parameter on
different servers. Checking the ``Unlock`` box for specific parameters splits
the form for these parameters, so different values can be specified for different
servers in the input boxes.

The ``DHCP Options`` panel allows specified DHCP options to be returned to
the clients connected to the subnet. In most cases, these options are common
for different servers assigned to the subnet. However, it is possible to differentiate
some options using a mechanism similar to the one described above for ``DHCP Parameters``.
Click ``Unlock setting DHCP options for individual servers`` and set the respective option
sets for different servers.

Each DHCP option specification begins with the selection of the option code from the dropdown
list. The input boxes displayed below the option code represent the option fields carried
by the option. Fill in these fields with values appropriate for the option.

If a DHCP option carries an array of fields, only the input box for the first field
is initially displayed. To add more fields to the array, expand the dropdown list
below the option code selector and select the correct option field type to
be added to the array. The option fields and the options can also be removed from
the form.

When the subnet form is complete, click the ``Submit`` button to save
the subnet and send it to the Kea servers. The ``Submit`` button is disabled if
the form has any invalid entries.

Updating Subnets
~~~~~~~~~~~~~~~~

To update an existing subnet configuration, click on the subnet in the dashboard
or in the subnets list to display detailed information about the subnet.
Click the ``Edit`` button to open the subnet update form. Note that only subnets
on servers with the ``subnet_cmds`` hook library loaded can
be updated.

Subnet configuration is described in detail in the :ref:`creating-subnets` section.
Here, we focus on the process of updating a subnet.

A subnet prefix cannot be modified for an updated subnet. To increase
or decrease a subnet prefix length, simply create a new subnet and delete the
existing one.

If a shared network field is cleared for the updated subnet, this subnet is
removed from the shared network on the Kea servers. If another shared network
is selected instead, the subnet is first removed from the existing shared
network and then added to the newly selected shared network.

A pool can be deleted from a subnet; however, it is important to understand the
ramifications. While the pool itself is removed from the configuration instantly,
the leases allocated in this pool are not. Kea maintains these leases in the lease
database and clients continue using these leases, until the leases expire or
the clients attempt to renew them. Lease extensions from the deleted pools are
refused to renewing clients; they will be allocated new leases from
the existing pools.

Use the ``Revert Changes`` button to remove all edits and restore
the original subnet information. Use ``Cancel`` to close the page
without applying any changes.

Deleting Subnets
~~~~~~~~~~~~~~~~

To delete a subnet from Stork and the Kea instances, navigate to the subnet view
from the dashboard or the subnets list and select the desired subnet. Click the
``Delete`` button and confirm the removal of the subnet from all Kea instances.
Deleting a subnet requires the Kea servers with the subnet to have
the ``subnet_cmds`` hook library loaded.

Creating Shared Networks
~~~~~~~~~~~~~~~~~~~~~~~~

Stork can configure new shared networks in the Kea instances with the ``subnet_cmds``
hook libraries. The shared networks group subnets with common configuration parameters,
and provide a common address space for the DHCP clients connected to different
subnets. To create a shared network, navigate to the shared networks list (``DHCP -> Shared Networks``) and click
the ``New Shared Network`` button.

A shared network must be assigned to one or more DHCP servers selected in the ``Assignments``
panel. All servers must be of the same kind (DHCPv4 or DHCPv6), so after selecting
the first server the list is limited to other servers of the same kind. The shared network
is created in all of the selected Kea servers.

A shared network name is mandatory. It is an arbitrary value that must be unique among
the servers connected to Stork.

The ``DHCP Parameters`` and ``DHCP Options`` specified for the shared network are common
for all subnets later added to this shared network. However, these parameters and options
specified at the subnet level override the common shared network-level values.

Similarly to :ref:`creating-subnets`, it is possible to unlock selected parameters and
options, and to specify different values for different servers holding the shared network
configuration.

When the form is ready, click the ``Submit`` button to create the shared network in Stork and
the Kea instances. This button is disabled if
the form has any invalid entries.

Updating Shared Networks
~~~~~~~~~~~~~~~~~~~~~~~~

To update an existing shared network configuration, click on the shared network in the dashboard
or in the shared networks list to display detailed information about the shared network.
Click the ``Edit`` button to open the shared-network update form. Note that only shared networks
on servers with the ``subnet_cmds`` hook library loaded can
be updated.

Removing the shared network from a server (in the ``Assignments`` panel) also removes
the subnets belonging to this shared network from the server. They are added back
when the server is added to the shared network.

Update the shared network as needed and click ``Submit`` to save the changes in
Stork and in the Kea instances.

Deleting Shared Networks
~~~~~~~~~~~~~~~~~~~~~~~~

To delete a shared network from Stork and the Kea instances, navigate to the shared networks view
from the dashboard or the shared networks list and select the desired shared network. Click the
``Delete`` button and confirm the removal of the shared network from all Kea instances.
Deleting a shared network requires the Kea servers with the shared network to have
the ``subnet_cmds`` hook library loaded.

Deleting a shared network also deletes all subnets it includes. To
preserve the subnets from the deleted shared network, click on each subnet
belonging to it, edit the subnet, clear the shared network selection in the
``Subnet`` panel, and save the subnet changes before deleting the empty shared network.

Host Reservations
=================

Listing Host Reservations
~~~~~~~~~~~~~~~~~~~~~~~~~

Kea DHCP servers can be configured to assign static resources or parameters to the
DHCP clients communicating with the servers. Most commonly these resources are the
IP addresses or delegated prefixes; however, Kea also allows assignment of hostnames,
PXE boot parameters, client classes, DHCP options, and other parameters. The mechanism by which
a given set of resources and/or parameters is associated with a given DHCP client
is called "host reservations."

A host reservation consists of one or more DHCP identifiers used to associate the
reservation with a client, e.g. MAC address, DUID, or client identifier;
and a collection of resources and/or parameters to be returned to the
client if the client's DHCP message is associated with the host reservation by one
of the identifiers. Stork can detect existing host reservations specified both in
the configuration files of the monitored Kea servers and in the host database
backends accessed via the Kea Host Commands hook library.

All reservations detected by Stork can be listed by selecting the ``DHCP``
menu option and then selecting ``Host Reservations``.

The first column in the presented view displays one or more DHCP identifiers
for each host in the format ``hw-address=0a:1b:bd:43:5f:99``, where
``hw-address`` is the identifier type. In this case, the identifier type is
the MAC address of the DHCP client for which the reservation has been specified.
Supported identifier types are described in the following sections of the Kea
Administrator Reference Manual (ARM):
`Host Reservations in DHCPv4 <https://kea.readthedocs.io/en/latest/arm/dhcp4-srv.html#host-reservations-in-dhcpv4>`_
and `Host Reservations in DHCPv6 <https://kea.readthedocs.io/en/latest/arm/dhcp6-srv.html#host-reservations-in-dhcpv6>`_.

The next two columns contain the static assignments of the IP addresses and/or
prefixes delegated to the clients. There may be one or more such IP reservations
for each host.

The ``Hostname`` column contains an optional hostname reservation, i.e., the
hostname assigned to the particular client by the DHCP servers via the
Hostname or Client FQDN option.

The ``Global/Subnet`` column contains the prefixes of the subnets to which the reserved
IP addresses and prefixes belong. If the reservation is global, i.e., is valid
for all configured subnets of the given server, the word "global" is shown
instead of the subnet prefix.

Finally, the ``Daemon Name`` column includes one or more links to
Kea daemons configured to assign each reservation to the
client. The number of daemons is typically greater than one
when Kea servers operate in the High Availability setup. In this case,
each of the HA peers uses the same configuration and may allocate IP
addresses and delegated prefixes to the same set of clients, including
static assignments via host reservations. If HA peers are configured
correctly, the reservations they share will have two links in the
``Daemon Name`` column. Next to each link there is a label indicating
whether the host reservation for the given server has been specified
in its configuration file or a host database (via the Host Commands
hook library).

The ``Filter Hosts`` input box is located above the ``Hosts`` table. It
allows hosts to be filtered by identifier types, identifier values, IP
reservations, and hostnames, and by globality, i.e., ``is:global`` and ``not:global``.
When filtering by DHCP identifier values, it is not necessary to use
colons between the pairs of hexadecimal digits. For example, the
reservation ``hw-address=0a:1b:bd:43:5f:99`` will be found
whether the filtering text is ``1b:bd:43`` or ``1bbd43``.


Host Reservation Usage Status
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

Clicking on a selected host in the host reservations list opens a new tab
that shows host details. The tab also includes information about
reserved address and delegated prefix usage. Stork needs to query the Kea
servers to gather the lease information for each address and prefix in the
selected reservation; it may take several seconds or longer before this
information is available. The lease information can be refreshed using the
``Leases`` button at the bottom of the tab.

The usage status is shown next to each IP address and delegated prefix.
Possible statuses and their meanings are listed in the table below.

.. table:: Possible IP reservation statuses
   :widths: 10 90

   +-----------------+---------------------------------------------------------------+
   | Status          | Meaning                                                       |
   +=================+===============================================================+
   | ``in use``      | There are valid leases assigned to the client. The client     |
   |                 | owns the reservation, or the reservation includes the         |
   |                 | ``flex-id`` or ``circuit-id`` identifier, making it impossible|
   |                 | to detect conflicts (see note below).                         |
   +-----------------+---------------------------------------------------------------+
   | ``expired``     | At least one of the leases assigned to the client owning      |
   |                 | the reservation is expired.                                   |
   +-----------------+---------------------------------------------------------------+
   | ``declined``    | The address is declined on at least one of the Kea servers.   |
   +-----------------+---------------------------------------------------------------+
   | ``in conflict`` | At least one of the leases for the given reservation is       |
   |                 | assigned to a client that does not own this reservation.      |
   +-----------------+---------------------------------------------------------------+
   | ``unused``      | There are no leases for the given reservation.                |
   +-----------------+---------------------------------------------------------------+

View status details by expanding a selected address or delegated prefix row.
Clicking on the selected address or delegated prefix navigates to the leases
search page, where all leases associated with the address or prefix can be
listed.

.. note::

   Detecting ``in conflict`` status is currently not supported for host
   reservations with the ``flex-id`` or ``circuit-id`` identifiers. If there are
   valid leases for such reservations, they are marked ``in use`` regardless
   of whether the conflict actually exists.

Sources of Host Reservations
~~~~~~~~~~~~~~~~~~~~~~~~~~~~

There are two ways to configure Kea servers to use host reservations. First,
the host reservations can be specified within the Kea configuration files; see
`Host Reservations in DHCPv4 <https://kea.readthedocs.io/en/latest/arm/dhcp4-srv.html#host-reservations-in-dhcpv4>`_
for details. The other way is to use a host database backend, as described in
`Storing Host Reservations in MySQL or PostgreSQL <https://kea.readthedocs.io/en/latest/arm/dhcp4-srv.html#storing-host-reservations-in-mysql-or-postgresql>`_.
The second solution requires the given Kea server to be configured to use the
Host Commands hook library (``host_cmds``). This library implements control commands used
to store and fetch the host reservations from the host database to which the Kea
server is connected. If the ``host_cmds`` hook library is not loaded, Stork
only presents the reservations specified within the Kea configuration files.

Stork periodically fetches the reservations from the host database backends
and updates them in the local database. The default interval at which Stork
refreshes host reservation information is set to 60 seconds. This means that
an update in the host reservation database is not visible in Stork until
up to 60 seconds after it was applied. This interval is configurable in the
Stork interface.

.. note::

   The list of host reservations must be manually refreshed by reloading the
   browser page to see the most recent updates fetched from the Kea servers.

Creating Host Reservations
~~~~~~~~~~~~~~~~~~~~~~~~~~

Navigate to ``DHCP -> Host Reservations`` to view the list of host reservations.
Clicking the ``New Host`` button opens a tab where a new
host reservation can be specified on one or more Kea servers. These Kea servers must be
configured to use the Host Commands hooks library; only servers with ``host_cmds``
loaded are available for selection in the ``DHCP Servers`` dropdown.

Both subnet-level and global host reservations can be created. Setting the
``Global reservation`` option disables subnet selection. Use the ``Subnet``
dropdown to select a subnet-level reservation. If the desired subnet is
not displayed in the dropdown, the selected DHCP servers may not include this
subnet in their configuration.

To associate the new host reservation with a DHCP client, select
one of the identifier types supported by Kea; the available identifiers vary
depending on whether the selected servers are running DHCPv4 or DHCPv6. The identifier
can be specified using ``hex`` or ``text`` format. For example, the ``hw-address``
is typically specified as a string of hexadecimal digits, such as ``ab:76:54:c6:45:31``.
In that case, select the ``hex`` option. Some identifiers, e.g. ``circuit-id``, are
often specified using "printable characters," e.g. ``circuit-no-1``. In that case,
select the ``text`` option. Please refer to
`Host Reservations in DHCPv4 <https://kea.readthedocs.io/en/latest/arm/dhcp4-srv.html#host-reservations-in-dhcpv4>`_
and `Host Reservations in DHCPv6 <https://kea.readthedocs.io/en/latest/arm/dhcp6-srv.html#host-reservations-in-dhcpv6>`_
for more details regarding the allowed DHCP identifiers and their formats.

Next, specify the actual reservations. It is possible
to specify at most one IPv4 address, but multiple IPv6 addresses and delegated prefixes
can be indicated.

The DHCPv4 ``siaddr``, ``sname``, and ``file`` fields can be statically assigned to
clients using host reservations. The relevant values in Kea and Stork are
``Next Server``, ``Server Hostname``, and ``Boot File Name``. Those values can only
be set for DHCPv4 servers; when editing a DHCPv6 host, those fields are not available.

It is possible to associate one or more client classes with a host. Kea servers
assign these classes to DHCP packets received from the client that has
the host reservation. Client classes are typically defined in the Kea
configurations, but not always. For example, built-in classes like
``DROP`` have no explicit definitions in configuration files.
Click the ``List`` button to select client classes from the list of
classes explicitly defined in the configurations of the monitored Kea servers.
Select the desired class names and click ``Insert``. If the desired class
name is not on the list, type the class name directly in the
input box and press Enter. Click on the X icon next to the class name
to delete it from the host reservation.

DHCP options can be added to the host reservation by clicking the ``Add Option``
button; the list of standard DHCP options is available via the dropdown.
However, if the list is missing a desired option, simply
type the option code in the dropdown. The ``Always Send`` checkbox specifies
whether the option should always be returned to a DHCP client assigned this
host reservation, regardless of whether the client requests this option from
the DHCP server.

Stork recognizes standard DHCP option formats. After selecting an option
code, the form is adjusted to include option fields suitable for the selected
option. If the option payload comprises an array of option fields, only the
first field (or the first group of the record field) is displayed by default.
Use the ``Add <field-type>`` button below the option code to add more fields
to the array.

.. note::

   Currently, Stork does not verify whether the specified options comply
   with the formats specified in the RFCs, nor does it check them against the
   runtime option definitions configured in Kea. If the wrong option
   format is specified, Stork tries to send the option to Kea for verification,
   but Kea rejects the new reservation. The reservation can be submitted
   again after correcting the option payload.

Use the ``Add <field-type>`` button to add suboptions to a DHCP option.
Stork supports top-level options with a maximum of two levels of suboptions.

If a host reservation is configured on several DHCP servers, all the
servers typically comprise the same set of parameters (i.e. IP addresses, hostname,
boot fields, client classes, and DHCP options). By default, creating a new
host reservation for multiple servers sends an identical copy of the host
reservation to each. It is possible to specify a different set of boot fields,
client classes, or options for different servers by selecting
``Configure individual server values`` at the top of the form. In this case,
specify the complete sets of boot fields, client classes, and options
for each DHCP server. Leaving them blank for some servers means that these
servers receive no boot fields, classes, or DHCP options with the reservation.

Updating Host Reservations
~~~~~~~~~~~~~~~~~~~~~~~~~~

In a selected host reservation's view, click the ``Edit`` button to
edit the host reservation information. The page automatically toggles editing
DHCP options individually for each server (see above) when it detects different
option sets on different servers using the reservation. Besides editing the
host reservation information, it is also possible to deselect some of the
servers (using the DHCP Servers dropdown), which deletes the reservation
from these servers.

Use the ``Revert Changes`` button to remove all edits and restore
the original host reservation information. Use ``Cancel`` to close the page
without applying any changes.

Deleting Host Reservations
~~~~~~~~~~~~~~~~~~~~~~~~~~

To delete a host reservation from all DHCP servers for which it is configured,
click on the reservation in the host reservations list. Click the ``Delete``
button at the bottom of the page and confirm the reservation deletion. Note that this
operation cannot be undone; the reservation is removed from the DHCP servers'
databases. To restore the reservation, it must be re-created.

Migrating Host Reservations
~~~~~~~~~~~~~~~~~~~~~~~~~~~

Stork can migrate host reservations from the Kea JSON configuration file into
the Kea host database. This feature is available on the host list page. The
hosts to be migrated are selected using the list filter. The filter may be
configured to select all hosts from a given subnet, Kea server, or by free text
search. The migration process starts when the ``Migrate`` button is clicked and
it is performed in the background.

The host reservations that reside both in the Kea JSON configuration file and in the
host database and are different from each other (are conflicting) cannot be
migrated. They will be skipped and the migration process will continue with the
remaining host reservations. The user needs to resolve the conflicts manually
to migrate such reservations.

During the migration process, the Stork server stops pulling the data from Kea
and locks the Kea daemons for modification. The lock is released when the
migration process is finished.
Therefore, the changes in the host reservations cannot be immediately seen in
the host reservations list, because the data is not pulled from Kea. Instead,
the migration progress may be monitored in the "Config Migration" page.

If any errors occur during the migration, the summary and list of them are
displayed in the "Config Migration" page. In this case, the user should fix
the errors and re-run the migration process. Also, if the server is shut
down or restarted during the migration, the process may be safely
repeated.

The migration can be interrupted anytime by clicking the ``Cancel`` button.

Stork migrates the host reservations by sending the command to the Kea. The Kea
must be configured to use the ``host_cmds`` hook library. First, the host
reservations are recreated in the host database, and then they are removed from
the JSON configuration. The host reservations are processed in batches of 100
reservations.

The migration process sends the ``config-write`` command at the end of each
batch. It is not recommended to alter the Kea configuration during the
migration process, especially the host reservations should not be modified
or deleted.

Leases
======

Lease Search
~~~~~~~~~~~~

Stork can search DHCP leases on monitored Kea servers, which is helpful
for troubleshooting issues with a particular IP address or delegated prefix.
It is also helpful in resolving lease allocation issues for certain DHCP clients.
The search mechanism utilizes Kea control commands to find leases on the monitored
servers. Operators must ensure that any Kea servers on which they intend to search
the leases have the `Lease Commands hook library <https://kea.readthedocs.io/en/latest/arm/hooks.html#lease-cmds-lease-commands>`_ loaded. Stork cannot search leases on Kea instances without
this library.

The lease search is available via the ``DHCP -> Lease Search`` menu. Enter one
of the searched lease properties in the search box:

- IPv4 address, e.g. ``192.0.2.3``
- IPv6 address or delegated prefix without prefix length, ``2001:db8::1``
- MAC address, e.g. ``01:02:03:04:05:06``
- DHCPv4 Client Identifier, e.g. ``01:02:03:04``
- DHCPv6 DUID, e.g. ``00:02:00:00:00:04:05:06:07``
- Hostname, e.g. ``myhost.example.org``

All identifier types can also be specified using notation with spaces,
e.g. 01 02 03 04 05 06, or notation with hexadecimal digits only, e.g. 010203040506.

To search all declined leases, type ``state:declined`` in the search box. Be aware that this query may
return a large result if there are many declined leases, and thus the query
processing time may also increase.

.. note::

    Kea versions 3.1.1 through 3.1.4 do not support ``state:declined`` queries.
    Prior to Kea 3.1.1, this kind of query was implemented using a mechanism
    which exposed a Kea implementation detail.  In Kea 3.1.1, this exposure was
    corrected. A suitable replacement API was not available until Kea 3.1.5.

Searching using partial text is currently unsupported. For example, searching by
partial IPv4 address ``192.0.2`` is not accepted by the search box. Partial MAC
address ``01:02:03`` is accepted but will return no results. Specify the complete
MAC address instead, e.g. ``01:02:03:04:05:06``. Searching leases in states other
than ``declined`` is also unsupported. For example, the text ``state:expired-reclaimed``
is not accepted by the search box.

The search utility automatically recognizes the specified lease type property and
communicates with the Kea servers to find leases using appropriate commands. Each
search attempt may result in several commands to multiple Kea servers; therefore,
it may take several seconds or more before Stork displays the search results.
If some Kea servers are unavailable or return an error, Stork
shows leases found on the servers which returned a "success" status, and displays a
warning message containing the list of Kea servers that returned an error.

If the same lease is found on two or more Kea servers, the results list contains
all that lease's occurrences. For example, if there is a pair of servers cooperating
via the High Availability hook library, the servers exchange the lease information, and each of them
maintains a copy of the lease database. In that case, the lease search on these
servers typically returns two occurrences of the same lease.

To display the detailed lease information, click the expand button (``>``) in the
first column for the selected lease.

Kea High Availability Status
============================

To check the High Availability (HA) status of a machine, go to the ``Services -> Kea Daemons``
menu. On the Kea Daemons page, click on a machine name in the list and scroll
down to the High Availability section. This information is
periodically refreshed according to the configured interval of the
Kea status puller (see ``Configuration`` -> ``Settings``).

Kea HA supports advanced resilience configurations with one central
server (hub) connected to multiple servers providing DHCP service in
different network segments (spokes). This configuration model is described
in the `Hub and Spoke Configuration section in the Kea ARM
<https://kea.readthedocs.io/en/latest/arm/hooks.html#hub-and-spoke-configuration>`_.
Internally, Kea maintains a separate state machine for each connection between
the hub and a server; we call this state machine a ``relationship``. The
hub has many relationships, and each spoke has a single relationship with the hub.
Stork presents HA status for each relationship separately (e.g., ``Relationship #1``,
``Relationship #2``, etc.). Note that each relationship may be in a different state.
For example: a hub may be in the ``partner-down`` state for ``Relationship #1``
and in the ``hot-standby`` state for ``Relationship #2``. The hub relationship
states depend on the availability of the respective spoke servers.

See the `High Availability section in the
Kea ARM
<https://kea.readthedocs.io/en/latest/arm/hooks.html#libdhcp-ha-so-high-availability-outage-resilience-for-kea-servers>`_
for details about the roles of the servers within the HA setup.

To see more information, click on the arrow button to the left of
each HA relationship to see the status details. The following picture shows a typical
High Availability status view for a relationship.

.. figure:: ./static/kea-ha-status.png
   :alt: High Availability status example


``This Server`` is the DHCP server (daemon)
whose daemon status is currently displayed; the ``Partner`` is its
active HA partner belonging to the same relationship. The partner belongs
to a different Kea instance running on a different machine; this machine may or
may not be monitored by Stork. The statuses of both servers are fetched by sending
the `status-get
<https://kea.readthedocs.io/en/latest/arm/hooks.html#the-status-get-command>`_
command to the Kea server whose details are displayed (``This Server``).
In the load-balancing and hot-standby modes, the server
periodically checks the status of its partner by sending it the
``ha-heartbeat`` command. Therefore, this information is not
always up-to-date; its age depends on the heartbeat command interval
(by default 10 seconds). The status of the partner returned by
Stork includes the age of the displayed status information.

The Stork status information contains the role, state, and scopes
served by each server. In the typical case, both servers are in
load-balancing state, which means that both are serving DHCP
clients. If the ``partner`` crashes, ``This Server`` transitions to
the ``partner-down`` state , which will be indicated in this view.
If ``This Server`` crashes, it will manifest as a communication
problem between Stork and the server.

The High Availability view also contains information about the
heartbeat status between the two servers, and information about
failover progress. The failover progress information is only
presented when one of the active servers has been unable to
communicate with the partner via the heartbeat exchange for a
time exceeding the ``max-heartbeat-delay`` threshold. If the
server is configured to monitor the DHCP traffic directed to the
partner, to verify that the partner is not responding to this
traffic before transitioning to the ``partner-down`` state, the
number of ``unacked`` clients (clients which failed to get a lease),
connecting clients (all clients currently trying to get a lease from
the partner), and analyzed packets are displayed. The system
administrator may use this information to diagnose why the failover
transition has not taken place or when such a transition is likely to
happen.

More about the High Availability status information provided by Kea can
be found in the `Kea ARM
<https://kea.readthedocs.io/en/latest/arm/hooks.html#the-status-get-command>`_.

Viewing the Kea Log
===================

Stork offers a simple log-viewing mechanism to diagnose issues with
monitored daemons.

.. note::

   This mechanism currently only supports viewing Kea log
   files. Monitoring other logging locations such as stdout, stderr,
   or syslog is also not supported.

Kea can be configured to save logs to multiple destinations. Different types
of log messages may be output into different log files: syslog, stdout,
or stderr. The list of log destinations used by the Kea daemon
is available on the ``Kea Daemons`` page: click on a Kea daemon to view its details.
Then, scroll down to the ``Loggers`` section.

This section contains a table with a list of configured loggers for
the selected daemon. For each configured logger, the logger's name,
logging severity, and output location are presented. The possible output
locations are: log file, stdout, stderr, or syslog. Stork can
display log output to log files, and shows a link to the associated
file.
Loggers that send output to stdout, stderr, and syslog are also listed,
but Stork is unable to display them.

Clicking on the selected log file navigates to its log viewer.
By default, the viewer displays the tail of the log file, up to 4000 characters.
Depending on the network latency and the size of the log file, it may take
several seconds or more before the log contents are fetched and displayed.

The log viewer title bar comprises three buttons. The button with the refresh
icon triggers a log-data fetch without modifying the size of the presented
data. Clicking on the ``+`` button extends the size of the viewed log tail
by 4000 characters and refreshes the data in the log viewer. Conversely,
clicking on the ``-`` button reduces the amount of presented data by
4000 characters. Each time any of these buttons is clicked, the viewer
discards the currently presented data and displays the latest part of the
log file tail.

Please keep in mind that extending the size of the viewed log tail may
slow down the log viewer and increase network congestion as
the amount of data fetched from the monitored machine grows.

Viewing the Kea Configuration as a JSON Tree
============================================

Kea uses JavaScript Object Notation (JSON) to represent its configuration
in the configuration files and the command channel. Parts of the Kea
configuration held in the `Configuration Backend <https://kea.readthedocs.io/en/latest/arm/config.html#kea-configuration-backend>`_
are also converted to JSON and returned over the control channel in that
format. The diagnosis of issues with a particular server often begins by
inspecting its configuration.

In the ``Kea Daemons`` view, select the appropriate daemon
to be inspected, and then click on the ``Raw Configuration``
button. The displayed tree view comprises the selected daemon's
configuration fetched using the Kea ``config-get`` command.

.. note::

   The ``config-get`` command returns the configuration currently in use
   by the selected Kea server. It is a combination of the configuration
   read from the configuration file and from the config backend, if Kea uses
   the backend. Therefore, the configuration tree presented in Stork may
   differ (sometimes significantly) from the configuration file contents.

The nodes with complex data types can be individually expanded and
collapsed. All nodes can also be expanded or collapsed by toggling
the ``Expand`` button. When expanding nodes
with many sub-nodes, they may be paginated to avoid degrading browser
performance.

Click the ``Refresh`` button to fetch and display the latest configuration.
Click ``Download`` to download the entire configuration into a text file.

.. note::

   Some configuration fields may contain sensitive data (e.g. passwords
   or tokens). The content of these fields is hidden, and a placeholder is shown.
   Configurations downloaded as JSON files by users other than super-admins contain
   null values in place of the sensitive data.

Configuration Review
====================

Kea DHCP servers are controlled by numerous configuration parameters, and there is a
risk of misconfiguration or inefficient server operation if those parameters
are misused. Stork can help determine typical problems in a Kea server
configuration, using built-in configuration checkers.

Stork generates configuration reports for a monitored Kea daemon when it
detects that the daemon's configuration has changed. To view the reports for the daemon,
navigate to the daemons page and select one of the daemons. The
``Configuration Review Reports`` panel lists issues and proposed configuration
updates generated by the configuration checkers. Each checker focuses on one
particular problem.

If some reports are considered false alarms, it is possible to
disable some configuration checkers for a selected daemon or globally for all
daemons. Click the ``Checkers`` button to open the list of available checkers and
their current state. Click on the values in the ``State`` column for the respective
checkers until they are in the desired states. Besides enabling and disabling
the checker, it is possible to configure it to use the globally specified
setting (i.e., globally enabled or globally disabled). The global settings
control the checker states for all daemons for which explicit states are not
selected.

Select ``Configuration -> Review Checkers`` from the menu bar to modify the
global states. Use the checkboxes in the ``State`` column to modify the global
states for the respective checkers.

The ``Selectors`` listed for each checker indicate the types of daemons whose
configurations they validate:

- ``each-daemon`` - run for all types of daemons
- ``kea-daemon`` - run for all Kea daemons
- ``kea-ca-daemon`` - run for Kea Control Agents
- ``kea-dhcp-daemon`` - run for DHCPv4 and DHCPv6 daemons
- ``kea-dhcp-v4-daemon`` - run for Kea DHCPv4 daemons
- ``kea-dhcp-v6-daemon`` - run for Kea DHCPv6 daemons
- ``kea-d2-daemon`` - run for Kea D2 daemons
- ``bind9-daemon`` - run for BIND 9 daemons

The ``Triggers`` indicate the conditions under which the checkers are executed. Currently,
there are three types of triggers:

- ``manual`` - run on user's request
- ``config change`` - run when daemon configuration change has been detected
- ``host reservations change`` - run when a change in the Kea host reservations database has been detected

The selectors and triggers are not configurable by users.

Synchronizing Kea Configurations
================================

Stork pullers periodically check Kea configurations against the local copies
stored in the Stork database. These local copies are only updated when Stork
detects any mismatch. This approach works fine in most cases and eliminates
the overhead of unnecessarily updating the local database. However, there are
possible scenarios when a mismatch between the configurations is not detected,
but it is still desirable to fetch and repopulate the configurations from the Kea
servers to Stork.

There are many internal operations in Stork that may be occurring when a configuration change
is detected (e.g., populating host reservations, log viewer initialization,
configuration reviews, and many others). Resynchronizing the configurations from Kea
triggers all these tasks. The resynchronization may correct some data integrity issues that
sometimes occur due to software bugs, network errors, or any other reason.

To schedule a configuration synchronization from the Kea servers, navigate to
``Services`` and then ``Kea Daemons``, and click on the ``Resynchronize Kea Configs`` button.
The pullers fetch and populate the updated configuration data, but this operation
takes time, depending on the configured puller intervals. Ensure the pullers
are not disabled on the ``Settings`` page; otherwise, the configurations will
never re-synchronize.
