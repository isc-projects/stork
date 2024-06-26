import { FormGroup } from '@angular/forms'
import { CreateSubnetBeginResponse, SharedNetwork, UpdateSubnetBeginResponse } from '../backend'
import { SelectableClientClass } from './selectable-client-class'
import { SelectableDaemon } from './selectable-daemon'
import { SubnetForm } from './subnet-set-form.service'
import { SelectableSharedNetwork } from './selectable-shared-network'

/**
 * Holds the state of the form created by the SubnetFormComponent.
 *
 * The state is shared with the parent component via the event emitter
 * and can be used to re-create the SubnetFormComponent with the already
 * edited form data. It is particularly useful when the component is
 * destroyed as a result of switching between different tabs.
 */
export class SubnetFormState {
    /**
     * A boolean value indicating if the updated form was passed to
     * the parent component when the SubnetFormComponent was destroyed.
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
     * A subnet id of the modified or created subnet.
     */
    subnetId: number = 0

    /**
     * An error to begin the transaction returned by the server.
     */
    initError: string

    /**
     * A form group comprising all form controls, arrays and other form
     * groups (a parent group for the SubnetFormComponent form).
     */
    group: FormGroup<SubnetForm>

    /**
     * A list of all daemons that can be selected from the drop down list.
     */
    allDaemons: SelectableDaemon[]

    /**
     * A filtered list of daemons comprising only those that match the
     * type of the first selected daemon.
     *
     * Maintaining a filtered list prevents the user from selecting the
     * servers of different kinds, e.g. one DHCPv4 and one DHCPv6 server.
     */
    filteredDaemons: SelectableDaemon[]

    /**
     * An array of client classes.
     */
    clientClasses: SelectableClientClass[]

    /**
     * All IPv4 shared networks received from the server.
     */
    allSharedNetworks4: SharedNetwork[]

    /**
     * All IPv6 shared networks received from the server.
     */
    allSharedNetworks6: SharedNetwork[]

    /**
     * An array of selectable shared networks.
     */
    selectableSharedNetworks: SelectableSharedNetwork[]

    /**
     * A flag set to true when DHCPv4 servers have been selected.
     */
    dhcpv4: boolean = false

    /**
     * A flag set to true when DHCPv6 servers have been selected.
     */
    dhcpv6: boolean = false

    /**
     * Indicates if the form is at the wizard stage.
     *
     * When the form is used to create a new subnet, the form initially displays
     * only the input box for the subnet prefix. This is because the form validation
     * highly depends on the subnet prefix. This boolean flag indicates if the form
     * is at this stage of subnet specification.
     */
    wizard: boolean = false

    /**
     * Names of the servers currently associated with the subnet.
     *
     * The names are displayed as tags next to the configuration parameters
     * and DHCP options.
     */
    servers: string[] = []

    /**
     * Holds the received server's response to the createSubnetBegin or updateSubnetBegin
     * call.
     *
     * It is required to revert the subnet edits.
     */
    savedSubnetBeginData: CreateSubnetBeginResponse | UpdateSubnetBeginResponse

    /**
     * Indicates if the form has been loaded.
     *
     * The component shows a progress spinner when this value is false.
     */
    loaded: boolean = false

    /**
     * Returns a daemon having the specified ID.
     *
     * @param id daemon ID.
     * @returns specified daemon or null if it doesn't exist.
     */
    getDaemonById(id: number): SelectableDaemon | null {
        return this.allDaemons.find((d) => d.id === id)
    }

    /**
     * Updates the form state according to the daemons selection.
     *
     * Depending on whether the user selected DHCPv4 or DHCPv6 servers, the
     * list of filtered daemons must be updated to prevent selecting different
     * daemon types. The boolean dhcpv4 and dhcpv6 flags also have to be tuned.
     * Based on the impact on these flags, the function checks whether or not
     * the new selection is a breaking change. Such a change requires that
     * some states of the form must be reset. In particular, if a user already
     * specified some options, the breaking change indicates that these options
     * must be removed from the form because they may be in conflict with the
     * new daemons selection.
     *
     * @param selectedDaemons new set of selected daemons' ids.
     * @param subnet subnet prefix.
     * @returns true if the update results in a breaking change, false otherwise.
     */
    updateFormForSelectedDaemons(selectedDaemons: number[], subnet?: string): boolean {
        let dhcpv6 = false
        let dhcpv4 = selectedDaemons.some((ss) => {
            return this.allDaemons.find((d) => d.id === ss && d.name === 'dhcp4')
        })
        if (!dhcpv4) {
            // If user selected no DHCPv4 server, perhaps selected a DHCPv6 server?
            dhcpv6 = selectedDaemons.some((ss) => {
                return this.allDaemons.find((d) => d.id === ss && d.name === 'dhcp6')
            })
        }
        // If user unselected DHCPv4 servers, unselected DHCPv6 servers or selected
        // DHCPv6 servers, it is a breaking change.
        let breakingChange = (this.dhcpv4 && !dhcpv4) || this.dhcpv6 !== dhcpv6

        // Remember new states.
        this.dhcpv4 = dhcpv4
        this.dhcpv6 = dhcpv6

        // Filter selectable other selectable servers based on the current selection.
        if (dhcpv4 || subnet?.includes('.')) {
            this.filteredDaemons = this.allDaemons.filter((d) => d.name === 'dhcp4')
            this.selectableSharedNetworks =
                this.allSharedNetworks4
                    ?.filter(
                        (sn) =>
                            sn.localSharedNetworks.every((lsn) => selectedDaemons?.some((id) => id === lsn.daemonId)) &&
                            sn.localSharedNetworks.length === selectedDaemons?.length
                    )
                    .map((sn) => {
                        return { name: sn.name, id: sn.id }
                    }) || []
        } else if (this.dhcpv6 || subnet?.includes(':')) {
            this.filteredDaemons = this.allDaemons.filter((d) => d.name === 'dhcp6')
            this.selectableSharedNetworks =
                this.allSharedNetworks6
                    ?.filter(
                        (sn) =>
                            sn.localSharedNetworks.every((lsn) => selectedDaemons?.some((id) => id === lsn.daemonId)) &&
                            sn.localSharedNetworks.length === selectedDaemons?.length
                    )
                    .map((sn) => {
                        return { name: sn.name, id: sn.id }
                    }) || []
        } else {
            this.filteredDaemons = this.allDaemons
            this.selectableSharedNetworks = []
        }
        return breakingChange
    }
}
