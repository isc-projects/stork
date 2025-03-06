import { fakeAsync, tick, ComponentFixture, TestBed, waitForAsync } from '@angular/core/testing'
import { FormsModule } from '@angular/forms'
import { HttpClientTestingModule } from '@angular/common/http/testing'
import { ActivatedRoute, Router } from '@angular/router'
import { RouterTestingModule } from '@angular/router/testing'
import { By } from '@angular/platform-browser'
import { of, throwError } from 'rxjs'

import { MessageService } from 'primeng/api'
import { TableModule } from 'primeng/table'

import { LeaseSearchPageComponent } from './lease-search-page.component'
import { DHCPService } from '../backend'
import { LocaltimePipe } from '../pipes/localtime.pipe'
import { datetimeToLocal } from '../utils'
import { BreadcrumbsComponent } from '../breadcrumbs/breadcrumbs.component'
import { HelpTipComponent } from '../help-tip/help-tip.component'
import { MessageModule } from 'primeng/message'
import { ProgressSpinnerModule } from 'primeng/progressspinner'
import { FieldsetModule } from 'primeng/fieldset'
import { OverlayPanelModule } from 'primeng/overlaypanel'
import { NoopAnimationsModule } from '@angular/platform-browser/animations'
import { BreadcrumbModule } from 'primeng/breadcrumb'
import { MessagesModule } from 'primeng/messages'
import { ToggleButtonModule } from 'primeng/togglebutton'
import { JsonTreeRootComponent } from '../json-tree-root/json-tree-root.component'
import { JsonTreeComponent } from '../json-tree/json-tree.component'
import { IdentifierComponent } from '../identifier/identifier.component'

describe('LeaseSearchPageComponent', () => {
    let component: LeaseSearchPageComponent
    let fixture: ComponentFixture<LeaseSearchPageComponent>
    let dhcpApi: DHCPService
    let msgService: MessageService
    let router: Router
    let route: ActivatedRoute

    beforeEach(waitForAsync(() => {
        TestBed.configureTestingModule({
            providers: [DHCPService, MessageService],
            imports: [
                FormsModule,
                HttpClientTestingModule,
                RouterTestingModule.withRoutes([
                    {
                        path: 'dhcp/leases',
                        component: LeaseSearchPageComponent,
                    },
                ]),
                TableModule,
                MessageModule,
                ProgressSpinnerModule,
                FieldsetModule,
                OverlayPanelModule,
                NoopAnimationsModule,
                BreadcrumbModule,
                MessagesModule,
                ToggleButtonModule,
            ],
            declarations: [
                LeaseSearchPageComponent,
                LocaltimePipe,
                BreadcrumbsComponent,
                HelpTipComponent,
                JsonTreeComponent,
                JsonTreeRootComponent,
                IdentifierComponent,
            ],
        }).compileComponents()
    }))

    beforeEach(() => {
        fixture = TestBed.createComponent(LeaseSearchPageComponent)
        component = fixture.componentInstance
        dhcpApi = fixture.debugElement.injector.get(DHCPService)
        msgService = fixture.debugElement.injector.get(MessageService)
        router = fixture.debugElement.injector.get(Router)
        route = fixture.debugElement.injector.get(ActivatedRoute)
        router.navigate(['dhcp/leases'])
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should ignore empty search text', () => {
        const searchInput = fixture.debugElement.query(By.css('#leases-search-input'))
        const searchInputElement = searchInput.nativeElement

        spyOn(dhcpApi, 'getLeases')

        // Simulate typing only spaces in the search box.
        searchInputElement.value = '    '
        searchInputElement.dispatchEvent(new Event('input'))
        searchInputElement.dispatchEvent(new KeyboardEvent('keyup', { key: 'Enter' }))
        fixture.detectChanges()

        // Make sure that search is not triggered.
        expect(dhcpApi.getLeases).not.toHaveBeenCalled()
    })

    it('should trigger leases search', fakeAsync(() => {
        const searchInput = fixture.debugElement.query(By.css('#leases-search-input'))
        const searchInputElement = searchInput.nativeElement

        const fakeLeases: any = {
            items: [
                {
                    id: 0,
                    ipAddress: '192.0.2.3',
                    state: 0,
                    appId: 1,
                    appName: 'kea@frog',
                    hwAddress: '01:02:03:04:05:06',
                    subnetId: 123,
                    cltt: 1616149050,
                    validLifetime: 3600,
                },
            ],
            total: 0,
            erredApps: [],
        }
        spyOn(dhcpApi, 'getLeases').and.returnValue(of(fakeLeases))

        // Simulate typing only spaces in the search box.
        searchInputElement.value = '192.1.0.1'
        searchInputElement.dispatchEvent(new Event('input'))
        searchInputElement.dispatchEvent(new KeyboardEvent('keyup', { key: 'Enter' }))
        tick()
        fixture.detectChanges()

        // Make sure that search is triggered.
        expect(dhcpApi.getLeases).toHaveBeenCalled()

        // Make sure that the information was correctly populated.
        expect(component.searchText).toBe('192.1.0.1')
        expect(component.lastSearchText).toBe('192.1.0.1')
        expect(component.searchStatus).toBe(component.Status.Searched)
        expect(component.leases.length).toBe(1)
        expect(component.leases[0].hasOwnProperty('id')).toBeTrue()
        expect(component.leases[0].id).toBe(1)

        // A warning message informing about erred apps should not be displayed.
        const erredAppsMessage = fixture.debugElement.query(By.css('#erred-apps-message'))
        expect(erredAppsMessage).toBeNull()
    }))

    it('should return correct lease state name', () => {
        expect(component.leaseStateAsText(null)).toBe('Valid')
        expect(component.leaseStateAsText(0)).toBe('Valid')
        expect(component.leaseStateAsText(1)).toBe('Declined')
        expect(component.leaseStateAsText(2)).toBe('Expired/Reclaimed')
        expect(component.leaseStateAsText(3)).toBe('Unknown')
    })

    it('should return correct lease type name', () => {
        expect(component.leaseTypeAsText(null)).toBe('IPv4 address')
        expect(component.leaseTypeAsText('IA_NA')).toBe('IPv6 address (IA_NA)')
        expect(component.leaseTypeAsText('IA_PD')).toBe('IPv6 prefix (IA_PD)')
        expect(component.leaseTypeAsText('XYZ')).toBe('Unknown')
    })

    it('should display DHCPv4 lease', () => {
        component.leases = [
            {
                id: 0,
                ipAddress: '192.0.2.3',
                state: 0,
                appId: 1,
                appName: 'kea@frog',
                hwAddress: '01:02:03:04:05:06',
                clientId: '51:52:53:54',
                hostname: 'faq.example.org',
                fqdnFwd: false,
                fqdnRev: true,
                subnetId: 123,
                cltt: 1616149050,
                validLifetime: 3600,
                userContext: { ISC: { 'client-classes': ['ALL', 'HA_primary', 'UNKNOWN'] } },
            },
        ]
        component.lastSearchText = '192.0.2.3'
        fixture.detectChanges()

        const leasesTable = fixture.debugElement.query(By.css('#leases-table'))
        const cols = leasesTable.queryAll(By.css('td'))
        expect(cols.length).toBe(5)

        // Expand button existence.
        expect(cols[0].children.length).toBe(1)
        const expandButton = cols[0].children[0].nativeElement
        expect(expandButton.nodeName).toBe('A')

        // Lease properties.
        expect(cols[1].nativeElement.innerText).toBe('192.0.2.3')
        expect(cols[2].nativeElement.innerText).toBe('IPv4 address')
        expect(cols[3].nativeElement.innerText).toBe('Valid')
        expect(cols[4].nativeElement.innerText).toBe('kea@frog')

        // Validate app link.
        expect(cols[4].children.length).toBe(1)
        expect(cols[4].children[0].attributes.href).toBe('/apps/kea/1')

        // Simulate expanding the lease information.
        expandButton.click()
        fixture.detectChanges()

        // Find the tables holding expanded information.
        const tables = fixture.debugElement.queryAll(By.css('table'))
        expect(tables.length).toBe(4)

        // Find allocation and expiration time.
        const startDate = new Date(1616149050000)
        const endDate = new Date(1616149050000 + 3600000)

        // Expected data in various tables within the expanded row.
        const expectedLeaseData: any = [
            [
                ['MAC address', '01:02:03:04:05:06'],
                ['Client Identifier', 'QRST'],
            ],
            [
                ['Subnet Identifier', '123'],
                ['Valid Lifetime', '3600 seconds'],
                ['Allocated at', datetimeToLocal(startDate)],
                ['Expires at', datetimeToLocal(endDate)],
            ],
            [
                ['Hostname', 'faq.example.org'],
                ['Forward DDNS', 'no'],
                ['Reverse DDNS', 'yes'],
            ],
        ]

        // Second, third and forth tables should contain expanded lease information.
        // For each table check if the data is correct.
        let tableIndex = 1
        for (const expectedDataGroup of expectedLeaseData) {
            const rows = tables[tableIndex].queryAll(By.css('tr'))
            expect(rows.length).toBe(expectedDataGroup.length)

            // For each table row, compare its contents with the expected data.
            let i = 0
            for (const row of rows) {
                expect(row.children.length).toBe(2)
                expect(row.children[0].nativeElement.innerText).toBe(expectedDataGroup[i][0] + ':')
                expect(row.children[1].nativeElement.innerText).toContain(expectedDataGroup[i][1])
                i++
            }
            tableIndex++
        }

        // Test User Context JSON tree content.
        const tree = fixture.debugElement.queryAll(By.directive(JsonTreeRootComponent))
        expect(tree).not.toBeNull()
        expect(Object.keys(tree).length).toBe(1)
        const treeComponent = tree[0].componentInstance as JsonTreeComponent
        expect(treeComponent).not.toBeNull()
        expect(treeComponent.value).not.toBeNull()
        expect(Object.keys(treeComponent.value).length).toBe(1)
        expect(treeComponent.value['ISC']).not.toBeNull()
        expect(Object.keys(treeComponent.value['ISC']).length).toBe(1)
        expect(treeComponent.value['ISC']['client-classes']).not.toBeNull()
        expect(treeComponent.value['ISC']['client-classes'].length).toBe(3)
        expect(treeComponent.value['ISC']['client-classes'][0]).toBe('ALL')
        expect(treeComponent.value['ISC']['client-classes'][1]).toBe('HA_primary')
        expect(treeComponent.value['ISC']['client-classes'][2]).toBe('UNKNOWN')

        // Test summary.
        const leasesSearchSummary = fixture.debugElement.query(By.css('#leases-search-summary-span'))
        expect(leasesSearchSummary.properties.innerText).toBe('Found 1 lease matching 192.0.2.3.')
    })

    it('should display declined DHCPv4 lease', () => {
        // Declined lease lacks MAC address and client identifier.
        component.leases = [
            {
                id: 0,
                ipAddress: '192.0.2.3',
                state: 1,
                appId: 1,
                appName: 'kea@frog',
                subnetId: 123,
                cltt: 1616149050,
                validLifetime: 3600,
            },
        ]
        component.lastSearchText = '192.0.2.3'
        fixture.detectChanges()

        const leasesTable = fixture.debugElement.query(By.css('#leases-table'))
        const cols = leasesTable.queryAll(By.css('td'))
        expect(cols.length).toBe(5)

        // Expand button existence.
        expect(cols[0].children.length).toBe(1)
        const expandButton = cols[0].children[0].nativeElement
        expect(expandButton.nodeName).toBe('A')

        // Lease properties.
        expect(cols[1].nativeElement.innerText).toBe('192.0.2.3')
        expect(cols[2].nativeElement.innerText).toBe('IPv4 address')
        expect(cols[3].nativeElement.innerText).toBe('Declined')
        expect(cols[4].nativeElement.innerText).toBe('kea@frog')

        // Validate app link.
        expect(cols[4].children.length).toBe(1)
        expect(cols[4].children[0].attributes.href).toBe('/apps/kea/1')

        // Simulate expanding the lease information.
        expandButton.click()
        fixture.detectChanges()

        // There should be one table holding the expanded information.
        // In particular, there should be no table with client identifiers
        // because they are not present for a declined lease.
        const tables = fixture.debugElement.queryAll(By.css('table'))
        expect(tables.length).toBe(2)

        // Find allocation and expiration time.
        const startDate = new Date(1616149050000)
        const endDate = new Date(1616149050000 + 3600000)

        // Expected data within the expanded row.
        const expectedLeaseData: any = [
            ['Subnet Identifier', '123'],
            ['Valid Lifetime', '3600 seconds'],
            ['Allocated at', datetimeToLocal(startDate)],
            ['Expires at', datetimeToLocal(endDate)],
        ]

        // Find rows.
        const rows = tables[1].queryAll(By.css('tr'))
        expect(rows.length).toBe(4)

        // For each table row, compare its contents with the expected data.
        let i = 0
        for (const row of rows) {
            expect(row.children.length).toBe(2)
            expect(row.children[0].nativeElement.innerText).toBe(expectedLeaseData[i][0] + ':')
            expect(row.children[1].nativeElement.innerText).toBe(expectedLeaseData[i][1])
            i++
        }

        // Test User Context JSON tree content.
        const tree = fixture.debugElement.queryAll(By.directive(JsonTreeRootComponent))
        expect(tree).not.toBeNull()
        expect(Object.keys(tree).length).toBe(0)

        // Test summary.
        const leasesSearchSummary = fixture.debugElement.query(By.css('#leases-search-summary-span'))
        expect(leasesSearchSummary.properties.innerText).toBe('Found 1 lease matching 192.0.2.3.')
    })

    it('should display DHCPv6 leases', () => {
        component.leases = [
            {
                id: 1,
                ipAddress: '2001:db8:1::1',
                leaseType: 'IA_NA',
                state: 1,
                appId: 2,
                appName: 'kea@ipv6',
                hwAddress: '01:02:03:04:05:06',
                duid: '01:02:03:04',
                hostname: 'faq.example.org',
                fqdnFwd: true,
                fqdnRev: false,
                subnetId: 234,
                iaid: 12,
                cltt: 1616149050,
                preferredLifetime: 900,
                validLifetime: 1800,
                userContext: { ISC: { 'client-classes': ['ALL', 'HA_primary', 'UNKNOWN'] } },
            },
            {
                id: 2,
                ipAddress: '3000::',
                prefixLength: 64,
                leaseType: 'IA_PD',
                state: 2,
                appId: 2,
                appName: 'kea@ipv6',
                duid: '01:02:03:05',
                subnetId: 345,
                iaid: 13,
                cltt: 1616149050,
                preferredLifetime: 900,
                validLifetime: 1800,
            },
        ]
        component.lastSearchText = '2001:db8:1::1'
        fixture.detectChanges()

        const leasesTable = fixture.debugElement.query(By.css('#leases-table'))
        const cols = leasesTable.queryAll(By.css('td'))
        expect(cols.length).toBe(10)

        // Address lease.

        // Expand button existence.
        expect(cols[0].children.length).toBe(1)
        const expandButton1 = cols[0].children[0].nativeElement
        expect(expandButton1.nodeName).toBe('A')

        // Lease properties.
        expect(cols[1].nativeElement.innerText).toBe('2001:db8:1::1')
        expect(cols[2].nativeElement.innerText).toBe('IPv6 address (IA_NA)')
        expect(cols[3].nativeElement.innerText).toBe('Declined')
        expect(cols[4].nativeElement.innerText).toBe('kea@ipv6')

        // Validate app link.
        expect(cols[4].children.length).toBe(1)
        expect(cols[4].children[0].attributes.href).toBe('/apps/kea/2')

        // Prefix lease.

        // Expand button existence.
        expect(cols[5].children.length).toBe(1)
        const expandButton2 = cols[5].children[0].nativeElement
        expect(expandButton2.nodeName).toBe('A')

        // Lease properties.
        expect(cols[6].nativeElement.innerText).toBe('3000::/64')
        expect(cols[7].nativeElement.innerText).toBe('IPv6 prefix (IA_PD)')
        expect(cols[8].nativeElement.innerText).toBe('Expired/Reclaimed')
        expect(cols[9].nativeElement.innerText).toBe('kea@ipv6')

        // Validate app link.
        expect(cols[9].children.length).toBe(1)
        expect(cols[9].children[0].attributes.href).toBe('/apps/kea/2')

        // Simulate expanding the lease information.
        expandButton1.click()
        fixture.detectChanges()
        expandButton2.click()
        fixture.detectChanges()

        // Find the table holding expanded information.
        const tables = fixture.debugElement.queryAll(By.css('table'))
        expect(tables.length).toBe(6)

        // Find allocation and expiration time.
        const startDate = new Date(1616149050000)
        const endDate = new Date(1616149050000 + 1800000)

        let expectedLeaseData: any = [
            [
                ['MAC address', '01:02:03:04:05:06'],
                ['DUID', '01:02:03:04'],
            ],
            [
                ['Subnet Identifier', '234'],
                ['IAID', '12'],
                ['Preferred Lifetime', '900 seconds'],
                ['Valid Lifetime', '1800 seconds'],
                ['Allocated at', datetimeToLocal(startDate)],
                ['Expires at', datetimeToLocal(endDate)],
            ],
            [
                ['Hostname', 'faq.example.org'],
                ['Forward DDNS', 'yes'],
                ['Reverse DDNS', 'no'],
            ],
        ]

        // Second and further tables should contain expanded lease information.
        // For each table check if the data is correct.
        let tableIndex = 1
        for (const expectedDataGroup of expectedLeaseData) {
            const rows = tables[tableIndex].queryAll(By.css('tr'))
            expect(rows.length).toBe(expectedDataGroup.length)

            // For each table row, compare its contents with the expected data.
            let i = 0
            for (const row of rows) {
                expect(row.children.length).toBe(2)
                expect(row.children[0].nativeElement.innerText).toBe(expectedDataGroup[i][0] + ':')
                expect(row.children[1].nativeElement.innerText).toContain(expectedDataGroup[i][1])
                i++
            }
            tableIndex++
        }

        // Test User Context JSON tree content.
        const tree = fixture.debugElement.queryAll(By.directive(JsonTreeRootComponent))
        expect(tree).not.toBeNull()
        expect(Object.keys(tree).length).toBe(1)
        const treeComponent = tree[0].componentInstance as JsonTreeComponent
        expect(treeComponent).not.toBeNull()
        expect(treeComponent.value).not.toBeNull()
        expect(Object.keys(treeComponent.value).length).toBe(1)
        expect(treeComponent.value['ISC']).not.toBeNull()
        expect(Object.keys(treeComponent.value['ISC']).length).toBe(1)
        expect(treeComponent.value['ISC']['client-classes']).not.toBeNull()
        expect(treeComponent.value['ISC']['client-classes'].length).toBe(3)
        expect(treeComponent.value['ISC']['client-classes'][0]).toBe('ALL')
        expect(treeComponent.value['ISC']['client-classes'][1]).toBe('HA_primary')
        expect(treeComponent.value['ISC']['client-classes'][2]).toBe('UNKNOWN')

        expectedLeaseData = [
            [['DUID', '01:02:03:05']],
            [
                ['Subnet Identifier', '345'],
                ['IAID', '13'],
                ['Preferred Lifetime', '900 seconds'],
                ['Valid Lifetime', '1800 seconds'],
                ['Allocated at', datetimeToLocal(startDate)],
                ['Expires at', datetimeToLocal(endDate)],
            ],
        ]

        // Fifth and sixth table should contain expanded lease information.
        // For each table check if the data is correct.
        tableIndex = 4
        for (const expectedDataGroup of expectedLeaseData) {
            const rows = tables[tableIndex].queryAll(By.css('tr'))
            expect(rows.length).toBe(expectedDataGroup.length)

            // For each table row, compare its contents with the expected data.
            let i = 0
            for (const row of rows) {
                expect(row.children.length).toBe(2)
                expect(row.children[0].nativeElement.innerText).toBe(expectedDataGroup[i][0] + ':')
                expect(row.children[1].nativeElement.innerText).toBe(expectedDataGroup[i][1])
                i++
            }
            tableIndex++
        }

        // Test summary.
        const leasesSearchSummary = fixture.debugElement.query(By.css('#leases-search-summary-span'))
        expect(leasesSearchSummary.properties.innerText).toBe('Found 2 leases matching 2001:db8:1::1.')
    })

    it('should display erred apps message', () => {
        component.erredApps = [
            {
                id: 1,
                name: 'kea@frog',
            },
            {
                id: 1,
                name: 'kea@frog',
            },
        ]
        component.lastSearchText = '192.0.2.3'
        fixture.detectChanges()

        // A warning message informing about erred apps should be displayed.
        const erredAppsMessage = fixture.debugElement.query(By.css('#erred-apps-message'))
        expect(erredAppsMessage).not.toBeNull()
    })

    it('should handle communication error', fakeAsync(() => {
        // Set erred apps to non-empty array.
        component.erredApps = [
            {
                id: 1,
                name: 'kea@frog',
            },
        ]
        // Do the same for leases.
        component.leases = [
            {
                id: 0,
                ipAddress: '192.0.2.3',
                state: 0,
                appId: 1,
                appName: 'kea@frog',
                hwAddress: '01:02:03:04:05:06',
                clientId: '01:02:03:04',
                hostname: 'faq.example.org',
                fqdnFwd: false,
                fqdnRev: true,
                subnetId: 123,
                cltt: 1616149050,
                validLifetime: 3600,
            },
        ]
        component.searchText = '192.0.2.0'

        // Simulate an error while getting leases from the server.
        spyOn(dhcpApi, 'getLeases').and.returnValue(throwError({ status: 404 }))

        // Spy on message service to ensure that error message is displayed.
        spyOn(msgService, 'add')

        component.searchLeases()
        tick()

        // The lease information and metadata should have been cleared.
        expect(component.leases.length).toBe(0)
        expect(component.erredApps.length).toBe(0)
        expect(component.searchStatus).toBe(component.Status.Searched)

        // An error message should have been displayed.
        expect(msgService.add).toHaveBeenCalled()
    }))

    it('should display error message for partial IPv4 address', () => {
        const searchInput = fixture.debugElement.query(By.css('#leases-search-input'))
        const searchInputElement = searchInput.nativeElement

        // Searching by partial addresses is not supported. Hint should be
        // displayed when currently typed text is recognized as partial
        // IPv4 address.
        const partialAddresses = ['192.', '192.255', '192.255.', '192.255.0', '192.255.0.']

        for (const partial of partialAddresses) {
            searchInputElement.value = partial
            searchInputElement.dispatchEvent(new Event('input'))
            searchInputElement.dispatchEvent(new KeyboardEvent('keyup', { key: 'Enter' }))
            fixture.detectChanges()

            // Ensure that the hint is displayed.
            const inputError = fixture.debugElement.query(By.css('#leases-search-input-error'))
            expect(inputError).not.toBeNull()
            expect(inputError.properties.innerText).toBe('Please enter the complete IPv4 address.')
            expect(component.invalidSearchText).toBeTrue()
        }

        // Text consisting of digits and full IPv4 address are valid.
        const validTexts = ['192', '192.0.2.1']

        for (const valid of validTexts) {
            searchInputElement.value = valid
            searchInputElement.dispatchEvent(new Event('input'))
            searchInputElement.dispatchEvent(new KeyboardEvent('keyup', { key: 'Enter' }))
            fixture.detectChanges()

            // Ensure that the hint is not displayed.
            const inputError = fixture.debugElement.query(By.css('#leases-search-input-error'))
            expect(inputError).toBeNull()
            expect(component.invalidSearchText).toBeFalse()
        }
    })

    it('should display error message for wrong use of colons', () => {
        const searchInput = fixture.debugElement.query(By.css('#leases-search-input'))
        const searchInputElement = searchInput.nativeElement

        const invalidTexts = [
            { text: '00:', error: 'Invalid trailing colon.' },
            { text: ':00', error: 'Invalid leading colon.' },
            { text: '::00:', error: 'Invalid trailing colon.' },
            { text: ':::', error: 'Invalid multiple consecutive colons.' },
            { text: '20:::34', error: 'Invalid multiple consecutive colons.' },
            { text: '20::::34', error: 'Invalid multiple consecutive colons.' },
            { text: '20: 34::', error: 'Invalid whitespace near a colon.' },
            { text: '20 :34', error: 'Invalid whitespace near a colon.' },
            { text: '2001::db8::1', error: 'Invalid IPv6 address.' },
        ]

        for (const invalid of invalidTexts) {
            searchInputElement.value = invalid.text
            searchInputElement.dispatchEvent(new Event('input'))
            searchInputElement.dispatchEvent(new KeyboardEvent('keyup', { key: 'Enter' }))
            fixture.detectChanges()

            // Ensure that the hint is displayed.
            const inputError = fixture.debugElement.query(By.css('#leases-search-input-error'))
            expect(inputError).not.toBeNull()
            expect(inputError.properties.innerText).toBe(invalid.error)
            expect(component.invalidSearchText).toBeTrue()
        }

        // Make sure that valid search text is accepted.
        const validTexts = ['::', '::1', '2001:db8:1::', '2001:db8::0']

        for (const valid of validTexts) {
            searchInputElement.value = valid
            searchInputElement.dispatchEvent(new Event('input'))
            searchInputElement.dispatchEvent(new KeyboardEvent('keyup', { key: 'Enter' }))
            fixture.detectChanges()

            // Ensure that the hint is not displayed.
            const inputError = fixture.debugElement.query(By.css('#leases-search-input-error'))
            expect(inputError).toBeNull()
        }
    })

    it('should display error message when invalid state is specified', () => {
        const searchInput = fixture.debugElement.query(By.css('#leases-search-input'))
        const searchInputElement = searchInput.nativeElement

        const invalidTexts = [
            { text: 'state:', error: 'Specify lease state.' },
            { text: 'state:default', error: 'Searching leases in the default state is unsupported.' },
            {
                text: 'state:expired-reclaimed',
                error: 'Searching leases in the expired-reclaimed state is unsupported.',
            },
            { text: 'state:dec', error: 'Use state:declined to search declined leases.' },
        ]

        for (const invalid of invalidTexts) {
            searchInputElement.value = invalid.text
            searchInputElement.dispatchEvent(new Event('input'))
            searchInputElement.dispatchEvent(new KeyboardEvent('keyup', { key: 'Enter' }))
            fixture.detectChanges()

            // Ensure that the hint is displayed.
            const inputError = fixture.debugElement.query(By.css('#leases-search-input-error'))
            expect(inputError).not.toBeNull()
            expect(inputError.properties.innerText).toBe(invalid.error)
            expect(component.invalidSearchText).toBeTrue()
        }

        // Make sure that valid search text is accepted.
        const validTexts = ['state:declined', 'state: declined']

        for (const valid of validTexts) {
            searchInputElement.value = valid
            searchInputElement.dispatchEvent(new Event('input'))
            searchInputElement.dispatchEvent(new KeyboardEvent('keyup', { key: 'Enter' }))
            fixture.detectChanges()

            // Ensure that the hint is not displayed.
            const inputError = fixture.debugElement.query(By.css('#leases-search-input-error'))
            expect(inputError).toBeNull()
        }
    })

    it('should trigger lease search when valid text query parameter is specified', fakeAsync(() => {
        spyOn(dhcpApi, 'getLeases').and.returnValue(throwError({ status: 404 }))
        router.navigate(['/dhcp/leases'], { queryParams: { text: 'abc' } })
        tick()
        component.ngOnInit()
        expect(dhcpApi.getLeases).toHaveBeenCalled()
    }))

    it('should not start lease search when invalid text query parameter is specified', fakeAsync(() => {
        spyOn(dhcpApi, 'getLeases').and.returnValue(throwError({ status: 404 }))
        // Specify partial IP address. The search should not be triggered and an
        // error message should be shown.
        router.navigate(['/dhcp/leases'], { queryParams: { text: '192.0.2' } })
        tick()
        component.ngOnInit()
        expect(dhcpApi.getLeases).not.toHaveBeenCalled()
        expect(component.invalidSearchText).toBeTrue()
    }))

    it('should clear empty text parameter', fakeAsync(() => {
        spyOn(dhcpApi, 'getLeases').and.returnValue(throwError({ status: 404 }))
        router.navigate(['/dhcp/leases'], { queryParams: { text: '   ' } })
        tick()
        component.ngOnInit()
        tick()
        expect(dhcpApi.getLeases).not.toHaveBeenCalled()
        expect(component.invalidSearchText).toBeFalse()
        expect(route.snapshot.queryParamMap.has('text')).toBeFalse()
    }))

    it('should have breadcrumbs', () => {
        const breadcrumbsElement = fixture.debugElement.query(By.directive(BreadcrumbsComponent))
        expect(breadcrumbsElement).not.toBeNull()
        const breadcrumbsComponent = breadcrumbsElement.componentInstance as BreadcrumbsComponent
        expect(breadcrumbsComponent).not.toBeNull()
        expect(breadcrumbsComponent.items).toHaveSize(2)
        expect(breadcrumbsComponent.items[0].label).toEqual('DHCP')
        expect(breadcrumbsComponent.items[1].label).toEqual('Lease Search')
    })
})
