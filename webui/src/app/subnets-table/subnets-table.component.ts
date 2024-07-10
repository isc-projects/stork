import { Component, OnDestroy, OnInit, ViewChild } from '@angular/core'
import { PrefilteredTable } from '../table'
import { DHCPService, Subnet } from '../backend'
import { Table, TableLazyLoadEvent } from 'primeng/table'
import { ActivatedRoute } from '@angular/router'
import { MessageService } from 'primeng/api'
import { Location } from '@angular/common'
import { lastValueFrom } from 'rxjs'
import { getErrorMessage } from '../utils'
import { extractUniqueSubnetPools, parseSubnetsStatisticValues, SubnetWithUniquePools } from '../subnets'
import { map } from 'rxjs/operators'

/**
 * Specifies the filter parameters for fetching subnets that may be specified
 * either in the URL query parameters or programmatically.
 */
export interface SubnetsFilter {
    text?: string
    appId?: number
    subnetId?: number
    dhcpVersion?: number
}

@Component({
    selector: 'app-subnets-table',
    templateUrl: './subnets-table.component.html',
    styleUrl: './subnets-table.component.sass',
})
export class SubnetsTableComponent extends PrefilteredTable<SubnetsFilter, Subnet> implements OnInit, OnDestroy {
    /**
     * Array of all numeric keys that are supported when filtering subnets via URL queryParams.
     * Note that it doesn't have to contain subnets prefilterKey, which is 'appId'.
     * prefilterKey by default is considered as a primary queryParam filter key.
     */
    queryParamNumericKeys: (keyof SubnetsFilter)[] = []

    /**
     * Array of all boolean keys that are supported when filtering subnets via URL queryParams.
     * Currently, no boolean key is supported in queryParams filtering.
     */
    queryParamBooleanKeys: (keyof SubnetsFilter)[] = []

    /**
     * Array of all numeric keys that can be used to filter subnets.
     */
    filterNumericKeys: (keyof SubnetsFilter)[] = ['appId', 'subnetId', 'dhcpVersion']

    /**
     * Array of all boolean keys that can be used to filter subnets.
     */
    filterBooleanKeys: (keyof SubnetsFilter)[] = []

    /**
     * Prefix of the stateKey. Will be used to evaluate stateKey.
     */
    stateKeyPrefix: string = 'subnets-table-session'

    /**
     * queryParam keyword of the filter by appId.
     */
    prefilterKey: keyof SubnetsFilter = 'appId'

    /**
     * PrimeNG table instance.
     */
    @ViewChild('subnetsTable') table: Table

    subnets: SubnetWithUniquePools[] = []

    constructor(
        private route: ActivatedRoute,
        private dhcpApi: DHCPService,
        private messageService: MessageService,
        private location: Location
    ) {
        super(route, location)
    }

    /**
     * Loads subnets from the database into the component.
     *
     * @param event Event object containing an index if the first row, maximum
     * number of rows to be returned and a text for subnets filtering. If it is
     * not specified, the current values are used when available.
     */
    loadData(event: TableLazyLoadEvent) {
        // Indicate that subnets refresh is in progress.
        this.dataLoading = true
        // The goal is to send to backend something as simple as:
        // this.someApi.getSubnets(JSON.stringify(event))

        lastValueFrom(
            this.dhcpApi
                .getSubnets(
                    event.first,
                    event.rows,
                    this.prefilterValue ?? this.getTableFilterValue('appId', event.filters),
                    this.getTableFilterValue('subnetId', event.filters),
                    this.getTableFilterValue('dhcpVersion', event.filters),
                    this.getTableFilterValue('text', event.filters)
                )
                // Custom parsing for statistics
                .pipe(
                    map((subnets) => {
                        if (subnets.items) {
                            parseSubnetsStatisticValues(subnets.items)
                        }
                        return subnets
                    })
                )
        )
            .then((data) => {
                this.subnets = data.items ? extractUniqueSubnetPools(data.items) : []
                this.totalRecords = data.total ?? 0
            })
            .catch((error) => {
                this.messageService.add({
                    severity: 'error',
                    summary: 'Cannot load subnets',
                    detail: getErrorMessage(error),
                })
            })
            .finally(() => {
                this.dataLoading = false
            })
    }

    /**
     * Component lifecycle hook called to perform clean-up when destroying the component.
     */
    ngOnDestroy(): void {
        super.onDestroy()
    }

    /**
     * Component lifecycle hook called upon initialization.
     */
    ngOnInit(): void {
        super.onInit()
    }
}
