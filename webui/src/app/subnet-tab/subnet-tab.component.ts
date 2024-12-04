import { Component, EventEmitter, Input, OnInit, Output } from '@angular/core'
import { DHCPOption, DHCPService, KeaConfigSubnetDerivedParameters, Subnet } from '../backend'
import { hasAddressPools, hasDifferentLocalSubnetOptions, hasPrefixPools } from '../subnets'
import { hasDifferentLocalSubnetPools } from '../subnets'
import { NamedCascadedParameters } from '../cascaded-parameters-board/cascaded-parameters-board.component'
import { deepEqual, getErrorMessage } from '../utils'
import { ConfirmationService, MessageService } from 'primeng/api'
import { lastValueFrom } from 'rxjs'

/**
 * A component displaying a tab for a selected subnet.
 */
@Component({
    selector: 'app-subnet-tab',
    templateUrl: './subnet-tab.component.html',
    styleUrls: ['./subnet-tab.component.sass'],
})
export class SubnetTabComponent implements OnInit {
    /**
     * Subnet data.
     */
    @Input() subnet: Subnet

    /**
     * An event emitter notifying a parent that user has clicked the
     * Edit button to modify the subnet.
     */
    @Output() subnetEditBegin = new EventEmitter<any>()

    /**
     * An event emitter notifying a parent that user has clicked the
     * Delete button to delete the subnet.
     */
    @Output() subnetDelete = new EventEmitter<any>()

    /**
     * DHCP parameters structured for display by the @link CascadedParametersBoard.
     *
     * The parameters are structured as an array of subnet-level, shared network-level
     * and global parameters.
     */
    dhcpParameters: Array<NamedCascadedParameters<KeaConfigSubnetDerivedParameters>> = []

    /**
     * DHCP options structured for display by the @link DhcpOptionSetView.
     *
     * The options are structured as an array of subnet-level, shared network-level
     * and global options.
     */
    dhcpOptions: DHCPOption[][][] = []

    /**
     * Disables the button deleting a subnet after clicking this button.
     */
    subnetDeleting = false

    /**
     * Component constructor.
     *
     * @param dhcpApi service used to communicate with the server over REST API.
     * @param confirmService confirmation service displaying the confirm dialog when
     * attempting to delete the subnet.
     * @param msgService service displaying error messages upon a communication
     *                   error with the server.
     */
    constructor(
        private dhcpApi: DHCPService,
        private confirmService: ConfirmationService,
        private msgService: MessageService
    ) {}

    /**
     * A component lifecycle hook invoked upon the component initialization.
     *
     * It initializes the @link dhcpParameters array by combining the subnet-level,
     * shared network-level and global parameters into an array. If the subnet does
     * not belong to a shared network, the array only contains subnet-level and
     * global parameters.
     */
    ngOnInit(): void {
        if (this.subnet?.localSubnets) {
            for (let ls of this.subnet.localSubnets) {
                this.dhcpParameters.push({
                    name: ls.appName,
                    parameters:
                        this.subnet.sharedNetwork?.length > 0
                            ? [
                                  ls.keaConfigSubnetParameters?.subnetLevelParameters || {},
                                  ls.keaConfigSubnetParameters?.sharedNetworkLevelParameters || {},
                                  ls.keaConfigSubnetParameters?.globalParameters || {},
                              ]
                            : [
                                  ls.keaConfigSubnetParameters?.subnetLevelParameters || {},
                                  ls.keaConfigSubnetParameters?.globalParameters || {},
                              ],
                })

                if (this.subnet.sharedNetwork?.length > 0) {
                    this.dhcpOptions.push([
                        ls.keaConfigSubnetParameters?.subnetLevelParameters?.options || [],
                        ls.keaConfigSubnetParameters?.sharedNetworkLevelParameters?.options || [],
                        ls.keaConfigSubnetParameters?.globalParameters?.options || [],
                    ])
                } else {
                    this.dhcpOptions.push([
                        ls.keaConfigSubnetParameters?.subnetLevelParameters?.options || [],
                        ls.keaConfigSubnetParameters?.globalParameters?.options || [],
                    ])
                }
            }
        }
    }

    /**
     * Checks if the subnet has IPv6 type.
     *
     * @return true if the subnet has IPv6 type, false otherwise.
     */
    get isIPv6(): boolean {
        return this.subnet.subnet.includes(':')
    }

    /**
     * Returns attributes used in constructing a link to a shared network.
     *
     * @returns a map of attributes including shared network name and a universe.
     */
    getSharedNetworkAttrs() {
        return {
            id: this.subnet.sharedNetworkId,
            name: this.subnet.sharedNetwork,
        }
    }

    /**
     * Checks if the subnet has any address pools.
     *
     * @returns true if the subnet has any address pools, false otherwise.
     */
    subnetHasAddressPools(): boolean {
        return hasAddressPools(this.subnet)
    }

    /**
     * Checks if the subnet has any prefix pools.
     *
     * @returns true if the subnet has any prefix pools, false otherwise.
     */
    subnetHasPrefixPools(): boolean {
        return hasPrefixPools(this.subnet)
    }

    /**
     * Check if all daemons using the subnet have the same configured pools.
     *
     * Usually all servers have the same set of pools configured for a subnet.
     * However, it is also a valid use case for the servers to have different
     * pools. In that case, the component must display the pools for individual
     * servers separately. This function checks if this is the case.
     *
     * @returns true if all servers have the same set of pools for a subnet,
     * false otherwise.
     */
    allDaemonsHaveEqualPools(): boolean {
        return !hasDifferentLocalSubnetPools(this.subnet)
    }

    /**
     * Checks if all DHCP servers owning the subnet have an equal set of
     * DHCP options.
     *
     * @returns true, if all DHCP servers have equal option set hashes, false
     *          otherwise.
     */
    allDaemonsHaveEqualDhcpOptions(): boolean {
        return !hasDifferentLocalSubnetOptions(this.subnet)
    }

    /**
     * Event handler called when user begins subnet editing.
     *
     * It emits an event to the parent component to notify that subnet is
     * is now edited.
     */
    onSubnetEditBegin(): void {
        this.subnetEditBegin.emit(this.subnet)
    }

    /**
     * Displays a dialog to confirm subnet deletion.
     */
    confirmDeleteSubnet() {
        this.confirmService.confirm({
            message: 'Are you sure that you want to permanently delete this subnet?',
            header: 'Delete Subnet',
            icon: 'pi pi-exclamation-triangle',
            accept: () => {
                this.deleteSubnet()
            },
        })
    }

    /**
     * Sends a request to the server to delete the subnet.
     */
    deleteSubnet() {
        // Disable the button for deleting the subnet to prevent pressing the
        // button multiple times and sending multiple requests.
        this.subnetDeleting = true
        lastValueFrom(this.dhcpApi.deleteSubnet(this.subnet.id))
            .then((/* data */) => {
                // Re-enable the delete button.
                this.subnetDeleting = false
                this.msgService.add({
                    severity: 'success',
                    summary: `Subnet ${this.subnet.subnet} successfully deleted`,
                })
                // Notify the parent that the subnet was deleted and the tab can be closed.
                this.subnetDelete.emit(this.subnet)
            })
            .catch((err) => {
                // Re-enable the delete button.
                this.subnetDeleting = false
                // Issues with deleting the host.
                const msg = getErrorMessage(err)
                this.msgService.add({
                    severity: 'error',
                    summary: 'Cannot delete the subnet',
                    detail: `Failed to delete the subnet ${this.subnet.subnet} : ${msg}`,
                    life: 10000,
                })
            })
    }

    /**
     * Indicates if the subnet has any user-context.
     */
    get hasUserContext(): boolean {
        return !!this.subnet.localSubnets?.some((ls) => ls.userContext)
    }

    /**
     * Indicates if all local subnets have the same user-context.
     */
    allDaemonsHaveEqualUserContext(): boolean {
        if (!(this.subnet.localSubnets?.length > 0)) {
            // If there are no local subnets, the user context is not relevant
            // and we can assume that the user context is the same.
            return true
        }

        const firstUserContext = this.subnet.localSubnets[0].userContext
        for (let i = 1; i < this.subnet.localSubnets.length; i++) {
            if (!deepEqual(firstUserContext, this.subnet.localSubnets[i].userContext)) {
                return false
            }
        }
        return true
    }
}
