import { ComponentFixture, TestBed } from '@angular/core/testing'

import { SharedNetworksTableComponent } from './shared-networks-table.component'
import { MessageService } from 'primeng/api'
import { HelpTipComponent } from '../help-tip/help-tip.component'
import { PluralizePipe } from '../pipes/pluralize.pipe'
import { TableModule } from 'primeng/table'
import { HttpClientTestingModule } from '@angular/common/http/testing'
import { ButtonModule } from 'primeng/button'
import { OverlayPanelModule } from 'primeng/overlaypanel'
import { InputNumberModule } from 'primeng/inputnumber'
import { FormsModule } from '@angular/forms'
import { PanelModule } from 'primeng/panel'
import { BrowserAnimationsModule } from '@angular/platform-browser/animations'
import { TagModule } from 'primeng/tag'
import { DropdownModule } from 'primeng/dropdown'
import { convertToParamMap, RouterModule } from '@angular/router'
import { SharedNetworksPageComponent } from '../shared-networks-page/shared-networks-page.component'
import { DHCPService, SharedNetwork } from '../backend'
import { of } from 'rxjs'
import { EntityLinkComponent } from '../entity-link/entity-link.component'
import { HumanCountComponent } from '../human-count/human-count.component'
import { SubnetBarComponent } from '../subnet-bar/subnet-bar.component'
import { TooltipModule } from 'primeng/tooltip'
import { HumanCountPipe } from '../pipes/human-count.pipe'
import { LocalNumberPipe } from '../pipes/local-number.pipe'
import { By } from '@angular/platform-browser'

describe('SharedNetworksTableComponent', () => {
    let component: SharedNetworksTableComponent
    let fixture: ComponentFixture<SharedNetworksTableComponent>
    let dhcpService: DHCPService
    let getNetworksSpy: jasmine.Spy<any>

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            providers: [MessageService],
            imports: [
                TableModule,
                HttpClientTestingModule,
                ButtonModule,
                OverlayPanelModule,
                InputNumberModule,
                FormsModule,
                PanelModule,
                BrowserAnimationsModule,
                TagModule,
                DropdownModule,
                RouterModule.forRoot([
                    {
                        path: 'dhcp/shared-networks',
                        pathMatch: 'full',
                        redirectTo: 'dhcp/shared-networks/all',
                    },
                    {
                        path: 'dhcp/shared-networks/:id',
                        component: SharedNetworksPageComponent,
                    },
                ]),
                TooltipModule,
            ],
            declarations: [
                SharedNetworksTableComponent,
                HelpTipComponent,
                PluralizePipe,
                EntityLinkComponent,
                HumanCountComponent,
                SubnetBarComponent,
                HumanCountPipe,
                LocalNumberPipe,
            ],
        }).compileComponents()

        fixture = TestBed.createComponent(SharedNetworksTableComponent)
        dhcpService = TestBed.inject(DHCPService)
        component = fixture.componentInstance

        const fakeResponses: any[] = [
            {
                items: [
                    {
                        id: 1,
                        name: 'frog',
                        subnets: [
                            {
                                clientClass: 'class-00-00',
                                id: 5,
                                localSubnets: [
                                    {
                                        appId: 27,
                                        appName: 'kea@localhost',
                                        id: 1,
                                        machineAddress: 'localhost',
                                        machineHostname: 'lv-pc',
                                        pools: [
                                            {
                                                pool: '1.0.0.4-1.0.255.254',
                                            },
                                        ],
                                    },
                                ],
                                subnet: '1.0.0.0/16',
                                statsCollectedAt: '2023-02-17T13:06:00.2134Z',
                                stats: {
                                    'assigned-addresses': '42',
                                    'total-addresses':
                                        '12345678901234567890123456789012345678901234567890123456789012345678901234567890',
                                    'declined-addresses': '0',
                                },
                            },
                        ],
                        stats: {
                            'assigned-addresses':
                                '12345678901234567890123456789012345678901234567890123456789012345678901234567890',
                            'total-addresses':
                                '1234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890',
                            'declined-addresses': '-2',
                        },
                        statsCollectedAt: '2022-01-19T12:10:22.513Z',
                    },
                ],
                total: 10496,
            },
            {
                items: [
                    {
                        id: 2,
                        name: 'frog-no-stats',
                        subnets: [
                            {
                                clientClass: 'class-00-00',
                                id: 5,
                                localSubnets: [
                                    {
                                        appId: 27,
                                        appName: 'kea@localhost',
                                        id: 1,
                                        machineAddress: 'localhost',
                                        machineHostname: 'lv-pc',
                                        pools: [
                                            {
                                                pool: '1.0.0.4-1.0.255.254',
                                            },
                                        ],
                                    },
                                ],
                                subnet: '1.0.0.0/16',
                            },
                        ],
                        statsCollectedAt: '1970-01-01T12:00:00.0Z',
                    },
                ],
                total: 10496,
            },
            {
                items: [
                    {
                        id: 3,
                        name: 'cat',
                        subnets: [
                            // Subnet represented by the double utilization bar.
                            {
                                clientClass: 'class-00-00',
                                id: 5,
                                localSubnets: [
                                    {
                                        appId: 27,
                                        appName: 'kea@localhost',
                                        id: 1,
                                        machineAddress: 'localhost',
                                        machineHostname: 'lv-pc',
                                    },
                                ],
                                subnet: 'fe80:1::/64',
                                statsCollectedAt: '2023-03-03T10:51:00.0000Z',
                                stats: {
                                    'assigned-nas': '42',
                                    'total-nas':
                                        '12345678901234567890123456789012345678901234567890123456789012345678901234567890',
                                    'declined-nas': '0',
                                    'assigned-pds': '24',
                                    'total-pds':
                                        '9012345678901234567890123456789012345678901234567890123456789012345678901234567890',
                                },
                                addrUtilization: 10,
                                pdUtilization: 15,
                            },
                            // Subnet represented by the single NA utilization bar.
                            {
                                clientClass: 'class-00-00',
                                id: 6,
                                localSubnets: [
                                    {
                                        appId: 27,
                                        appName: 'kea@localhost',
                                        id: 1,
                                        machineAddress: 'localhost',
                                        machineHostname: 'lv-pc',
                                    },
                                ],
                                subnet: 'fe80:2::/64',
                                statsCollectedAt: '2023-03-03T10:51:00.0000Z',
                                stats: {
                                    'assigned-nas': '42',
                                    'total-nas':
                                        '12345678901234567890123456789012345678901234567890123456789012345678901234567890',
                                    'declined-nas': '0',
                                    'assigned-pds': '0',
                                    'total-pds': '0',
                                },
                                addrUtilization: 20,
                                pdUtilization: 0,
                            },
                            // Subnet represented by the single PD utilization bar.
                            {
                                clientClass: 'class-00-00',
                                id: 7,
                                localSubnets: [
                                    {
                                        appId: 27,
                                        appName: 'kea@localhost',
                                        id: 1,
                                        machineAddress: 'localhost',
                                        machineHostname: 'lv-pc',
                                    },
                                ],
                                subnet: 'fe80:3::/64',
                                statsCollectedAt: '2023-03-03T10:51:00.0000Z',
                                stats: {
                                    'assigned-nas': '0',
                                    'total-nas': '0',
                                    'declined-nas': '0',
                                    'assigned-pds': '0',
                                    'total-pds':
                                        '9012345678901234567890123456789012345678901234567890123456789012345678901234567890',
                                },
                                addrUtilization: 0,
                                pdUtilization: 35,
                            },
                            // Subnet represented by the double utilization bar
                            {
                                clientClass: 'class-00-00',
                                id: 8,
                                localSubnets: [
                                    {
                                        appId: 27,
                                        appName: 'kea@localhost',
                                        id: 2,
                                        machineAddress: 'localhost',
                                        machineHostname: 'lv-pc',
                                    },
                                ],
                                subnet: 'fe80:4::/64',
                                statsCollectedAt: '2023-03-03T10:51:00.0000Z',
                                stats: {
                                    'assigned-nas': '0',
                                    'total-nas': '0',
                                    'declined-nas': '0',
                                    'assigned-pds': '0',
                                    'total-pds': '0',
                                },
                                addrUtilization: 0,
                                pdUtilization: 0,
                            },
                        ],
                        statsCollectedAt: '1970-01-01T12:00:00.0Z',
                    },
                ],
                total: 10496,
            },
        ]
        getNetworksSpy = spyOn(dhcpService, 'getSharedNetworks')
        // Prepare response when no filtering is applied.
        getNetworksSpy.withArgs(0, 10, null, null, null).and.returnValue(of(fakeResponses[0]))
        // Prepare response when shared networks are filtered by text to get an item without stats.
        getNetworksSpy.withArgs(0, 10, null, null, 'frog-no-stats').and.returnValue(of(fakeResponses[1]))
        // Prepare response when shared networks are filtered by text to get an item with 4 subnets.
        getNetworksSpy.withArgs(0, 10, null, null, 'cat').and.returnValue(of(fakeResponses[2]))
        // Prepare responses for table filtering tests.
        getNetworksSpy.withArgs(0, 10, 5, null, 'cat').and.returnValue(of(fakeResponses[2]))
        getNetworksSpy.withArgs(0, 10, 5, 6, 'cat').and.returnValue(of(fakeResponses[2]))

        fixture.detectChanges()

        component.clearFilters(component.table)
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should load initial data', async () => {
        // Data loading should be in progress.
        expect(component.dataLoading).toBeTrue()

        await fixture.whenStable()
        fixture.detectChanges()

        // Data loading should be done.
        expect(getNetworksSpy).toHaveBeenCalledOnceWith(0, 10, null, null, null)
        expect(component.dataLoading).toBeFalse()
        // Records count should be updated.
        expect(component.totalRecords).toBe(10496)
    })

    it('should convert shared network statistics to big integers', async () => {
        // Act
        await fixture.whenStable()
        fixture.detectChanges()

        // Assert
        expect(getNetworksSpy).toHaveBeenCalledOnceWith(0, 10, null, null, null)
        const stats: { [key: string]: BigInt } = component.dataCollection[0].stats as any
        expect(stats['assigned-addresses']).toBe(
            BigInt('12345678901234567890123456789012345678901234567890123456789012345678901234567890')
        )
        expect(stats['total-addresses']).toBe(
            BigInt(
                '1234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890'
            )
        )
        expect(stats['declined-addresses']).toBe(BigInt('-2'))
    })

    it('should convert subnet statistics to big integers', async () => {
        // Act
        await fixture.whenStable()
        fixture.detectChanges()

        // Assert
        expect(getNetworksSpy).toHaveBeenCalledOnceWith(0, 10, null, null, null)
        const stats: { [key: string]: BigInt } = component.dataCollection[0].subnets[0].stats as any
        expect(stats['assigned-addresses']).toBe(BigInt('42'))
        expect(stats['total-addresses']).toBe(
            BigInt('12345678901234567890123456789012345678901234567890123456789012345678901234567890')
        )
        expect(stats['declined-addresses']).toBe(BigInt('0'))
    })

    it('should not fail on empty statistics', async () => {
        // Act
        // Filter by text to get subnet without stats.
        component.updateFilterFromQueryParameters(convertToParamMap({ text: 'frog-no-stats' }))
        await fixture.whenStable()
        fixture.detectChanges()

        // Assert
        expect(getNetworksSpy).toHaveBeenCalledWith(0, 10, null, null, 'frog-no-stats')
        expect(component.dataCollection[0].stats).toBeUndefined()
        // No throw
    })

    it('should detect IPv6 subnets', () => {
        const networks: SharedNetwork[] = [
            {
                subnets: [{ subnet: '10.0.0.0/8' }, { subnet: '192.168.0.0/16' }],
            },
        ]

        component.dataCollection = networks
        expect(component.isAnyIPv6SubnetVisible).toBeFalse()

        networks.push({
            subnets: [{ subnet: 'fe80::/64' }],
        })
        expect(component.isAnyIPv6SubnetVisible).toBeTrue()
    })

    it('should display proper utilization bars', async () => {
        // Filter by text to get shared network with proper data.
        component.updateFilterFromQueryParameters(convertToParamMap({ text: 'cat' }))
        await fixture.whenStable()
        fixture.detectChanges()

        expect(getNetworksSpy).toHaveBeenCalledWith(0, 10, null, null, 'cat')
        expect(component.dataCollection.length).toBe(1)
        expect(component.dataCollection[0].subnets.length).toBe(4)

        const barElements = fixture.debugElement.queryAll(By.directive(SubnetBarComponent))
        expect(barElements.length).toBe(4)

        for (let i = 0; i < barElements.length; i++) {
            const barElement = barElements[i]
            const bar: SubnetBarComponent = barElement.componentInstance
            expect(bar.isIPv6).toBeTrue()

            switch (i) {
                case 0:
                    expect(bar.hasZeroAddressStats).toBeFalse()
                    expect(bar.hasZeroDelegatedPrefixStats).toBeFalse()
                    expect(bar.addrUtilization).toBe(10)
                    expect(bar.pdUtilization).toBe(15)
                    break
                case 1:
                    expect(bar.hasZeroAddressStats).toBeFalse()
                    expect(bar.hasZeroDelegatedPrefixStats).toBeTrue()
                    expect(bar.addrUtilization).toBe(20)
                    expect(bar.pdUtilization).toBe(0)
                    break
                case 2:
                    expect(bar.hasZeroAddressStats).toBeTrue()
                    expect(bar.hasZeroDelegatedPrefixStats).toBeFalse()
                    expect(bar.addrUtilization).toBe(0)
                    expect(bar.pdUtilization).toBe(35)
                    break
                case 3:
                    expect(bar.hasZeroAddressStats).toBeTrue()
                    expect(bar.hasZeroDelegatedPrefixStats).toBeTrue()
                    expect(bar.addrUtilization).toBe(0)
                    expect(bar.pdUtilization).toBe(0)
                    break
            }
        }
    })

    it('should display error about wrong query params filter', async () => {
        // Filter with query params that have wrong syntax.
        component.updateFilterFromQueryParameters(convertToParamMap({ appId: 'xyz', dhcpVersion: 7 }))
        await fixture.whenStable()
        fixture.detectChanges()

        // Check that correct error feedback is displayed.
        const errors = fixture.debugElement.queryAll(By.css('small.p-error'))
        expect(errors).toBeTruthy()
        expect(errors.length).toBe(2)
        expect(errors[0].nativeElement.innerText).toBe('Please specify appId as a number (e.g., appId=4).')
        expect(errors[1].nativeElement.innerText).toBe('Filter dhcpVersion allows only values: 4, 6.')
    })

    it('should filter table records', async () => {
        // Initial data was loaded.
        await fixture.whenStable()
        fixture.detectChanges()
        expect(getNetworksSpy).toHaveBeenCalledWith(0, 10, null, null, null)

        // Get filter inputs.
        const filterInputs = fixture.debugElement.queryAll(By.css('.p-column-filter input'))
        expect(filterInputs).toBeTruthy()

        // First is filter by appId, second is text search filter.
        expect(filterInputs.length).toBe(2)
        let input = filterInputs[1].nativeElement

        // Filter by text.
        input.value = 'cat'
        input.dispatchEvent(new Event('input'))

        // Verify that the API was called for that filter.
        await fixture.whenStable()
        fixture.detectChanges()
        expect(getNetworksSpy).toHaveBeenCalledWith(0, 10, null, null, 'cat')

        // Filter by kea app id.
        const pressFiveEvent = new KeyboardEvent('event', {
            code: 'Digit5',
            key: '5',
            keyCode: '5'.charCodeAt(0),
        })
        const inputNumberEls = fixture.debugElement.queryAll(By.css('.p-column-filter p-inputnumber'))
        expect(inputNumberEls).toBeTruthy()
        expect(inputNumberEls.length).toBe(1)
        let inputComponent = inputNumberEls[0].componentInstance
        inputComponent.onInputKeyPress(pressFiveEvent)

        // Verify that the API was called for that filter.
        await fixture.whenStable()
        fixture.detectChanges()
        expect(getNetworksSpy).toHaveBeenCalledWith(0, 10, 5, null, 'cat')

        // Filter by DHCP version.
        const dropdownContainer = fixture.debugElement.query(By.css('.p-column-filter .p-dropdown')).nativeElement
        dropdownContainer.click()
        await fixture.whenStable()
        fixture.detectChanges()
        const items = fixture.debugElement.query(By.css('.p-dropdown-items'))
        // Click second option.
        items.children[1].children[0].nativeElement.click()

        // Verify that the API was called for that filter.
        await fixture.whenStable()
        fixture.detectChanges()
        expect(getNetworksSpy).toHaveBeenCalledWith(0, 10, 5, 6, 'cat')
    })
})
