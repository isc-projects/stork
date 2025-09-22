import { Component, viewChild } from '@angular/core'

import { MessageService } from 'primeng/api'

import { DHCPService } from '../backend'
import { getErrorMessage } from '../utils'
import { lastValueFrom } from 'rxjs'
import { HostForm } from '../forms/host-form'
import { Host } from '../backend'
import { HostsTableComponent } from '../hosts-table/hosts-table.component'

/**
 * This component implements a page which displays hosts along with
 * their DHCP identifiers and IP reservations. The list of hosts is
 * paged and can be filtered by provided URL queryParams or by
 * using form inputs responsible for filtering. The list
 * contains hosts reservations for all subnets and also contain global
 * reservations, i.e. those that are not associated with any particular
 * subnet.
 *
 * This component is also responsible for viewing given host reservation
 * details in tab view, switching between tabs, closing them etc.
 */
@Component({
    selector: 'app-hosts-page',
    standalone: false,
    templateUrl: './hosts-page.component.html',
    styleUrls: ['./hosts-page.component.sass'],
})
export class HostsPageComponent {
    /**
     * Table with hosts component.
     */
    hostsTable = viewChild<HostsTableComponent>('hostsTableComponent')

    breadcrumbs = [{ label: 'DHCP' }, { label: 'Host Reservations' }]

    /**
     * Function used to asynchronously provide the host reservation based on given host ID.
     */
    hostProvider: (id: number) => Promise<Host> = (id) => lastValueFrom(this.dhcpApi.getHost(id))

    /**
     * Function used to provide new HostForm instance.
     */
    hostFormProvider: () => HostForm = () => new HostForm()

    /**
     * Constructor.
     *
     * @param dhcpApi server API used to gather hosts information.
     * @param messageService message service used to display error messages to a user.
     */
    constructor(
        private dhcpApi: DHCPService,
        private messageService: MessageService
    ) {}

    /**
     * Generates a host tab label.
     *
     * Different host reservation properties may be used to generate the label,
     * depending on their availability:
     * - first reserved IP address,
     * - first reserved delegated prefix,
     * - hostname,
     * - first DHCP identifier,
     * - host reservation ID.
     *
     * @param host host information from which the label should be generated.
     * @returns generated host label.
     */
    hostLabelProvider = (host: Host) => {
        if (host.addressReservations && host.addressReservations.length > 0) {
            return host.addressReservations[0].address
        }

        if (host.prefixReservations && host.prefixReservations.length > 0) {
            return host.prefixReservations[0].address
        }

        if (host.hostname && host.hostname.length > 0) {
            return host.hostname
        }

        if (host.hostIdentifiers && host.hostIdentifiers.length > 0) {
            return host.hostIdentifiers[0].idType + '=' + host.hostIdentifiers[0].idHexValue
        }

        return '[' + host.id + ']'
    }

    /**
     * Function used to call REST API endpoint responsible for deleting the transaction of the 'create new host reservation' form.
     */
    callCreateHostDeleteTransaction = (transactionID: number) => {
        lastValueFrom(this.dhcpApi.createHostDelete(transactionID)).catch((err) => {
            const msg = getErrorMessage(err)
            this.messageService.add({
                severity: 'error',
                summary: 'Failed to delete configuration transaction',
                detail: 'Failed to delete configuration transaction: ' + msg,
                life: 10000,
            })
        })
    }

    /**
     * Function used to call REST API endpoint responsible for deleting the transaction of the 'update existing host reservation' form.
     */
    callUpdateHostDeleteTransaction = (hostID: number, transactionID) => {
        lastValueFrom(this.dhcpApi.updateHostDelete(hostID, transactionID)).catch((err) => {
            const msg = getErrorMessage(err)
            this.messageService.add({
                severity: 'error',
                summary: 'Failed to delete configuration transaction',
                detail: 'Failed to delete configuration transaction: ' + msg,
                life: 10000,
            })
        })
    }
}
