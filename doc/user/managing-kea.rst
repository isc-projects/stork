**************************
Managing Kea Configuration
**************************

Host Reservations
=================

Creating Host Reservations
~~~~~~~~~~~~~~~~~~~~~~~~~~

Navigate to ``DHCP -> Host Reservations`` to list the host reservations.
Click the ``New Host`` button. It opens a tab where you can specify a new
host reservation in one or more Kea servers. These Kea servers must be
configured to use the ``host_cmds`` hooks library, and only these servers
are available for selection in the ``DHCP Servers`` dropdown.

You have a choice between a subnet-level or global host reservation.
Selecting a subnet using the ``Subnet`` dropdown is required for a
subnet-level reservation. If the desired subnet is not displayed in the
dropdown, it is possible that the selected DHCP servers do not include this
subnet in their configuration. Setting the ``Global reservation`` option
disables subnet selection.

To associate the new host reservation with a DHCP client, you can select
one of the identifier types supported by Kea. Available identifiers differ
depending on whether the user selected DHCPv4 or DHCPv6 servers. The identifier
can be specified using ``hex`` or ``text`` format. For example, the ``hw-address``
is typically specified as a string of hexadecimal digits: ``ab:76:54:c6:45:31``.
In that case, select ``hex`` option. Some identifiers, e.g. ``circuit-id``, are
often specified using "printable characters", e.g. ``circuit-no-1``. In that case,
select ``text`` option. Please refer to
`Host Reservations in DHCPv4 <https://kea.readthedocs.io/en/latest/arm/dhcp4-srv.html?#host-reservations-in-dhcpv4>`_
and `Host Reservations in DHCPv6 <https://kea.readthedocs.io/en/latest/arm/dhcp6-srv.html#host-reservations-in-dhcpv6>`_
for more details regarding allowed DHCP identifiers and their formats.

Further in the form, you can specify the actual reservations. It is possible
to specify at most one IPv4 address. In the case of the DHCPv6 servers, it is
possible to specify multiple IPv6 addresses and delegated prefixes.

The DHCPv4 ``siaddr``, ``sname`` and ``file`` fields can be statically assigned to
the clients using host reservations. The relevant values in Kea and Stork are:
``Next Server``, ``Server Hostname``, and ``Boot File Name``. You can only set these
values for the DHCPv4 servers. The form lacks controls for setting them when
editing a DHCPv6 host.

It is possible to associate one or more client classes with a host. Kea servers
assign these classes to the DHCP packets received from the client having
the host reservation. Client classes are typically defined in the Kea
configurations but not necessarily. For example, built-in classes like
``DROP`` have no explicit definitions in the configuration files.
You can click the ``List`` button to select client classes from the list of
classes explicitly defined in the configurations of the monitored Kea servers.
Select the desired class names and click ``Insert``. If the desired class
name is not on the list, you can type the class name directly in the
input box and press enter. Click on the cross icon next to the class name
to delete it from the host reservation.

DHCP options can be added to the host reservation by clicking the ``Add Option``
button. The list of the standard DHCP options is available via the dropdown.
However, if the list is missing a desired option, you can simply
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

   Currently, Stork does not verify whether or not the specified options comply
   with the formats specified in the RFCs, nor does it check them against the
   runtime option definitions configured in Kea. If you specify wrong option
   format, Stork will try to send the option to Kea for verification,
   and Kea will reject the new reservation. The reservation can be submitted
   again after correcting the option payload.

Please use the ``Add <field-type>`` button to add suboptions to a DHCP option.
Stork supports top-level options with maximum two levels of suboptions.

If a host reservation is configured in several DHCP servers, typically, all
servers comprise the same set of parameters (i.e., IP addresses, hostname,
boot fields, client classes and DHCP options). By default, creating a new
host reservation for several servers sends an identical copy of the host
reservation to each. You may choose to specify a different set of boot fields,
client classes or options for different servers by selecting
``Configure individual server vaues`` at the top of the form. In this case,
you must specify the complete sets of boot fields, client classes and options
for each DHCP server. Leaving them blank for some servers means that these
servers receive no boot fields, classes or DHCP options with the reservation.

Updating Host Reservations
~~~~~~~~~~~~~~~~~~~~~~~~~~

In a selected host reservation's view, click ``Edit`` button to open a form for
editing host reservation information. The form automatically toggles editing
DHCP options individually for each server (see above) when it detects different
option sets on different servers using the reservation. Besides editing the
host reservation information, it is also possible to deselect some of the
servers (using the DHCP Servers dropdown), which will delete the reservation
from these servers.

Use the ``Revert Changes`` button to remove all applied changes and restore
the original host reservation information. Use ``Cancel`` to close the form
without applying the changes.

Deleting Host Reservations
~~~~~~~~~~~~~~~~~~~~~~~~~~

To delete a host reservation from all DHCP servers for which it is configured,
click on the reservation in the host reservations list. Find the ``Delete``
button and confirm the reservation deletion. Use it with caution because this
operation cannot be undone. The reservation is removed from the DHCP servers'
databases. It must be re-created to be restored.

.. note::

   The ``Delete`` button is unavailable for host reservations configured in the
   Kea configuration files or when the reservations are configured in the host
   database, but the ``host_cmds`` hook library is not loaded.

Subnets
=======

.. _creating-subnets:

Creating Subnets
~~~~~~~~~~~~~~~~

Stork can configure new subnets in the Kea instances with the ``subnet_cmds``
hook library loaded. Navigate to ``DHCP -> Subnets`` to display the subnets list. Click
the ``New Subnet`` button. The opened form initially contains only an input box where
subnet prefix must be specified. It can be an IPv4 (e.g., ``192.0.2.0/24``) or
IPv6 prefix (e.g., ``2001:db8:1::/64``). Click the ``Proceed`` button to expand the
form where the remaining subnet configuration can be entered.

The Stork subnet form is designed to specify the common subnet configuration that
can be instantly populated to multiple DHCP servers. Configuring the same subnet in
multiple Kea instances is specific to the deployments where service redundancy is
required (e.g., deployments using high availability or with a shared lease database).
Hence, when configuring a new subnet it is possible to select multiple DHCP servers
in the ``Assignments`` panel. The subnet will be populated to these servers. Please
note that the list of servers only contains those matching the subnet prefix
(IPv4 or IPv6). Additionally, only the servers with the ``subnet_cmds`` hook library
are listed.

The new subnet may be assigned to a shared network in the ``Subnet`` panel. The shared
networks list can be empty for two reasons:

- There are no shared networks in the selected Kea instances.
- Some Kea instances selected for the subnet lack shared networks specification.

In the first case, a desired shared network should be created before creating the subnet.
In the latter case, the shared network should be updated, and assigned to all servers
to which the subnet will be assigned. Suppose you want to add a new subnet and assign
it to the ``server 1`` and ``server 2``. If this subnet must be also added to the shared
network that is only assigned to the ``server 1``, first edit the shared network, assign
it to the ``server 2`` besides the ``server 1``. Then, create new subnet, assign it to the
``server 1`` and ``server 2``. The shared networks list should now contain our shared network.
Select this shared network from the list in the subnet form. Once the shared network is
selected it is not possible to change the assignments of the subnet to the servers. To
change these assignments, first unassign the subnet from the shared network. Click the
``cross`` button located to the right of the selected shared network name. The subnet
assignments can now be changed.

The subnet usually comes with one or more address pools (both IPv4 and IPv6). It may
also contain the delegated prefix pools (IPv6 only). The DHCP servers assign leases
to the clients from the resources available in these pools. The address pool boundaries
are specified as a pair of addresses (i.e., first and last address). Both addresses
must match the subnet prefix (must be within this subnet). The first address must be
lower or equal the last address. If they are equal, the pool contains exactly one
address. Empty pools are not allowed.

In some deployments multiple DHCP servers can share the same subnets but they may
include different pools. In this scenario, the administrators avoid the conflicts
whereby two servers offer the same address (from overlapping pools) to different
clients. Stork facilicates this scenario by allowing assigning a pool to a subset
of the DHCP servers assigned to the subnet. If the pool should be included in
all servers, pick all servers in the pool's ``Assignments`` panel. Note that, besides
specifying the pool boundaries and assigments, each expandable pool panel also
contains the form controls to specify some pool-level configuration parameters,
such as: ``Client Class``, ``Pool ID`` etc. It is also possible to specify pool-level
DHCP options.

Create more pools as needed using the ``Add Pool`` button. Click ``Delete Pool``
to remove selected pool from the subnet.

Delegated prefix pools can be added for IPv6 subnets. The delegated prefix pool
boundaries are specified differently than the address pool boundaries. Also, the
delegated prefix pool prefix does not have to match (belong to) the subnet prefix.
The delegated prefix pool comprises an actual prefix (e.g., ``3000::/64``) and
a delegated prefix length (e.g, ``96``). The delegated prefix length must be
greater than or equal prefix length. In the examples above ``96 > 64``. If they are
equal, the delegated prefix pool contains exactly one prefix.

The `RFC 6603 <https://www.rfc-editor.org/rfc/rfc6603.html>`_ describes the mechanism
to allow exclusion of one specific prefix from a delegated prefix set in DHCPv6.
This prefix can be optionally specified as ``Excluded Prefix`` for a delegated
prefix pool. This prefix must belong to the delegated prefix and its length must be
greater than the delegated prefix length.

The Kea subnet configuration contains ``DHCP Parameters`` which contain different
aspects of lease assignment in that subnet. By default, each DHCP server in the
subnet gets the same values of the DHCP parameters. In some cases, however, an
administrator can choose to specify different values for the same parameter for
different servers. Check ``Unlock`` box for the selected parameters. It splits
the form for these parameters, so you can specify different values for different
servers in the input boxes marked with the colored server names.

The ``DHCP Options`` panel allows for specifying DHCP options to be returned to
the clients connected to the subnet. In most cases, these options are common
for different servers assigned to the subnet. However, it is possible to differentiate
some options using similar mechanism to the one described above for the ``DHCP Parameters``.
Click ``Unlock setting DHCP options for individual servers`` and set respective option
sets for different servers.

Each DHCP option specification begins with the selection of the option code from the dropdown
list. The input boxes displayed below the option code represent the option fields carried
by the option. Fill these fields with the values appropriate for the option.

If a DHCP option carries an array of fields, only the input box for the first field
is initially displayed. To add more fields to the array, expand the dropdown list
right below the option code selector, and select correct option field type to
be added to the array. The option fields and the options can also be removed from
the form.

When the subnet form holds the necessary data, click the ``Submit`` button to save
the subnet and send it to the Kea servers. The ``Submit`` button is disabled as
long as the form has some invalid entries.

Updating Subnets
~~~~~~~~~~~~~~~~

To update an existing subnet configuration click on the subnet in the dashboard
or in the subnets list. The detailed information about the subnet is displayed.
Click the ``Edit`` button to open the subnet update form. Note that only a subnet
associated with the servers configured to use ``subnet_cmds`` hook library can
be updated.

Subnet configuration is described in detail in the :ref:`creating-subnets` section.
Here, we are going to describe some specific behavior pertaining to updating
a subnet.

A subnet prefix cannot be modified for an updated subnet. If you need to increase
or decrease a subnet prefix length simply create new subnet and delete the
existing one.

If a shared network field is cleared for the updated subnet, this subnet will be
removed from the shared network in the Kea servers. If another shared network
is selected instead, the subnet will be first removed from the existing shared
network and then added to the newly selected shared network.

You can delete a pool from a subnet. However, it is important to understand the
implications. While the pool itself is removed from the configuration instantly,
the leases allocated in this pool are not. Kea maintains these leases in the lease
database and the clients continue using the leases until the leases expire or
until the clients attempt to renew them. The renewing clients will be refused to
extend the leases belonging to the deleted pools and allocated new leases from
the existing pools.

Finally, the form for updating a subnet contains the ``Revert Changes`` button that
allows for dropping all changes to the subnet configuration since the form was
opened.

Deleting Subnets
~~~~~~~~~~~~~~~~

To delete a subnet from Stork and the Kea instances navigate to the subnet view
from the dashboard or the subnets list. Click the ``Delete`` button and confirm
the deletion. It will remove the subnet from all Kea instances holding this
subnet. Deleting a subnet requires that the Kea servers holding the subnet run
the ``subnet_cmds`` hook library.

Shared Networks
===============

Creating Shared Networks
~~~~~~~~~~~~~~~~~~~~~~~~

Stork can configure new shared networks in the Kea instances with the ``subnet_cmds``
hook libraries. The shared networks group subnets with common configuration parameters,
and to provide a common address space for the DHCP clients connected to different
subnets. Navigate to the shared networks list (``DHCP -> Shared Networks``). Click
the ``New Shared Network`` button.

Shared network must be assigned to one or more DHCP servers selected in the ``Assignments``
panel. All servers must be of the same kind (DHCPv4 or DHCPv6). Therefore, after selecting
the first server the list is reduced to the servers of the same kind. The shared network
will be created in all of the selected Kea servers.

A shared network name is mandatory. It is an arbitrary value that must be unique among
the servers connected to Stork.

The ``DHCP Parameters`` and ``DHCP Options`` specified for the shared network are common
for all subnets later added to this shared network. However, the same parameters and options
specified at the subnet level override these common shared network-level values.

Similarly to :ref:`creating-subnets`, it is possible to unlock selected parameters and
options, and specify different values for different servers holding the shared network
configuration.

When the form is ready, the shared network can be created in Stork and
the Kea instances by clicking the ``Submit`` button. This button is disabled as long as
the form has some invalid entries.

Updating Shared Networks
~~~~~~~~~~~~~~~~~~~~~~~~

To update an existing shared network configuration click on the shared network
in the dashboard or in the shared networks list. The detailed information about
the shared network is displayed. Click the ``Edit`` button to open a shared network
update form. Note that only a shared network associated with the servers configured
to use ``subnet_cmds`` hook library can be updated.

Removing the shared network from a server (in the ``Assignments`` panel) also removes
the subnets belonging to this shared network from the server. They are added back
when the server is added to the shared network.

Update the shared network as needed and click ``Submit`` to save the changes in
Stork and in the Kea instances.

Deleting Shared Networks
~~~~~~~~~~~~~~~~~~~~~~~~

To delete a shared network from Stork and the Kea instances navigate to the subnet
view from the dashboard or the subnets list. Click the ``Delete`` button and confirm
the deletion. It will remove the shared network from all Kea instances holding this
shared network. Deleting a shared network requires that the Kea servers holding the
shared network run the ``subnet_cmds`` hook library.

Deleting a shared network also deletes all subnets it includes. If you intend to
preserve the subnets from the deleted shared network, click on each subnet
belonging to it, edit the subnet, clear the shared network selection in the
``Subnet`` panel, and save the subnet changes. Next, delete the empty shared network.
