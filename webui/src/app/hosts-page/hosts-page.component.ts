import { Component, viewChild } from '@angular/core'

import { MessageService } from 'primeng/api'

import { DHCPService } from '../backend'
import { getErrorMessage } from '../utils'
import { lastValueFrom } from 'rxjs'
import { HostForm } from '../forms/host-form'
import { Host } from '../backend'
import { HostsTableComponent } from '../hosts-table/hosts-table.component'
import { TabType } from '../tab'
import { ComponentTab, TabViewComponent } from '../tab-view/tab-view.component'

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
    templateUrl: './hosts-page.component.html',
    styleUrls: ['./hosts-page.component.sass'],
})
export class HostsPageComponent {
    /**
     * Table with hosts component.
     */
    hostsTable = viewChild<HostsTableComponent>('hostsTableComponent')

    tabView = viewChild(TabViewComponent)

    breadcrumbs = [{ label: 'DHCP' }, { label: 'Host Reservations' }]

    hostProvider: (id: number) => Promise<Host> = (id) => lastValueFrom(this.dhcpApi.getHost(id))
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
     *
     * @param hostID
     * @param transactionID
     */
    cancelHostUpdateTransaction(hostID: number, transactionID: number) {
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

    /**
     *
     * @param tab
     */
    onTabClosed(tab: ComponentTab) {
        console.log('onHostTabClosed', tab)
        if (!tab.form) {
            return
        }

        const transactionID = (tab.form.formState as HostForm).transactionID
        if (tab.tabType === TabType.New && transactionID > 0 && !tab.form.submitted) {
            lastValueFrom(this.dhcpApi.createHostDelete(transactionID)).catch((err) => {
                let msg = err.statusText
                if (err.error && err.error.message) {
                    msg = err.error.message
                }

                this.messageService.add({
                    severity: 'error',
                    summary: 'Failed to delete configuration transaction',
                    detail: 'Failed to delete configuration transaction: ' + msg,
                    life: 10000,
                })
            })
        } else if (tab.tabType === TabType.Edit && tab.value > 0 && transactionID > 0 && !tab.form.submitted) {
            this.cancelHostUpdateTransaction(tab.value, transactionID)
        }
    }

    protected readonly TabType = TabType

    /**
     *
     * @param hostID
     * @param formState
     * @param tabType
     */
    onFormCancel(formState: HostForm, tabType: TabType, hostID?: number) {
        console.log('onHostFormCancel', hostID, formState.transactionID, tabType)
        if (hostID && formState.transactionID && tabType === TabType.Edit) {
            this.cancelHostUpdateTransaction(hostID, formState.transactionID)
            // formState.transactionID = 0
        }

        this.tabView()?.onCancelForm(tabType, hostID || undefined)
    }
}
