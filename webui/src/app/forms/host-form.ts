import { FormGroup } from '@angular/forms'
import { IPv4CidrRange, IPv6CidrRange, Validator } from 'ip-num'
import { KeaDaemon } from '../backend/model/keaDaemon'
import { Subnet } from '../backend/model/subnet'
import { IPType } from '../iptype'

/**
 * Holds the state of the form created by the HostFormComponent.
 *
 * The state is shared with the parent component via the event emitter
 * and can be used to re-create the HostFormComponent with the already
 * edited form data. It is particularly useful when the component is
 * destroyed as a result of switching between different tabs.
 */
export class HostForm {
    /**
     * A boolean value indicating if the updated form was passed to
     * the parent component when the HostFormComponent was destroyed.
     *
     * When the component is re-created it checks this value to decide
     * whether or not the form should be initialized with default values.
     */
    preserved: boolean = false

    /**
     * A transaction id returned by the server after sending the
     * request to begin one.
     */
    transactionId: number = 0

    /**
     * An error to begin the transaction returned by the server.
     */
    initError: string

    /**
     * A form group comprising all form controls, arrays and other form
     * groups (a parent group for the HostFormComponent form).
     */
    group: FormGroup

    /**
     * A list of all daemons that can be selected from the drop down list.
     */
    allDaemons: KeaDaemon[]

    /**
     * A filtered list of daemons comprising only those that match the
     * type of the first selected daemon.
     *
     * Maintaining a filtered list prevents the user from selecting the
     * servers of different kinds, e.g. one DHCPv4 and one DHCPv6 server.
     */
    filteredDaemons: KeaDaemon[]

    /**
     * A list of subnets that can be selected from the drop down list.
     *
     * An actual drop down list can be shorter depending on the list of
     * selected servers. It displays only the subnets that the selected
     * servers serve.
     */
    allSubnets: Subnet[]

    /**
     * An array of selectable subnets according to the current form data.
     *
     * Suppose a user selected a server in the form. In this case, this
     * array comprises only the subnets served by this server.
     */
    filteredSubnets: Subnet[]

    /**
     * A flag set to true when DHCPv4 servers have been selected.
     */
    dhcpv4: boolean = false

    /**
     * A flag set to true when DHCPv6 servers have been selected.
     */
    dhcpv6: boolean = false

    /**
     * Returns an address range of a selected subnet.
     *
     * If no subnet is selected, a null value is returned.
     *
     * @returns an array of a subnet prefix and the corresponding IP range.
     */
    getSelectedSubnetRange(): [string, IPv4CidrRange | IPv6CidrRange] | null {
        const selected = this.filteredSubnets.find((fs) => fs.id === this.group.get('selectedSubnet').value)
        if (!selected) {
            return null
        }
        // Use the subnet to find the address range.
        let subnetRange: IPv4CidrRange | IPv6CidrRange
        if (Validator.isValidIPv4CidrRange(selected.subnet)[0]) {
            return [selected.subnet, IPv4CidrRange.fromCidr(selected.subnet)]
        }
        if (Validator.isValidIPv6CidrRange(selected.subnet)[0]) {
            return [selected.subnet, IPv6CidrRange.fromCidr(selected.subnet)]
        }
        return null
    }
}
