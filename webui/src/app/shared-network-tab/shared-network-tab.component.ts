import { Component, EventEmitter, Input, OnInit, Output } from '@angular/core'
import { SharedNetworkWithUniquePools, hasDifferentLocalSharedNetworkOptions } from '../subnets'
import { NamedCascadedParameters } from '../cascaded-parameters-board/cascaded-parameters-board.component'
import { DHCPOption, KeaConfigSubnetDerivedParameters } from '../backend'

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
     * A component lifecycle hook invoked upon the component initialization.
     *
     * It initializes the @link dhcpParameters arrray by combining the shared
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
}
