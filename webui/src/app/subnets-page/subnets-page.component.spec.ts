import { By } from '@angular/platform-browser'
import { ComponentFixture, TestBed, fakeAsync, tick, waitForAsync, flush } from '@angular/core/testing'

import { SubnetsPageComponent } from './subnets-page.component'
import { FormsModule, ReactiveFormsModule } from '@angular/forms'
import { DropdownModule } from 'primeng/dropdown'
import { TableModule } from 'primeng/table'
import { SubnetBarComponent } from '../subnet-bar/subnet-bar.component'
import { TooltipModule } from 'primeng/tooltip'
import { ActivatedRoute, Router, convertToParamMap, NavigationEnd, ActivatedRouteSnapshot } from '@angular/router'
import { DHCPService, SettingsService, Subnet, UsersService } from '../backend'
import { HttpClientTestingModule } from '@angular/common/http/testing'
import { BehaviorSubject, of, throwError } from 'rxjs'
import { ConfirmationService, MessageService } from 'primeng/api'
import { BreadcrumbsComponent } from '../breadcrumbs/breadcrumbs.component'
import { HelpTipComponent } from '../help-tip/help-tip.component'
import { BreadcrumbModule } from 'primeng/breadcrumb'
import { OverlayPanelModule } from 'primeng/overlaypanel'
import { NoopAnimationsModule } from '@angular/platform-browser/animations'
import { DelegatedPrefixBarComponent } from '../delegated-prefix-bar/delegated-prefix-bar.component'
import { HumanCountComponent } from '../human-count/human-count.component'
import { LocalNumberPipe } from '../pipes/local-number.pipe'
import { RouterTestingModule } from '@angular/router/testing'
import { MessageModule } from 'primeng/message'
import { HumanCountPipe } from '../pipes/human-count.pipe'
import { TabMenu, TabMenuModule } from 'primeng/tabmenu'
import { EntityLinkComponent } from '../entity-link/entity-link.component'
import { AddressPoolBarComponent } from '../address-pool-bar/address-pool-bar.component'
import { SubnetTabComponent } from '../subnet-tab/subnet-tab.component'
import { FieldsetModule } from 'primeng/fieldset'
import { CascadedParametersBoardComponent } from '../cascaded-parameters-board/cascaded-parameters-board.component'
import { DhcpOptionSetViewComponent } from '../dhcp-option-set-view/dhcp-option-set-view.component'
import { TreeModule } from 'primeng/tree'
import { SubnetFormComponent } from '../subnet-form/subnet-form.component'
import { ProgressSpinnerModule } from 'primeng/progressspinner'
import { CheckboxModule } from 'primeng/checkbox'
import { ButtonModule } from 'primeng/button'
import { ChipsModule } from 'primeng/chips'
import { DividerModule } from 'primeng/divider'
import { InputNumberModule } from 'primeng/inputnumber'
import { MessagesModule } from 'primeng/messages'
import { MultiSelectModule } from 'primeng/multiselect'
import { TagModule } from 'primeng/tag'
import { TriStateCheckboxModule } from 'primeng/tristatecheckbox'
import { SplitButtonModule } from 'primeng/splitbutton'
import { ToastModule } from 'primeng/toast'
import { DhcpClientClassSetFormComponent } from '../dhcp-client-class-set-form/dhcp-client-class-set-form.component'
import { DhcpOptionFormComponent } from '../dhcp-option-form/dhcp-option-form.component'
import { DhcpOptionSetFormComponent } from '../dhcp-option-set-form/dhcp-option-set-form.component'
import { SharedParametersFormComponent } from '../shared-parameters-form/shared-parameters-form.component'
import { AccordionModule } from 'primeng/accordion'
import { AddressPoolFormComponent } from '../address-pool-form/address-pool-form.component'
import { ConfirmDialogModule } from 'primeng/confirmdialog'
import { HttpErrorResponse, HttpEvent } from '@angular/common/http'
import { TabType } from '../tab'
import { SubnetsTableComponent } from '../subnets-table/subnets-table.component'
import { PanelModule } from 'primeng/panel'
import { PluralizePipe } from '../pipes/pluralize.pipe'
import { PlaceholderPipe } from '../pipes/placeholder.pipe'
import { ArrayValueSetFormComponent } from '../array-value-set-form/array-value-set-form.component'
import { ParameterViewComponent } from '../parameter-view/parameter-view.component'
import { SettingService } from '../setting.service'

describe('SubnetsPageComponent', () => {
    let component: SubnetsPageComponent
    let fixture: ComponentFixture<SubnetsPageComponent>
    let dhcpService: DHCPService
    let messageService: MessageService
    let settingService: SettingService
    let route: ActivatedRoute
    let router: Router
    let routerEventSubject: BehaviorSubject<NavigationEnd>

    beforeEach(waitForAsync(() => {
        TestBed.configureTestingModule({
            providers: [ConfirmationService, DHCPService, UsersService, MessageService, SettingsService],
            imports: [
                AccordionModule,
                FormsModule,
                DropdownModule,
                TableModule,
                TooltipModule,
                RouterTestingModule.withRoutes([
                    {
                        path: 'dhcp/subnets',
                        pathMatch: 'full',
                        redirectTo: 'dhcp/subnets/all',
                    },
                    {
                        path: 'dhcp/subnets/:id',
                        component: SubnetsPageComponent,
                    },
                ]),
                HttpClientTestingModule,
                BreadcrumbModule,
                OverlayPanelModule,
                NoopAnimationsModule,
                MessageModule,
                TabMenuModule,
                FieldsetModule,
                TreeModule,
                ProgressSpinnerModule,
                ButtonModule,
                CheckboxModule,
                ChipsModule,
                DividerModule,
                InputNumberModule,
                MessagesModule,
                MultiSelectModule,
                TagModule,
                TriStateCheckboxModule,
                ReactiveFormsModule,
                SplitButtonModule,
                ToastModule,
                ConfirmDialogModule,
                PanelModule,
            ],
            declarations: [
                AddressPoolFormComponent,
                SubnetsPageComponent,
                SubnetBarComponent,
                BreadcrumbsComponent,
                HelpTipComponent,
                DelegatedPrefixBarComponent,
                HumanCountComponent,
                HumanCountPipe,
                LocalNumberPipe,
                EntityLinkComponent,
                AddressPoolBarComponent,
                SubnetTabComponent,
                CascadedParametersBoardComponent,
                DhcpOptionSetViewComponent,
                SubnetFormComponent,
                DhcpClientClassSetFormComponent,
                DhcpOptionFormComponent,
                DhcpOptionSetFormComponent,
                SharedParametersFormComponent,
                SubnetsTableComponent,
                PluralizePipe,
                PlaceholderPipe,
                ArrayValueSetFormComponent,
                ParameterViewComponent,
            ],
        })
        dhcpService = TestBed.inject(DHCPService)
        messageService = TestBed.inject(MessageService)
        settingService = TestBed.inject(SettingService)
        fixture = TestBed.createComponent(SubnetsPageComponent)
        component = fixture.componentInstance
        route = fixture.debugElement.injector.get(ActivatedRoute)
        route.snapshot = {
            paramMap: convertToParamMap({}),
            queryParamMap: convertToParamMap({}),
        } as ActivatedRouteSnapshot
        router = fixture.debugElement.injector.get(Router)
        routerEventSubject = new BehaviorSubject(new NavigationEnd(1, 'dhcp/subnets', 'dhcp/subnets/all'))
        spyOnProperty(router, 'events').and.returnValue(routerEventSubject)

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
                                pools: [
                                    {
                                        pool: '1.0.0.4-1.0.255.254',
                                    },
                                ],
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
                                pools: [
                                    {
                                        pool: '1.0.0.4-1.0.255.254',
                                    },
                                ],
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
                                pools: [
                                    {
                                        pool: '1.1.0.4-1.1.255.254',
                                    },
                                ],
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
                                pools: [
                                    {
                                        pool: '1.0.0.4-1.0.255.254',
                                    },
                                ],
                            },
                        ],
                        statsCollectedAt: '1970-01-01T12:00:00.0Z',
                        subnet: '1.0.0.0/16',
                    },
                ],
                total: 10496,
            },
        ]

        const getSubnetsSpy = spyOn(dhcpService, 'getSubnets')
        // Prepare response when no filtering is applied.
        getSubnetsSpy.withArgs(0, 10, null, null, null, null).and.returnValue(of(fakeResponses[0]))
        // Prepare response when subnets are filtered by text.
        getSubnetsSpy.withArgs(0, 10, null, null, null, '1.0.0.0/16').and.returnValue(of(fakeResponses[1]))
        // Prepare response when subnets are filtered by subnet Id.
        getSubnetsSpy.withArgs(0, 10, null, 5, null, null).and.returnValue(of(fakeResponses[1]))

        const updateSubnetBeginResp: any = {
            id: 123,
            subnet: {
                id: 5,
                subnet: '192.0.2.0/24',
                localSubnets: [
                    {
                        id: 123,
                        daemonId: 1,
                        appName: 'server 1',
                        pools: [],
                        keaConfigSubnetParameters: {
                            subnetLevelParameters: {
                                allocator: 'random',
                                options: [],
                                optionsHash: '',
                            },
                        },
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
        }

        const updateSubnetBeginSpy = spyOn(dhcpService, 'updateSubnetBegin')
        // Prepare response when updateSubnetBegin is called for subnet id 5.
        updateSubnetBeginSpy.withArgs(5).and.returnValue(of(updateSubnetBeginResp))

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

        spyOn(settingService, 'getSettings').and.returnValue(
            of({
                grafanaUrl: 'http://localhost:3000',
            } as any)
        )
    }))

    /**
     * Triggers the component handler called when the route changes.
     * @param params The parameters to pass to the route.
     */
    function navigate(params: { id?: number | string }) {
        route.snapshot = {
            paramMap: convertToParamMap(params),
            queryParamMap: convertToParamMap({}),
        } as ActivatedRouteSnapshot

        const eid = routerEventSubject.getValue().id + 1
        routerEventSubject.next(new NavigationEnd(eid, `dhcp/subnets/${params.id}`, `dhcp/subnets/${params.id}`))

        flush()
        fixture.detectChanges()
    }

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should fetch grafana url', async () => {
        component.ngOnInit()
        await fixture.whenStable()
        expect(component.grafanaUrl).toBe('http://localhost:3000')
    })

    it('should convert statistics to big integers', async () => {
        // Act
        await fixture.whenStable()

        // Assert
        expect(component.table.dataCollection).toBeTruthy()
        expect(component.table.dataCollection.length).toBeGreaterThan(0)
        const stats: { [key: string]: BigInt } = component.table.dataCollection[0].stats as any
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
        // Filter by text to get subnet without stats.
        component.table.filter$.next({ text: '1.0.0.0/16' })
        // Act
        fixture.detectChanges()
        await fixture.whenStable()

        // Assert
        expect(component.table.dataCollection[0].stats).toBeUndefined()
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

        component.table.dataCollection = subnets
        expect(component.table.isAnyIPv6SubnetVisible).toBeFalse()

        subnets.push({ subnet: 'fe80::/64' })
        expect(component.table.isAnyIPv6SubnetVisible).toBeTrue()
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
        // Act
        await fixture.whenStable()

        component.table.filter$.next({ subnetId: 5 })

        fixture.detectChanges()
        await fixture.whenStable()

        // Assert
        expect(dhcpService.getSubnets).toHaveBeenCalledWith(0, 10, null, 5, null, null)
        // One subnet record is expected after filtering.
        expect(component.table.dataCollection).toBeTruthy()
        expect(component.table.dataCollection.length).toBe(1)
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
        expect(component.table.hasAssignedMultipleKeaSubnetIds(subnet)).toBeFalse()
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
        expect(component.table.hasAssignedMultipleKeaSubnetIds(subnet)).toBeTrue()
    })

    it('should recognize that all local subnets have the same IDs if the local subnets list is empty', () => {
        // Arrange
        const subnet: Subnet = {
            subnet: 'fe80::/64',
            localSubnets: [],
        }

        // Act & Assert
        expect(component.table.hasAssignedMultipleKeaSubnetIds(subnet)).toBeFalse()
    })

    it('should close new subnet form when form is submitted', fakeAsync(() => {
        tick()
        fixture.detectChanges()

        const createSubnetBeginResp: any = {
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
        }

        const getSubnetResp: any = {
            id: 5,
            subnet: '192.0.2.0/24',
            localSubnets: [
                {
                    id: 123,
                    daemonId: 1,
                    appName: 'server 1',
                    pools: [],
                    keaConfigSubnetParameters: {
                        subnetLevelParameters: {
                            allocator: 'random',
                            options: [],
                            optionsHash: '',
                        },
                    },
                },
            ],
        }

        const okResp: any = {
            status: 200,
        }

        spyOn(dhcpService, 'createSubnetBegin').and.returnValue(of(createSubnetBeginResp))
        spyOn(dhcpService, 'createSubnetDelete').and.returnValue(of(okResp))
        spyOn(dhcpService, 'getSubnet').and.returnValue(of(getSubnetResp))

        navigate({ id: 'new' })

        tick()
        fixture.detectChanges()

        expect(component.openedTabs.length).toBe(2)

        expect(dhcpService.createSubnetBegin).toHaveBeenCalled()

        expect(component.openedTabs.length).toBe(2)
        expect(component.openedTabs[1].state.hasOwnProperty('transactionId')).toBeTrue()
        expect(component.openedTabs[1].state.transactionId).toBe(123)

        component.onSubnetFormSubmit(component.openedTabs[1].state)

        tick()
        fixture.detectChanges()

        expect(dhcpService.getSubnet).toHaveBeenCalled()
        expect(component.tabs.length).toBe(2)
        expect(component.openedTabs.length).toBe(2)
        expect(component.activeTabIndex).toBe(1)
        expect(component.openedTabs[1].tabType).toBe(TabType.Display)

        expect(dhcpService.createSubnetDelete).not.toHaveBeenCalled()
    }))

    it('should close subnet update form when form is submitted', fakeAsync(() => {
        tick()
        fixture.detectChanges()

        const subnet: any = {
            id: 5,
            subnet: '192.0.2.0/24',
            localSubnets: [
                {
                    id: 123,
                    daemonId: 1,
                    appName: 'server 1',
                    pools: [],
                    keaConfigSubnetParameters: {
                        subnetLevelParameters: {
                            allocator: 'random',
                            options: [],
                            optionsHash: '',
                        },
                    },
                },
            ],
        }

        const okResp: any = {
            status: 200,
        }

        spyOn(dhcpService, 'updateSubnetDelete').and.returnValue(of(okResp))
        spyOn(dhcpService, 'getSubnet').and.returnValue(of(subnet))

        navigate({ id: 5 })

        tick()
        fixture.detectChanges()

        expect(component.openedTabs.length).toBe(2)

        component.onSubnetEditBegin({ id: 5 })

        tick()
        fixture.detectChanges()
        tick()
        fixture.detectChanges()

        expect(dhcpService.updateSubnetBegin).toHaveBeenCalled()

        expect(component.openedTabs.length).toBe(2)
        expect(component.openedTabs[1].state.hasOwnProperty('transactionId')).toBeTrue()
        expect(component.openedTabs[1].state.transactionId).toBe(123)

        component.onSubnetFormSubmit(component.openedTabs[1].state)

        tick()
        fixture.detectChanges()

        expect(dhcpService.getSubnet).toHaveBeenCalled()
        expect(component.tabs.length).toBe(2)
        expect(component.openedTabs.length).toBe(2)
        expect(component.activeTabIndex).toBe(1)
        expect(component.openedTabs[1].tabType).toBe(TabType.Display)

        expect(dhcpService.updateSubnetDelete).not.toHaveBeenCalled()
    }))

    it('should keep the tab open when getting a subnet after submission fails', fakeAsync(() => {
        const okResp: any = {
            status: 200,
        }

        spyOn(dhcpService, 'updateSubnetDelete').and.returnValue(of(okResp))
        spyOn(dhcpService, 'getSubnet').and.returnValues(
            of({ id: 5 }) as any,
            throwError(() => new HttpErrorResponse({ status: 404 }))
        )

        navigate({ id: 5 })

        tick()
        fixture.detectChanges()

        expect(component.openedTabs.length).toBe(2)

        component.onSubnetEditBegin({ id: 5 })

        tick()
        fixture.detectChanges()
        tick()
        fixture.detectChanges()

        expect(dhcpService.updateSubnetBegin).toHaveBeenCalled()

        expect(component.openedTabs.length).toBe(2)
        expect(component.openedTabs[1].state.hasOwnProperty('transactionId')).toBeTrue()
        expect(component.openedTabs[1].state.transactionId).toBe(123)

        component.onSubnetFormSubmit(component.openedTabs[1].state)

        tick()
        fixture.detectChanges()

        expect(dhcpService.getSubnet).toHaveBeenCalled()
        expect(component.tabs.length).toBe(2)
        expect(component.openedTabs.length).toBe(2)
        expect(component.activeTabIndex).toBe(1)
        expect(component.openedTabs[1].tabType).toBe(TabType.Display)

        expect(dhcpService.updateSubnetDelete).not.toHaveBeenCalled()
    }))

    it('should cancel a new subnet transaction when a tab is closed', fakeAsync(() => {
        tick()
        fixture.detectChanges()

        const createSubnetBeginResp: any = {
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
        }

        const okResp: any = {
            status: 200,
        }

        spyOn(dhcpService, 'createSubnetBegin').and.returnValue(of(createSubnetBeginResp))
        spyOn(dhcpService, 'createSubnetDelete').and.returnValue(of(okResp))

        navigate({ id: 'new' })

        tick()
        fixture.detectChanges()

        expect(component.openedTabs.length).toBe(2)

        expect(dhcpService.createSubnetBegin).toHaveBeenCalled()

        expect(component.openedTabs.length).toBe(2)
        expect(component.openedTabs[1].state.hasOwnProperty('transactionId')).toBeTrue()
        expect(component.openedTabs[1].state.transactionId).toBe(123)

        component.closeTabByIndex(1)

        tick()
        fixture.detectChanges()

        expect(component.tabs.length).toBe(1)
        expect(component.activeTabIndex).toBe(0)

        expect(dhcpService.createSubnetDelete).toHaveBeenCalled()
    }))

    it('should cancel an update transaction when a tab is closed', fakeAsync(() => {
        tick()
        fixture.detectChanges()

        const okResp: any = {
            status: 200,
        }

        spyOn(dhcpService, 'updateSubnetDelete').and.returnValue(of(okResp))
        spyOn(dhcpService, 'getSubnet').and.returnValue(of({ id: 5 }) as any)

        navigate({ id: 5 })

        tick()
        fixture.detectChanges()

        expect(component.openedTabs.length).toBe(2)

        component.onSubnetEditBegin({ id: 5 })

        tick()
        fixture.detectChanges()
        tick()
        fixture.detectChanges()

        expect(dhcpService.updateSubnetBegin).toHaveBeenCalled()

        expect(component.openedTabs.length).toBe(2)
        expect(component.openedTabs[1].state.hasOwnProperty('transactionId')).toBeTrue()
        expect(component.openedTabs[1].state.transactionId).toBe(123)

        component.closeTabByIndex(1)

        tick()
        fixture.detectChanges()

        expect(component.tabs.length).toBe(1)
        expect(component.activeTabIndex).toBe(0)

        expect(dhcpService.updateSubnetDelete).toHaveBeenCalled()
    }))

    it('should cancel a new subnet transaction when cancel button is clicked', fakeAsync(() => {
        tick()
        fixture.detectChanges()

        const createSubnetBeginResp: any = {
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
        }

        const okResp: any = {
            status: 200,
        }

        spyOn(dhcpService, 'createSubnetBegin').and.returnValue(of(createSubnetBeginResp))
        spyOn(dhcpService, 'createSubnetDelete').and.returnValue(of(okResp))

        navigate({ id: 'new' })

        tick()
        fixture.detectChanges()

        expect(component.openedTabs.length).toBe(2)

        expect(dhcpService.createSubnetBegin).toHaveBeenCalled()

        expect(component.openedTabs.length).toBe(2)
        expect(component.openedTabs[1].state.hasOwnProperty('transactionId')).toBeTrue()
        expect(component.openedTabs[1].state.transactionId).toBe(123)

        component.onSubnetFormCancel()

        tick()
        fixture.detectChanges()

        expect(component.tabs.length).toBe(1)
        expect(component.openedTabs.length).toBe(1)
        expect(component.activeTabIndex).toBe(0)
        expect(component.openedTabs[0].tabType).toBe(TabType.List)

        expect(dhcpService.createSubnetDelete).toHaveBeenCalled()
    }))

    it('should cancel transaction when cancel button is clicked', fakeAsync(() => {
        tick()
        fixture.detectChanges()

        const okResp: any = {
            status: 200,
        }

        spyOn(dhcpService, 'updateSubnetDelete').and.returnValue(of(okResp))
        spyOn(dhcpService, 'getSubnet').and.returnValue(of({ id: 5 }) as any)

        navigate({ id: 5 })

        tick()
        fixture.detectChanges()

        expect(component.openedTabs.length).toBe(2)

        component.onSubnetEditBegin({ id: 5 })

        tick()
        fixture.detectChanges()
        tick()
        fixture.detectChanges()

        expect(dhcpService.updateSubnetBegin).toHaveBeenCalled()

        expect(component.openedTabs.length).toBe(2)
        expect(component.openedTabs[1].state.hasOwnProperty('transactionId')).toBeTrue()
        expect(component.openedTabs[1].state.transactionId).toBe(123)

        // Cancel editing. It should close the form and the transaction should be deleted.
        component.onSubnetFormCancel(5)

        tick()
        fixture.detectChanges()

        expect(component.tabs.length).toBe(2)
        expect(component.openedTabs.length).toBe(2)
        expect(component.activeTabIndex).toBe(1)
        expect(component.openedTabs[1].tabType).toBe(TabType.Display)

        expect(dhcpService.updateSubnetDelete).toHaveBeenCalled()
    }))

    it('should show error message when transaction canceling fails', fakeAsync(() => {
        tick()
        fixture.detectChanges()

        const createSubnetBeginResp: any = {
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
        }

        spyOn(dhcpService, 'createSubnetBegin').and.returnValue(of(createSubnetBeginResp))
        spyOn(dhcpService, 'createSubnetDelete').and.returnValue(
            throwError(() => new HttpErrorResponse({ status: 404 }))
        )
        spyOn(messageService, 'add')

        navigate({ id: 'new' })

        tick()
        fixture.detectChanges()

        expect(component.openedTabs.length).toBe(2)

        expect(dhcpService.createSubnetBegin).toHaveBeenCalled()

        component.onSubnetFormCancel()

        tick()
        fixture.detectChanges()

        expect(dhcpService.createSubnetDelete).toHaveBeenCalled()
        expect(messageService.add).toHaveBeenCalled()
    }))

    it('should show error message when transaction canceling fails second', fakeAsync(() => {
        tick()
        fixture.detectChanges()

        spyOn(dhcpService, 'updateSubnetDelete').and.returnValue(
            throwError(() => new HttpErrorResponse({ status: 404 }))
        )
        spyOn(dhcpService, 'getSubnet').and.returnValue(of({ id: 5 }) as any)
        spyOn(messageService, 'add')

        navigate({ id: 5 })

        tick()
        fixture.detectChanges()

        expect(component.openedTabs.length).toBe(2)

        component.onSubnetEditBegin({ id: 5 })

        tick()
        fixture.detectChanges()
        tick()
        fixture.detectChanges()

        expect(dhcpService.updateSubnetBegin).toHaveBeenCalled()

        component.onSubnetFormCancel(5)

        tick()
        fixture.detectChanges()

        expect(dhcpService.updateSubnetDelete).toHaveBeenCalled()
        expect(messageService.add).toHaveBeenCalled()
    }))

    it('should close subnet tab when subnet is deleted', fakeAsync(() => {
        tick()
        fixture.detectChanges()

        const subnet: Subnet & HttpEvent<Subnet> = {
            id: 5,
            subnet: '192.0.2.0/24',
            localSubnets: [
                {
                    id: 123,
                    daemonId: 1,
                    appName: 'server 1',
                    pools: [],
                    keaConfigSubnetParameters: {
                        subnetLevelParameters: {
                            allocator: 'random',
                            options: [],
                            optionsHash: '',
                        },
                    },
                },
            ],
            type: undefined,
        }

        spyOn(dhcpService, 'getSubnet').and.returnValue(of(subnet))

        // Open subnet tab.
        navigate({ id: 5 })

        tick()
        fixture.detectChanges()

        expect(component.openedTabs.length).toBe(2)

        // Simulate the notification that the subnet has been deleted.
        component.onSubnetDelete(subnet)

        tick()
        fixture.detectChanges()

        // The main subnet tab should only be left.
        expect(component.openedTabs.length).toBe(1)
    }))
})
