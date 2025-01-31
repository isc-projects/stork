import { By } from '@angular/platform-browser'
import { ComponentFixture, TestBed, fakeAsync, tick, waitForAsync } from '@angular/core/testing'

import { SharedNetworksPageComponent } from './shared-networks-page.component'
import { FormsModule, ReactiveFormsModule } from '@angular/forms'
import { DropdownModule } from 'primeng/dropdown'
import { TableModule } from 'primeng/table'
import { TooltipModule } from 'primeng/tooltip'
import { SubnetBarComponent } from '../subnet-bar/subnet-bar.component'
import {
    ActivatedRoute,
    ActivatedRouteSnapshot,
    convertToParamMap,
    NavigationEnd,
    Router,
    RouterModule,
} from '@angular/router'
import { DHCPService, SharedNetwork } from '../backend'
import { HttpClientTestingModule } from '@angular/common/http/testing'
import { BehaviorSubject, of } from 'rxjs'
import { BreadcrumbsComponent } from '../breadcrumbs/breadcrumbs.component'
import { HelpTipComponent } from '../help-tip/help-tip.component'
import { BreadcrumbModule } from 'primeng/breadcrumb'
import { OverlayPanelModule } from 'primeng/overlaypanel'
import { NoopAnimationsModule } from '@angular/platform-browser/animations'
import { HumanCountComponent } from '../human-count/human-count.component'
import { HumanCountPipe } from '../pipes/human-count.pipe'
import { LocalNumberPipe } from '../pipes/local-number.pipe'
import { HttpEvent } from '@angular/common/http'
import { EntityLinkComponent } from '../entity-link/entity-link.component'
import { ConfirmationService, MessageService } from 'primeng/api'
import { TabMenu, TabMenuModule } from 'primeng/tabmenu'
import { SharedNetworkTabComponent } from '../shared-network-tab/shared-network-tab.component'
import { FieldsetModule } from 'primeng/fieldset'
import { UtilizationStatsChartComponent } from '../utilization-stats-chart/utilization-stats-chart.component'
import { UtilizationStatsChartsComponent } from '../utilization-stats-charts/utilization-stats-charts.component'
import { AddressPoolBarComponent } from '../address-pool-bar/address-pool-bar.component'
import { DelegatedPrefixBarComponent } from '../delegated-prefix-bar/delegated-prefix-bar.component'
import { DividerModule } from 'primeng/divider'
import { ChartModule } from 'primeng/chart'
import { PlaceholderPipe } from '../pipes/placeholder.pipe'
import { TabType } from '../tab'
import { SharedNetworkFormComponent } from '../shared-network-form/shared-network-form.component'
import { ProgressSpinnerModule } from 'primeng/progressspinner'
import { ButtonModule } from 'primeng/button'
import { MultiSelectModule } from 'primeng/multiselect'
import { SharedParametersFormComponent } from '../shared-parameters-form/shared-parameters-form.component'
import { CheckboxModule } from 'primeng/checkbox'
import { DhcpOptionSetFormComponent } from '../dhcp-option-set-form/dhcp-option-set-form.component'
import { DhcpOptionFormComponent } from '../dhcp-option-form/dhcp-option-form.component'
import { InputNumberModule } from 'primeng/inputnumber'
import { ArrayValueSetFormComponent } from '../array-value-set-form/array-value-set-form.component'
import { ChipsModule } from 'primeng/chips'
import { DhcpClientClassSetFormComponent } from '../dhcp-client-class-set-form/dhcp-client-class-set-form.component'
import { ConfirmDialogModule } from 'primeng/confirmdialog'
import { SharedNetworksTableComponent } from '../shared-networks-table/shared-networks-table.component'
import { PanelModule } from 'primeng/panel'
import { PluralizePipe } from '../pipes/pluralize.pipe'
import { TagModule } from 'primeng/tag'
import { PositivePipe } from '../pipes/positive.pipe'

describe('SharedNetworksPageComponent', () => {
    let component: SharedNetworksPageComponent
    let fixture: ComponentFixture<SharedNetworksPageComponent>
    let dhcpService: DHCPService
    let route: ActivatedRoute
    let router: Router
    let routerEventSubject: BehaviorSubject<NavigationEnd>

    beforeEach(waitForAsync(() => {
        TestBed.configureTestingModule({
            imports: [
                BreadcrumbModule,
                ButtonModule,
                ChartModule,
                CheckboxModule,
                ChipsModule,
                ConfirmDialogModule,
                DividerModule,
                DropdownModule,
                FieldsetModule,
                FormsModule,
                HttpClientTestingModule,
                InputNumberModule,
                MultiSelectModule,
                NoopAnimationsModule,
                OverlayPanelModule,
                ProgressSpinnerModule,
                ReactiveFormsModule,
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
                TableModule,
                TabMenuModule,
                TooltipModule,
                PanelModule,
                TagModule,
            ],
            declarations: [
                AddressPoolBarComponent,
                ArrayValueSetFormComponent,
                BreadcrumbsComponent,
                DhcpClientClassSetFormComponent,
                DhcpOptionFormComponent,
                DhcpOptionSetFormComponent,
                EntityLinkComponent,
                HelpTipComponent,
                HumanCountComponent,
                HumanCountPipe,
                LocalNumberPipe,
                DelegatedPrefixBarComponent,
                PlaceholderPipe,
                PositivePipe,
                SharedNetworkFormComponent,
                SharedNetworksPageComponent,
                SharedNetworkTabComponent,
                SharedParametersFormComponent,
                SubnetBarComponent,
                UtilizationStatsChartComponent,
                UtilizationStatsChartsComponent,
                SharedNetworksTableComponent,
                PluralizePipe,
            ],
            providers: [ConfirmationService, MessageService],
        })

        dhcpService = TestBed.inject(DHCPService)
        fixture = TestBed.createComponent(SharedNetworksPageComponent)
        component = fixture.componentInstance
        route = fixture.debugElement.injector.get(ActivatedRoute)
        route.snapshot = {
            paramMap: convertToParamMap({}),
            queryParamMap: convertToParamMap({}),
        } as ActivatedRouteSnapshot
        router = fixture.debugElement.injector.get(Router)
        routerEventSubject = new BehaviorSubject(
            new NavigationEnd(1, 'dhcp/shared-networks', 'dhcp/shared-networks/all')
        )

        spyOnProperty(router, 'events').and.returnValue(routerEventSubject)

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
        const getNetworksSpy = spyOn(dhcpService, 'getSharedNetworks')
        // Prepare response when no filtering is applied.
        getNetworksSpy.withArgs(0, 10, null, null, null).and.returnValue(of(fakeResponses[0]))
        // Prepare response when shared networks are filtered by text to get an item without stats.
        getNetworksSpy.withArgs(0, 10, null, null, 'frog-no-stats').and.returnValue(of(fakeResponses[1]))
        // Prepare response when shared networks are filtered by text to get an item with 4 subnets.
        getNetworksSpy.withArgs(0, 10, null, null, 'cat').and.returnValue(of(fakeResponses[2]))

        spyOn(dhcpService, 'getSharedNetwork').and.returnValues(
            of(fakeResponses[0].items[0] as HttpEvent<SharedNetwork>)
        )

        fixture = TestBed.createComponent(SharedNetworksPageComponent)
        component = fixture.componentInstance
        fixture.detectChanges()

        // PrimeNG TabMenu is using setTimeout() logic when scrollable property is set to true.
        // This makes testing in fakeAsync zone unexpected, so disable 'scrollable' feature in tests.
        const m = fixture.debugElement.query(By.directive(TabMenu))
        if (m?.context) {
            m.context.scrollable = false
        }

        // PrimeNG table is stateful in the component, so clear stored filter between tests.
        component.table.table.clearFilterValues()
        component.table.filter$.next({})

        fixture.detectChanges()
    }))

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should have breadcrumbs', () => {
        const breadcrumbsElement = fixture.debugElement.query(By.directive(BreadcrumbsComponent))
        expect(breadcrumbsElement).not.toBeNull()
        const breadcrumbsComponent = breadcrumbsElement.componentInstance as BreadcrumbsComponent
        expect(breadcrumbsComponent).not.toBeNull()
        expect(breadcrumbsComponent.items).toHaveSize(2)
        expect(breadcrumbsComponent.items[0].label).toEqual('DHCP')
        expect(breadcrumbsComponent.items[1].label).toEqual('Shared Networks')
    })

    it('should open and close tabs', fakeAsync(() => {
        component.openTabBySharedNetworkId(1)
        tick()
        fixture.detectChanges()

        expect(component.openedTabs.length).toBe(2)
        expect(component.activeTabIndex).toBe(1)

        component.closeTabByIndex(1)

        expect(component.openedTabs.length).toBe(1)
        expect(component.activeTabIndex).toBe(0)

        component.closeTabByIndex(0)

        expect(component.openedTabs.length).toBe(1)
        expect(component.activeTabIndex).toBe(0)
    }))

    it('should cancel transaction for new shared network when cancel button is clicked', fakeAsync(() => {
        tick()
        fixture.detectChanges()

        const createSharedNetworkBeginResp: any = {
            id: 123,
            daemons: [
                {
                    id: 1,
                    name: 'dhcp4',
                    app: {
                        name: 'first',
                    },
                },
            ],
            sharedNetworks4: [],
            sharedNetworks6: [],
            clientClasses: [],
        }

        const okResp: any = {
            status: 200,
        }

        spyOn(dhcpService, 'createSharedNetworkBegin').and.returnValue(of(createSharedNetworkBeginResp))
        spyOn(dhcpService, 'createSharedNetworkDelete').and.returnValue(of(okResp))

        component.openNewSharedNetworkTab()
        fixture.detectChanges()
        tick()

        expect(component.openedTabs.length).toBe(2)

        expect(dhcpService.createSharedNetworkBegin).toHaveBeenCalled()

        expect(component.openedTabs.length).toBe(2)
        expect(component.openedTabs[1].state.transactionId).toBe(123)

        // Cancel editing. It should close the form and the transaction should be deleted.
        component.onSharedNetworkFormCancel()
        fixture.detectChanges()
        tick()

        expect(component.tabs.length).toBe(1)
        expect(component.openedTabs.length).toBe(1)
        expect(component.activeTabIndex).toBe(0)
        expect(component.openedTabs[0].tabType).toBe(TabType.List)

        expect(dhcpService.createSharedNetworkDelete).toHaveBeenCalled()
    }))

    it('should cancel transaction for shared network update when cancel button is clicked', fakeAsync(() => {
        tick()
        fixture.detectChanges()

        const updateSharedNetworkBeginResp: any = {
            id: 123,
            sharedNetwork: {
                id: 1,
                name: 'stanza',
                universe: 4,
                localSharedNetworks: [
                    {
                        appId: 234,
                        daemonId: 1,
                        appName: 'server 1',
                        keaConfigSharedNetworkParameters: {
                            sharedNetworkLevelParameters: {
                                allocator: 'random',
                                options: [
                                    {
                                        alwaysSend: true,
                                        code: 5,
                                        encapsulate: '',
                                        fields: [
                                            {
                                                fieldType: 'ipv4-address',
                                                values: ['192.0.2.1'],
                                            },
                                        ],
                                        options: [],
                                        universe: 4,
                                    },
                                ],
                                optionsHash: '123',
                            },
                        },
                    },
                ],
                subnets: [
                    {
                        id: 123,
                        subnet: '192.0.2.0/24',
                        sharedNetwork: 'floor3',
                        sharedNetworkId: 3,
                        localSubnets: [
                            {
                                id: 123,
                                appId: 234,
                                daemonId: 1,
                                appName: 'server 1',
                            },
                        ],
                    },
                ],
            },
            daemons: [
                {
                    id: 1,
                    name: 'dhcp4',
                    app: {
                        name: 'first',
                    },
                },
            ],
            sharedNetworks4: [],
            sharedNetworks6: [],
            clientClasses: [],
        }

        const okResp: any = {
            status: 200,
        }

        spyOn(dhcpService, 'updateSharedNetworkBegin').and.returnValue(of(updateSharedNetworkBeginResp))
        spyOn(dhcpService, 'updateSharedNetworkDelete').and.returnValue(of(okResp))

        component.openTabBySharedNetworkId(1)
        tick()
        fixture.detectChanges()

        expect(component.openedTabs.length).toBe(2)

        component.onSharedNetworkEditBegin({ id: 1 })
        fixture.detectChanges()
        tick()

        expect(dhcpService.updateSharedNetworkBegin).toHaveBeenCalled()

        expect(component.openedTabs.length).toBe(2)
        expect(component.openedTabs[1].state.transactionId).toBe(123)

        // Cancel editing. It should close the form and the transaction should be deleted.
        component.onSharedNetworkFormCancel(1)
        fixture.detectChanges()
        tick()

        expect(component.tabs.length).toBe(2)
        expect(component.openedTabs.length).toBe(2)
        expect(component.activeTabIndex).toBe(1)
        expect(component.openedTabs[1].tabType).toBe(TabType.Display)

        expect(dhcpService.updateSharedNetworkDelete).toHaveBeenCalled()
    }))

    it('should close subnet tab when subnet is deleted', fakeAsync(() => {
        tick()
        fixture.detectChanges()

        // Open subnet tab.
        component.openTabBySharedNetworkId(1)
        fixture.detectChanges()
        tick()
        expect(component.openedTabs.length).toBe(2)

        // Simulate the notification that the shared network has been deleted.
        component.onSharedNetworkDelete({
            id: 1,
        })
        fixture.detectChanges()
        tick()

        // The main shared network tab should only be left.
        expect(component.openedTabs.length).toBe(1)
    }))
})
