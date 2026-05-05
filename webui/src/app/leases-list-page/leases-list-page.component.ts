import { Component, viewChild } from '@angular/core'
import { BreadcrumbsComponent } from '../breadcrumbs/breadcrumbs.component'
import { TabViewComponent } from '../tab-view/tab-view.component'
import { LeasesListTableComponent } from '../leases-list-table/leases-list-table.component'

/**
 * This component implements a page which displays leases gathered by
 * the Lease Tracking feature. The list of leases is
 * paged and can be filtered by provided URL queryParams or by
 * using form inputs responsible for filtering. The list
 * contains leases for all subnets.
 */
@Component({
    selector: 'app-leases-list-page',
    templateUrl: './leases-list-page.component.html',
    styleUrls: ['./leases-list-page.component.sass'],
    imports: [BreadcrumbsComponent, TabViewComponent, LeasesListTableComponent],
})
export class LeasesListPageComponent {
    /**
     * Table with leases component.
     */
    leasesListTable = viewChild<LeasesListTableComponent>('leasesListTableComponent')

    breadcrumbs = [{ label: 'DHCP' }, { label: 'Leases List' }]
}
