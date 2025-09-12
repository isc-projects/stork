import { Component, OnDestroy, OnInit, viewChild } from '@angular/core'

import { DHCPService } from '../backend'
import { getErrorMessage } from '../utils'
import { extractUniqueSubnetPools, parseSubnetStatisticValues, SubnetWithUniquePools } from '../subnets'
import { SettingService } from '../setting.service'
import { Subscription, lastValueFrom } from 'rxjs'
import { map } from 'rxjs/operators'
import { Settings } from '../backend'
import { MessageService } from 'primeng/api'
import { SubnetFormState } from '../forms/subnet-form'
import { SubnetsTableComponent } from '../subnets-table/subnets-table.component'

/**
 * Component for presenting DHCP subnets.
 */
@Component({
    selector: 'app-subnets-page',
    templateUrl: './subnets-page.component.html',
    styleUrls: ['./subnets-page.component.sass'],
})
export class SubnetsPageComponent implements OnInit, OnDestroy {
    private subscriptions = new Subscription()
    breadcrumbs = [{ label: 'DHCP' }, { label: 'Subnets' }]

    /**
     * Table with subnets component.
     */
    table = viewChild<SubnetsTableComponent>('subnetsTableComponent')

    /**
     * Indicates if the component is loading data from the server.
     */
    loading: boolean = false

    /**
     * Base URL to Grafana.
     */
    grafanaUrl: string

    /**
     * ID of the DHCPv4 dashboard in Grafana.
     */
    grafanaDhcp4DashboardId: string

    /**
     * ID of the DHCPv6 dashboard in Grafana.
     */
    grafanaDhcp6DashboardId: string

    /**
     * Function used to asynchronously provide the subnet based on given subnet ID.
     */
    subnetProvider: (id: number) => Promise<SubnetWithUniquePools> = (id) =>
        lastValueFrom(
            // Fetch data from API.
            this.dhcpApi.getSubnet(id).pipe(
                map((subnet) => {
                    parseSubnetStatisticValues(subnet)
                    subnet = extractUniqueSubnetPools(subnet)[0]
                    return subnet as SubnetWithUniquePools
                })
            )
        )

    /**
     * Function used to provide new SubnetFormState instance.
     */
    subnetFormProvider: () => SubnetFormState = () => new SubnetFormState()

    constructor(
        private dhcpApi: DHCPService,
        private settingSvc: SettingService,
        private messageService: MessageService
    ) {}

    ngOnDestroy(): void {
        this.subscriptions.unsubscribe()
    }

    ngOnInit() {
        // ToDo: Silent error catching
        this.subscriptions.add(
            this.settingSvc.getSettings().subscribe(
                (data: Settings) => {
                    if (!data) {
                        return
                    }

                    this.grafanaUrl = data.grafanaUrl
                    this.grafanaDhcp4DashboardId = data.grafanaDhcp4DashboardId
                    this.grafanaDhcp6DashboardId = data.grafanaDhcp6DashboardId
                },
                (error) => {
                    this.messageService.add({
                        severity: 'error',
                        summary: 'Cannot fetch server settings',
                        detail: getErrorMessage(error),
                    })
                }
            )
        )
    }

    /**
     * Function used to call REST API endpoint responsible for deleting the transaction of the 'create new subnet' form.
     */
    callCreateSubnetDeleteTransaction = (transactionID: number) => {
        lastValueFrom(this.dhcpApi.createSubnetDelete(transactionID)).catch((err) => {
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
     * Function used to call REST API endpoint responsible for deleting the transaction of the 'update existing subnet' form.
     */
    callUpdateSubnetDeleteTransaction = (subnetID: number, transactionID: number) => {
        lastValueFrom(this.dhcpApi.updateSubnetDelete(subnetID, transactionID)).catch((err) => {
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
