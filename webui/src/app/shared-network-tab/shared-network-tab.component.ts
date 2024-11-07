import { Component, EventEmitter, Input, OnInit, Output } from '@angular/core'
import { SharedNetworkWithUniquePools, hasDifferentLocalSharedNetworkOptions } from '../subnets'
import { NamedCascadedParameters } from '../cascaded-parameters-board/cascaded-parameters-board.component'
import { DHCPOption, DHCPService, KeaConfigSubnetDerivedParameters, SharedNetwork } from '../backend'
import { ConfirmationService, MessageService } from 'primeng/api'
import { lastValueFrom } from 'rxjs'
import { getErrorMessage } from '../utils'

@Component({
    selector: 'app-shared-network-tab',
    templateUrl: './shared-network-tab.component.html',
    styleUrls: ['./shared-network-tab.component.sass'],
})
export class SharedNetworkTabComponent implements OnInit {
    /**
     * Shared network data.
     */
    @Input() sharedNetwork: SharedNetworkWithUniquePools

    /**
     * An event emitter notifying a parent that user has clicked the
     * Edit button to modify the shared network.
     */
    @Output() sharedNetworkEditBegin = new EventEmitter<any>()

    /**
     * An event emitter notifying a parent that user has clicked the
     * Delete button to delete the shared network.
     */
    @Output() sharedNetworkDelete = new EventEmitter<SharedNetwork>()

    /**
     * DHCP parameters structured for display by the @link CascadedParametersBoard.
     *
     * The parameters are structured as an array of shared network-level and global parameters.
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
     * Disables the button deleting a shared network after clicking this button.
     */
    sharedNetworkDeleting = false

    /**
     * Component constructor.
     *
     * @param dhcpApi service used to communicate with the server over REST API.
     * @param confirmService confirmation service displaying the confirm dialog when
     *        attempting to delete the shared network.
     * @param msgService service displaying error messages upon a communication
     *        error with the server.
     */
    constructor(
        private dhcpApi: DHCPService,
        private confirmService: ConfirmationService,
        private msgService: MessageService
    ) {}

    /**
     * A component lifecycle hook invoked upon the component initialization.
     *
     * It initializes the @link dhcpParameters array by combining the shared
     * network-level and global parameters into an array.
     */
    ngOnInit(): void {
        if (this.sharedNetwork?.localSharedNetworks) {
            for (let ls of this.sharedNetwork.localSharedNetworks) {
                this.dhcpParameters.push({
                    name: ls.appName,
                    parameters: [
                        ls.keaConfigSharedNetworkParameters?.sharedNetworkLevelParameters,
                        ls.keaConfigSharedNetworkParameters?.globalParameters,
                    ],
                })

                this.dhcpOptions.push([
                    ls.keaConfigSharedNetworkParameters?.sharedNetworkLevelParameters?.options,
                    ls.keaConfigSharedNetworkParameters?.globalParameters?.options,
                ])
            }
        }
    }

    /**
     * Checks if all DHCP servers owning the shared network have an equal set of
     * DHCP options.
     *
     * @returns true, if all DHCP servers have equal option set hashes, false
     *          otherwise.
     */
    allDaemonsHaveEqualDhcpOptions(): boolean {
        return !hasDifferentLocalSharedNetworkOptions(this.sharedNetwork)
    }

    /**
     * Event handler called when user begins shared network editing.
     *
     * It emits an event to the parent component to notify that shared network
     * is now edited.
     */
    onSharedNetworkEditBegin(): void {
        this.sharedNetworkEditBegin.emit(this.sharedNetwork)
    }

    /**
     * Displays a dialog to confirm shared network deletion.
     */
    confirmDeleteSharedNetwork() {
        this.confirmService.confirm({
            message: 'Are you sure that you want to permanently delete this shared network and its subnets?',
            header: 'Delete Shared Network',
            icon: 'pi pi-exclamation-triangle',
            accept: () => {
                this.deleteSharedNetwork()
            },
        })
    }

    /**
     * Sends a request to the server to delete the shared network.
     */
    deleteSharedNetwork() {
        // Disable the button for deleting the shared network to prevent pressing the
        // button multiple times and sending multiple requests.
        this.sharedNetworkDeleting = true
        lastValueFrom(this.dhcpApi.deleteSharedNetwork(this.sharedNetwork.id))
            .then((/* data */) => {
                this.msgService.add({
                    severity: 'success',
                    summary: `Shared network ${this.sharedNetwork.name} successfully deleted`,
                })
                // Notify the parent that the shared network was deleted and the tab can be closed.
                this.sharedNetworkDelete.emit(this.sharedNetwork)
            })
            .catch((err) => {
                // Re-enable the delete button.
                // Issues with deleting the host.
                const msg = getErrorMessage(err)
                this.msgService.add({
                    severity: 'error',
                    summary: 'Cannot delete the shared network',
                    detail: `Failed to delete the shared network ${this.sharedNetwork.name} : ${msg}`,
                    life: 10000,
                })
            })
            .finally(() => {
                // Re-enable the delete button.
                this.sharedNetworkDeleting = false
            })
    }
}
