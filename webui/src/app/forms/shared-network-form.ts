import { FormGroup } from '@angular/forms'
import { CreateSharedNetworkBeginResponse, UpdateSharedNetworkBeginResponse } from '../backend'
import { SelectableClientClass } from './selectable-client-class'
import { SelectableDaemon } from './selectable-daemon'
import { SharedNetworkForm } from './subnet-set-form.service'
import { IPType } from '../iptype'

/**
 * Holds the state of the form created by the SharedNetworkFormComponent.
 *
 * The state is shared with the parent component via the event emitter
 * and can be used to re-create the SharedNetworkFormComponent with the
 * already edited form data. It is particularly useful when the component
 * is destroyed as a result of switching between different tabs.
 */
export class SharedNetworkFormState {
    /**
     * A transaction id returned by the server after sending the
     * request to begin one.
     */
    private _transactionId: number = 0

    /**
     * An id of the modified or created shared network.
     */
    public sharedNetworkId: number = 0

    /**
     * An error to begin the transaction returned by the server.
     */
    private _initError: string

    /**
     * A list of all daemons that can be selected from the drop down list.
     */
    private _allDaemons: SelectableDaemon[] = []

    /**
     * A filtered list of daemons comprising only those that match the
     * type of the first selected daemon.
     *
     * Maintaining a filtered list prevents the user from selecting the
     * servers of different kinds, e.g. one DHCPv4 and one DHCPv6 server.
     */
    private _filteredDaemons: SelectableDaemon[] = []

    /**
     * An array of client classes.
     */
    private _clientClasses: SelectableClientClass[] = []

    /**
     * All IPv4 shared networks names received from the server.
     */
    private _allSharedNetworks4: string[] = []

    /**
     * All IPv6 shared networks names received from the server.
     */
    private _allSharedNetworks6: string[] = []

    /**
     * A flag set to true when DHCPv4 servers have been selected.
     */
    private _dhcpv4: boolean = false

    /**
     * A flag set to true when DHCPv6 servers have been selected.
     */
    private _dhcpv6: boolean = false

    /**
     * Names of the servers currently associated with the shared networks.
     *
     * The names are displayed as tags next to the configuration parameters
     * and DHCP options.
     */
    private _servers: string[] = []

    /**
     * Indicates if the form has been loaded.
     *
     * The component shows a progress spinner when this value is false.
     */
    private _loaded: boolean = false

    /**
     * Holds the received server's response to the updateSharedNetworkBegin
     * call.
     *
     * It is required to revert the subnet edits.
     */
    public savedSharedNetworkBeginData: CreateSharedNetworkBeginResponse | UpdateSharedNetworkBeginResponse

    /**
     * A form group comprising all form controls, arrays and other form
     * groups (a parent group for the SharedNetworkFormComponent form).
     */
    public group: FormGroup<SharedNetworkForm>

    /**
     * Returns transaction id returned by the server after sending the
     * request to begin one.
     */
    get transactionId(): number {
        return this._transactionId
    }

    /**
     * Returns a filtered list of daemons comprising only those that match the
     * type of the first selected daemon.
     *
     * Maintaining a filtered list prevents the user from selecting the
     * servers of different kinds, e.g. one DHCPv4 and one DHCPv6 server.
     */
    get filteredDaemons(): SelectableDaemon[] {
        return this._filteredDaemons
    }

    /**
     * Returns an index of the selected daemon in the filtered daemons list.
     *
     * @param daemonId daemon ID.
     * @returns A daemon index of a negative value if the daemon is not found.
     */
    getFilteredDaemonIndex(daemonId: number): number {
        return daemonId ? this.filteredDaemons.findIndex((fd) => fd.id === daemonId) : -1
    }

    /**
     * Returns a daemon having the specified ID.
     *
     * @param id daemon ID.
     * @returns specified daemon or null if it doesn't exist.
     */
    private getDaemonById(id: number): SelectableDaemon | null {
        return this._allDaemons.find((d) => d.id === id)
    }

    /**
     * Returns an array of available client classes.
     *
     * The returned classes can be used in the shared network configuration.
     */
    get clientClasses(): SelectableClientClass[] {
        return this._clientClasses
    }

    /**
     * Returns current shared network IP type.
     *
     * @returns IPv6 type when DHCPv6 servers have been selected for a
     *          shared network, IPv4 otherwise.
     */
    get ipType(): IPType {
        return this._dhcpv6 ? IPType.IPv6 : IPType.IPv4
    }

    /**
     * Returns a flag indicating if the shared network has IPv6 type.
     *
     * @returns true if the shared network has IPv6 type.
     */
    get dhcpv6(): boolean {
        return this._dhcpv6
    }

    /**
     * Returns the indication if the form has been loaded.
     *
     * The component shows a progress spinner when this value is false.
     */
    get loaded(): boolean {
        return this._loaded
    }

    /**
     * Marks the form loaded.
     */
    markLoaded(): void {
        this._loaded = true
    }

    /**
     * Returns names of the servers currently associated with the shared networks.
     *
     * The names are displayed as tags next to the configuration parameters
     * and DHCP options.
     */
    get servers(): string[] {
        return this._servers
    }

    /**
     * Assigns the server labels based on the daemon IDs.
     *
     * The servers are assigned in the same order in which the IDs have
     * been specified.
     *
     * @param daemonIds an array of daemon IDs.
     */
    updateServers(daemonIds: number[]): void {
        this._servers = daemonIds.map((id) => this.getDaemonById(id)?.label)
    }

    /**
     * Returns form initialization error.
     */
    get initError(): string {
        return this._initError
    }

    /**
     * Sets form initialization error.
     *
     * @param err an initialization error.
     */
    setInitError(err: string): void {
        this._initError = err
        this.markLoaded()
    }

    /**
     * Returns a list of shared network name for the current shared network
     * IP type.
     */
    get existingSharedNetworkNames(): string[] {
        return this._dhcpv6 ? this._allSharedNetworks6 : this._allSharedNetworks4
    }

    /**
     * Given the server response it sets the state values.
     *
     * It doesn't initialize the form group. Setting it is a component's
     * responsibility.
     *
     * @param response A received server response.
     */
    initStateFromServerResponse(response: CreateSharedNetworkBeginResponse | UpdateSharedNetworkBeginResponse): void {
        // Success. Clear any existing errors.
        this._initError = null

        // The server should return new transaction id and a current list of
        // daemons to select.
        this._transactionId = response.id
        this._allDaemons = []
        this._allDaemons = response.daemons.map((d) => {
            return {
                id: d.id,
                appId: d.app?.id,
                appType: d.app?.type,
                name: d.name,
                version: d.version,
                label: `${d.app?.name}/${d.name}`,
            }
        })
        // Initially, list all daemons.
        this._filteredDaemons = this._allDaemons
        this._allSharedNetworks4 =
            response.sharedNetworks4?.filter(
                (name) => name !== (response as UpdateSharedNetworkBeginResponse)?.sharedNetwork?.name
            ) || []
        this._allSharedNetworks6 =
            response.sharedNetworks6?.filter(
                (name) => name !== (response as UpdateSharedNetworkBeginResponse)?.sharedNetwork?.name
            ) || []
        this._clientClasses =
            response.clientClasses?.map((c) => {
                return { name: c }
            }) || []

        // If we update an existing subnet the subnet information should be in the response.
        const updateResponse = response as UpdateSharedNetworkBeginResponse
        if (this.sharedNetworkId && updateResponse?.sharedNetwork) {
            // Get the server names to be displayed next to the configuration parameters.
            this.updateServers(updateResponse.sharedNetwork.localSharedNetworks.map((lsn) => lsn.daemonId))
            // Save the shared network information in case we need to revert the form changes.
            // Determine whether it is an IPv6 or IPv4 shared network.
            this._dhcpv4 = updateResponse.sharedNetwork.universe === IPType.IPv4
            this._dhcpv6 = updateResponse.sharedNetwork.universe === IPType.IPv6
        }
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
     * @returns true if the update results in a breaking change, false otherwise.
     */
    updateFormForSelectedDaemons(selectedDaemons: number[]): boolean {
        let universe = (this.savedSharedNetworkBeginData as UpdateSharedNetworkBeginResponse)?.sharedNetwork?.universe
        let dhcpv6 = false
        let dhcpv4 = selectedDaemons.some((ss) => {
            return this._allDaemons.find((d) => d.id === ss && d.name === 'dhcp4')
        })
        if (!dhcpv4) {
            // If user selected no DHCPv4 server, perhaps selected a DHCPv6 server?
            dhcpv6 = selectedDaemons.some((ss) => {
                return this._allDaemons.find((d) => d.id === ss && d.name === 'dhcp6')
            })
        }
        // If user selected or unselected DHCP servers of a certain kind it is a
        // breaking change. In the DHCPv4 case we don't treat the transition from
        // non-DHCPv4 to DHCPv4 case as a breaking change because, by default, we
        // assume DHCPv4 case when no servers are initially selected. In this case
        // the form remains unchanged after selecting the first server when new
        // shared network is defined.
        const breakingChange = (this._dhcpv4 && !dhcpv4) || this._dhcpv6 !== dhcpv6

        // Remember new states.
        this._dhcpv4 = dhcpv4
        this._dhcpv6 = dhcpv6

        // Filter selectable other selectable servers based on the current selection.
        if (dhcpv4 || universe === IPType.IPv4) {
            this._filteredDaemons = this._allDaemons.filter((d) => d.name === 'dhcp4')
        } else if (this._dhcpv6 || universe === IPType.IPv6) {
            this._filteredDaemons = this._allDaemons.filter((d) => d.name === 'dhcp6')
        } else {
            this._filteredDaemons = this._allDaemons
        }
        return breakingChange
    }
}
