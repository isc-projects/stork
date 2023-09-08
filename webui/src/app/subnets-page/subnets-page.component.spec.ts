import { By } from '@angular/platform-browser'
import { ComponentFixture, TestBed, waitForAsync } from '@angular/core/testing'

import { SubnetsPageComponent } from './subnets-page.component'
import { FormsModule } from '@angular/forms'
import { DropdownModule } from 'primeng/dropdown'
import { TableModule } from 'primeng/table'
import { SubnetBarComponent } from '../subnet-bar/subnet-bar.component'
import { TooltipModule } from 'primeng/tooltip'
import { ActivatedRoute, Router } from '@angular/router'
import { DHCPService, SettingsService, Subnet, UsersService } from '../backend'
import { HttpClientTestingModule } from '@angular/common/http/testing'
import { of } from 'rxjs'
import { MessageService } from 'primeng/api'
import { BreadcrumbsComponent } from '../breadcrumbs/breadcrumbs.component'
import { HelpTipComponent } from '../help-tip/help-tip.component'
import { BreadcrumbModule } from 'primeng/breadcrumb'
import { OverlayPanelModule } from 'primeng/overlaypanel'
import { NoopAnimationsModule } from '@angular/platform-browser/animations'
import { DelegatedPrefixBarComponent } from '../delegated-prefix-bar/delegated-prefix-bar.component'
import { HumanCountComponent } from '../human-count/human-count.component'
import { NumberPipe } from '../pipes/number.pipe'
import { RouterTestingModule } from '@angular/router/testing'
import { MessageModule } from 'primeng/message'
import { HumanCountPipe } from '../pipes/human-count.pipe'
import { TabMenuModule } from 'primeng/tabmenu'
import { EntityLinkComponent } from '../entity-link/entity-link.component'
import { AddressPoolBarComponent } from '../address-pool-bar/address-pool-bar.component'
import { MockParamMap } from '../utils'

describe('SubnetsPageComponent', () => {
    let component: SubnetsPageComponent
    let fixture: ComponentFixture<SubnetsPageComponent>
    let dhcpService: DHCPService
    let router: Router

    beforeEach(waitForAsync(() => {
        TestBed.configureTestingModule({
            providers: [
                DHCPService,
                UsersService,
                MessageService,
                SettingsService,
                {
                    provide: ActivatedRoute,
                    useValue: {
                        snapshot: { queryParamMap: new MockParamMap() },
                        queryParamMap: of(new MockParamMap()),
                        paramMap: of(new MockParamMap()),
                    },
                },
                RouterTestingModule,
            ],
            imports: [
                FormsModule,
                DropdownModule,
                TableModule,
                TooltipModule,
                RouterTestingModule,
                HttpClientTestingModule,
                BreadcrumbModule,
                OverlayPanelModule,
                NoopAnimationsModule,
                MessageModule,
                TabMenuModule,
            ],
            declarations: [
                SubnetsPageComponent,
                SubnetBarComponent,
                BreadcrumbsComponent,
                HelpTipComponent,
                DelegatedPrefixBarComponent,
                HumanCountComponent,
                HumanCountPipe,
                NumberPipe,
                EntityLinkComponent,
                AddressPoolBarComponent,
            ],
        })
        dhcpService = TestBed.inject(DHCPService)
        router = TestBed.inject(Router)
    }))

    beforeEach(() => {
        const fakeResponses: any = [
            {
                items: [
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
                                pools: ['1.0.0.4-1.0.255.254'],
                            },
                            {
                                appId: 28,
                                appName: 'kea2@localhost',
                                // Misconfiguration,  all local subnets in a
                                // subnet should share the same subnet ID. In
                                // this case, we display a value from the first
                                // local subnet.
                                id: 2,
                                machineAddress: 'host',
                                machineHostname: 'lv-pc2',
                                pools: ['1.0.0.4-1.0.255.254'],
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
                        subnet: '1.0.0.0/16',
                    },
                    {
                        clientClass: 'class-00-01',
                        id: 42,
                        localSubnets: [
                            {
                                appId: 27,
                                appName: 'kea@localhost',
                                machineAddress: 'localhost',
                                machineHostname: 'lv-pc',
                                pools: ['1.1.0.4-1.1.255.254'],
                            },
                        ],
                        statsCollectedAt: null,
                        subnet: '1.1.0.0/16',
                    },
                    {
                        id: 67,
                        localSubnets: [
                            {
                                id: 4,
                                appId: 28,
                                appName: 'ha@localhost',
                                machineAddress: 'localhost',
                                machineHostname: 'ha-cluster-1',
                            },
                            {
                                id: 4,
                                appId: 28,
                                appName: 'ha@localhost',
                                machineAddress: 'localhost',
                                machineHostname: 'ha-cluster-2',
                            },
                            {
                                id: 4,
                                appId: 28,
                                appName: 'ha@localhost',
                                machineAddress: 'localhost',
                                machineHostname: 'ha-cluster-3',
                            },
                        ],
                        statsCollectedAt: '2022-01-16T14:16:00.000Z',
                        subnet: '1.1.1.0/24',
                    },
                ],
                total: 10496,
            },
            {
                items: [
                    {
                        clientClass: 'class-00-00',
                        id: 5,
                        localSubnets: [
                            {
                                appId: 28,
                                appName: 'kea2@localhost',
                                id: 2,
                                machineAddress: 'host',
                                machineHostname: 'lv-pc2',
                                pools: ['1.0.0.4-1.0.255.254'],
                            },
                        ],
                        statsCollectedAt: '1970-01-01T12:00:00.0Z',
                        subnet: '1.0.0.0/16',
                    },
                ],
                total: 10496,
            },
        ]
        spyOn(dhcpService, 'getSubnets').and.returnValues(
            // The subnets are fetched twice before the unit test starts.
            of(fakeResponses[0]),
            of(fakeResponses[0]),
            of(fakeResponses[1])
        )

        fixture = TestBed.createComponent(SubnetsPageComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should convert statistics to big integers', async () => {
        // Act
        await fixture.whenStable()

        // Assert
        const stats: { [key: string]: BigInt } = component.subnets[0].stats as any
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

    it('should not fail on empty statistics', async () => {
        // Act
        component.loadSubnets({})
        await fixture.whenStable()

        // Assert
        expect(component.subnets[0].stats).toBeUndefined()
        // No throw
    })

    it('should have breadcrumbs', () => {
        const breadcrumbsElement = fixture.debugElement.query(By.directive(BreadcrumbsComponent))
        expect(breadcrumbsElement).not.toBeNull()
        const breadcrumbsComponent = breadcrumbsElement.componentInstance as BreadcrumbsComponent
        expect(breadcrumbsComponent).not.toBeNull()
        expect(breadcrumbsComponent.items).toHaveSize(2)
        expect(breadcrumbsComponent.items[0].label).toEqual('DHCP')
        expect(breadcrumbsComponent.items[1].label).toEqual('Subnets')
    })

    it('should detect IPv6 subnets', () => {
        const subnets: Subnet[] = [{ subnet: '10.0.0.0/8' }, { subnet: '192.168.0.0/16' }]

        component.subnets = subnets
        expect(component.isAnyIPv6SubnetVisible).toBeFalse()

        subnets.push({ subnet: 'fe80::/64' })
        expect(component.isAnyIPv6SubnetVisible).toBeTrue()
    })

    it('should display the Kea subnet ID', async () => {
        // Act
        await fixture.whenStable()
        fixture.detectChanges()
        await fixture.whenRenderingDone()

        // Assert
        const cells = fixture.debugElement.queryAll(By.css('table tbody tr td:last-child'))
        expect(cells.length).toBe(3)
        const cellValues = cells.map((c) => (c.nativeElement as HTMLElement).textContent.trim())
        // First subnet has various Kea subnet IDs.
        expect(cellValues).toContain('1  2 Inconsistent IDs')
        // Second subnet misses the Kea subnet ID.
        expect(cellValues).toContain('')
        // Third subnet has identical Kea subnet IDs.
        expect(cellValues).toContain('4')
    })

    it('should filter subnets by the Kea subnet ID', async () => {
        // Arrange
        const input = fixture.debugElement.query(By.css('#filter-subnets-text-field'))
        const spy = spyOn(router, 'navigate')

        // Act
        await fixture.whenStable()

        component.filterText = 'subnetId:1'
        input.triggerEventHandler('keyup', null)

        await fixture.whenStable()

        // Assert
        expect(spy).toHaveBeenCalledOnceWith(
            ['/dhcp/subnets'],
            jasmine.objectContaining({
                queryParams: {
                    text: null,
                    subnetId: '1',
                    appId: null,
                },
            })
        )
    })

    it('should detect that the subnet has only references to the local subnets with identical IDs', () => {
        // Arrange
        const subnet: Subnet = {
            subnet: 'fe80::/64',
            localSubnets: [
                {
                    id: 1,
                },
                {
                    id: 1,
                },
            ],
        }

        // Act & Assert
        expect(component.hasAssignedMultipleKeaSubnetIds(subnet)).toBeFalse()
    })

    it('should detect that the subnet has references to the local subnets with various IDs', () => {
        // Arrange
        const subnet: Subnet = {
            subnet: 'fe80::/64',
            localSubnets: [
                {
                    id: 1,
                },
                {
                    id: 2,
                },
            ],
        }

        // Act & Assert
        expect(component.hasAssignedMultipleKeaSubnetIds(subnet)).toBeTrue()
    })

    it('should recognize that all local subnets have the same IDs if the local subnets list is empty', () => {
        // Arrange
        const subnet: Subnet = {
            subnet: 'fe80::/64',
            localSubnets: [],
        }

        // Act & Assert
        expect(component.hasAssignedMultipleKeaSubnetIds(subnet)).toBeFalse()
    })
})
