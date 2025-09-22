import { Component, viewChild } from '@angular/core'

import { DHCPService } from '../backend'
import { getErrorMessage } from '../utils'
import { parseSubnetStatisticValues, extractUniqueSharedNetworkPools, SharedNetworkWithUniquePools } from '../subnets'
import { lastValueFrom } from 'rxjs'
import { finalize, map } from 'rxjs/operators'
import { MessageService } from 'primeng/api'
import { SharedNetworkFormState } from '../forms/shared-network-form'
import { SharedNetworksTableComponent } from '../shared-networks-table/shared-networks-table.component'

/**
 * Component for presenting shared networks in a table.
 */
@Component({
    selector: 'app-shared-networks-page',
    standalone: false,
    templateUrl: './shared-networks-page.component.html',
    styleUrls: ['./shared-networks-page.component.sass'],
})
export class SharedNetworksPageComponent {
    breadcrumbs = [{ label: 'DHCP' }, { label: 'Shared Networks' }]

    /**
     * Table with shared networks component.
     */
    networksTableComponent = viewChild<SharedNetworksTableComponent>('networksTableComponent')

    /**
     * Indicates if the component is loading data from the server.
     */
    loading: boolean = false

    /**
     * Function used to asynchronously provide the shared network based on given ID.
     * @param sharedNetworkID ID of the shared network
     */
    sharedNetworkProvider: (sharedNetworkID: number) => Promise<SharedNetworkWithUniquePools> = (
        sharedNetworkID: number
    ) => {
        this.loading = true
        return lastValueFrom(
            this.dhcpApi.getSharedNetwork(sharedNetworkID).pipe(
                map((sharedNetwork) => {
                    if (sharedNetwork) {
                        parseSubnetStatisticValues(sharedNetwork)
                        sharedNetwork = extractUniqueSharedNetworkPools([sharedNetwork])[0]
                    }
                    return sharedNetwork as SharedNetworkWithUniquePools
                }),
                finalize(() => (this.loading = false))
            )
        )
    }

    /**
     * Function used to provide new SharedNetworkFormState instance.
     */
    sharedNetworkFormProvider: () => SharedNetworkFormState = () => new SharedNetworkFormState()

    /**
     * Function used to call REST API endpoint responsible for deleting the transaction of the 'create new shared network' form.
     * @param transactionID ID of the transaction to be deleted
     */
    callCreateNetworkDeleteTransaction = (transactionID: number) =>
        lastValueFrom(this.dhcpApi.createSharedNetworkDelete(transactionID)).catch((err) => {
            let msg = getErrorMessage(err)
            this.messageService.add({
                severity: 'error',
                summary: 'Failed to delete configuration transaction',
                detail: 'Failed to delete configuration transaction: ' + msg,
                life: 10000,
            })
        })

    /**
     * Function used to call REST API endpoint responsible for deleting the transaction of the 'update existing shared network' form.
     * @param entityID shared network ID for which the transaction is to be deleted
     * @param transactionID ID of the transaction to be deleted
     */
    callUpdateNetworkDeleteTransaction = (entityID: number, transactionID: number) =>
        lastValueFrom(this.dhcpApi.updateSharedNetworkDelete(entityID, transactionID)).catch((err) => {
            let msg = getErrorMessage(err)
            this.messageService.add({
                severity: 'error',
                summary: 'Failed to delete configuration transaction',
                detail: 'Failed to delete configuration transaction: ' + msg,
                life: 10000,
            })
        })

    /**
     * Constructor.
     *
     * @param messageService message service.
     * @param dhcpApi a service for communication with the server.
     */
    constructor(
        private messageService: MessageService,
        private dhcpApi: DHCPService
    ) {}
}
