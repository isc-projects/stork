**************************
Managing Kea Configuration
**************************

Host Reservations
=================

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
`Host Reservations in DHCPv4 <https://kea.readthedocs.io/en/latest/arm/dhcp4-srv.html?#host-reservations-in-dhcpv4>`_
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

.. note::

   The ``Delete`` button is unavailable for host reservations configured in the
   Kea configuration files, or when the reservations are configured in the host
   database but the ``host_cmds`` hook library is not loaded.

Subnets
=======

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

Shared Networks
===============

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
