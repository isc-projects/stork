import { By } from '@angular/platform-browser'
import { ComponentFixture, TestBed, fakeAsync, tick, waitForAsync } from '@angular/core/testing'

import { SubnetsPageComponent } from './subnets-page.component'
import { FormsModule, ReactiveFormsModule } from '@angular/forms'
import { DropdownModule } from 'primeng/dropdown'
import { TableModule } from 'primeng/table'
import { SubnetBarComponent } from '../subnet-bar/subnet-bar.component'
import { TooltipModule } from 'primeng/tooltip'
import { ActivatedRoute, Router, convertToParamMap } from '@angular/router'
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
import { TabMenuModule } from 'primeng/tabmenu'
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
import { HttpEvent } from '@angular/common/http'
import { TabType } from '../tab'

describe('SubnetsPageComponent', () => {
    let component: SubnetsPageComponent
    let fixture: ComponentFixture<SubnetsPageComponent>
    let dhcpService: DHCPService
    let messageService: MessageService
    let router: Router
    let paramMap: any
    let paramMapSubject: BehaviorSubject<any>

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
            ],
        })
        dhcpService = TestBed.inject(DHCPService)
        messageService = TestBed.inject(MessageService)
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
        spyOn(dhcpService, 'getSubnets').and.returnValues(
            // The subnets are fetched twice before the unit test starts.
            // Some tests call the getSubnets more than once.
            of(fakeResponses[0]),
            of(fakeResponses[0]),
            of(fakeResponses[1]),
            of(fakeResponses[1])
        )

        fixture = TestBed.createComponent(SubnetsPageComponent)
        component = fixture.componentInstance
        const route = fixture.debugElement.injector.get(ActivatedRoute)
        paramMap = convertToParamMap({})
        paramMapSubject = new BehaviorSubject(paramMap)
        spyOnProperty(route, 'paramMap').and.returnValue(paramMapSubject)
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
                    subnetId: 1,
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

    it('should close new subnet form when form is submitted', fakeAsync(() => {
        component.loadSubnets({})
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

        paramMapSubject.next(convertToParamMap({ id: 'new' }))
        fixture.detectChanges()
        tick()

        expect(component.openedTabs.length).toBe(2)

        fixture.detectChanges()
        tick()

        expect(dhcpService.createSubnetBegin).toHaveBeenCalled()

        expect(component.openedTabs.length).toBe(2)
        expect(component.openedTabs[1].state.hasOwnProperty('transactionId')).toBeTrue()
        expect(component.openedTabs[1].state.transactionId).toBe(123)

        component.onSubnetFormSubmit(component.openedTabs[1].state)
        tick()

        expect(dhcpService.getSubnet).toHaveBeenCalled()
        expect(component.tabs.length).toBe(2)
        expect(component.openedTabs.length).toBe(2)
        expect(component.activeTabIndex).toBe(1)
        expect(component.openedTabs[1].tabType).toBe(TabType.Display)

        expect(dhcpService.createSubnetDelete).not.toHaveBeenCalled()
    }))

    it('should close subnet update form when form is submitted', fakeAsync(() => {
        component.loadSubnets({})
        tick()
        fixture.detectChanges()

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

        const okResp: any = {
            status: 200,
        }

        spyOn(dhcpService, 'updateSubnetBegin').and.returnValue(of(updateSubnetBeginResp))
        spyOn(dhcpService, 'updateSubnetDelete').and.returnValue(of(okResp))
        spyOn(dhcpService, 'getSubnet').and.returnValue(of(updateSubnetBeginResp.subnet))

        paramMapSubject.next(convertToParamMap({ id: 5 }))
        fixture.detectChanges()
        tick()

        expect(component.openedTabs.length).toBe(2)

        component.onSubnetEditBegin({ id: 5 })
        fixture.detectChanges()
        tick()

        expect(dhcpService.updateSubnetBegin).toHaveBeenCalled()

        expect(component.openedTabs.length).toBe(2)
        expect(component.openedTabs[1].state.hasOwnProperty('transactionId')).toBeTrue()
        expect(component.openedTabs[1].state.transactionId).toBe(123)

        component.onSubnetFormSubmit(component.openedTabs[1].state)
        tick()

        expect(dhcpService.getSubnet).toHaveBeenCalled()
        expect(component.tabs.length).toBe(2)
        expect(component.openedTabs.length).toBe(2)
        expect(component.activeTabIndex).toBe(1)
        expect(component.openedTabs[1].tabType).toBe(TabType.Display)

        expect(dhcpService.updateSubnetDelete).not.toHaveBeenCalled()
    }))

    it('should keep the tab open when getting a subnet after submission fails', fakeAsync(() => {
        component.loadSubnets({})
        tick()
        fixture.detectChanges()

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

        const okResp: any = {
            status: 200,
        }

        spyOn(dhcpService, 'updateSubnetBegin').and.returnValue(of(updateSubnetBeginResp))
        spyOn(dhcpService, 'updateSubnetDelete').and.returnValue(of(okResp))
        spyOn(dhcpService, 'getSubnet').and.returnValues(of({ id: 5 }) as any, throwError({ status: 404 }))

        paramMapSubject.next(convertToParamMap({ id: 5 }))
        fixture.detectChanges()
        tick()

        expect(component.openedTabs.length).toBe(2)

        component.onSubnetEditBegin({ id: 5 })
        fixture.detectChanges()
        tick()

        expect(dhcpService.updateSubnetBegin).toHaveBeenCalled()

        expect(component.openedTabs.length).toBe(2)
        expect(component.openedTabs[1].state.hasOwnProperty('transactionId')).toBeTrue()
        expect(component.openedTabs[1].state.transactionId).toBe(123)

        component.onSubnetFormSubmit(component.openedTabs[1].state)
        tick()

        expect(dhcpService.getSubnet).toHaveBeenCalled()
        expect(component.tabs.length).toBe(2)
        expect(component.openedTabs.length).toBe(2)
        expect(component.activeTabIndex).toBe(1)
        expect(component.openedTabs[1].tabType).toBe(TabType.Display)

        expect(dhcpService.updateSubnetDelete).not.toHaveBeenCalled()
    }))

    it('should cancel a new subnet transaction when a tab is closed', fakeAsync(() => {
        component.loadSubnets({})
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

        paramMapSubject.next(convertToParamMap({ id: 'new' }))
        fixture.detectChanges()
        tick()

        expect(component.openedTabs.length).toBe(2)

        fixture.detectChanges()
        tick()

        expect(dhcpService.createSubnetBegin).toHaveBeenCalled()

        expect(component.openedTabs.length).toBe(2)
        expect(component.openedTabs[1].state.hasOwnProperty('transactionId')).toBeTrue()
        expect(component.openedTabs[1].state.transactionId).toBe(123)

        component.closeTabByIndex(1)
        fixture.detectChanges()
        tick()

        expect(component.tabs.length).toBe(1)
        expect(component.activeTabIndex).toBe(0)

        expect(dhcpService.createSubnetDelete).toHaveBeenCalled()
    }))

    it('should cancel an update transaction when a tab is closed', async () => {
        component.loadSubnets({})
        await fixture.whenStable()
        fixture.detectChanges()

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

        const okResp: any = {
            status: 200,
        }

        spyOn(dhcpService, 'updateSubnetBegin').and.returnValue(of(updateSubnetBeginResp))
        spyOn(dhcpService, 'updateSubnetDelete').and.returnValue(of(okResp))
        spyOn(dhcpService, 'getSubnet').and.returnValue(of({ id: 5 }) as any)

        paramMapSubject.next(convertToParamMap({ id: 5 }))
        fixture.detectChanges()
        await fixture.whenStable()

        expect(component.openedTabs.length).toBe(2)

        component.onSubnetEditBegin({ id: 5 })
        fixture.detectChanges()
        await fixture.whenStable()

        expect(dhcpService.updateSubnetBegin).toHaveBeenCalled()

        expect(component.openedTabs.length).toBe(2)
        expect(component.openedTabs[1].state.hasOwnProperty('transactionId')).toBeTrue()
        expect(component.openedTabs[1].state.transactionId).toBe(123)

        component.closeTabByIndex(1)
        fixture.detectChanges()
        await fixture.whenStable()

        expect(component.tabs.length).toBe(1)
        expect(component.activeTabIndex).toBe(0)

        expect(dhcpService.updateSubnetDelete).toHaveBeenCalled()
    })

    it('should cancel a new subnet transaction when cancel button is clicked', fakeAsync(() => {
        component.loadSubnets({})
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

        paramMapSubject.next(convertToParamMap({ id: 'new' }))
        fixture.detectChanges()
        tick()

        expect(component.openedTabs.length).toBe(2)

        fixture.detectChanges()
        tick()

        expect(dhcpService.createSubnetBegin).toHaveBeenCalled()

        expect(component.openedTabs.length).toBe(2)
        expect(component.openedTabs[1].state.hasOwnProperty('transactionId')).toBeTrue()
        expect(component.openedTabs[1].state.transactionId).toBe(123)

        component.onSubnetFormCancel()
        fixture.detectChanges()
        tick()

        expect(component.tabs.length).toBe(1)
        expect(component.openedTabs.length).toBe(1)
        expect(component.activeTabIndex).toBe(0)
        expect(component.openedTabs[0].tabType).toBe(TabType.List)

        expect(dhcpService.createSubnetDelete).toHaveBeenCalled()
    }))

    it('should cancel transaction when cancel button is clicked', async () => {
        component.loadSubnets({})
        await fixture.whenStable()
        fixture.detectChanges()

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

        const okResp: any = {
            status: 200,
        }

        spyOn(dhcpService, 'updateSubnetBegin').and.returnValue(of(updateSubnetBeginResp))
        spyOn(dhcpService, 'updateSubnetDelete').and.returnValue(of(okResp))
        spyOn(dhcpService, 'getSubnet').and.returnValue(of({ id: 5 }) as any)

        paramMapSubject.next(convertToParamMap({ id: 5 }))
        fixture.detectChanges()
        await fixture.whenStable()

        expect(component.openedTabs.length).toBe(2)

        component.onSubnetEditBegin({ id: 5 })
        fixture.detectChanges()
        await fixture.whenStable()

        expect(dhcpService.updateSubnetBegin).toHaveBeenCalled()

        expect(component.openedTabs.length).toBe(2)
        expect(component.openedTabs[1].state.hasOwnProperty('transactionId')).toBeTrue()
        expect(component.openedTabs[1].state.transactionId).toBe(123)

        // Cancel editing. It should close the form and the transaction should be deleted.
        component.onSubnetFormCancel(5)
        fixture.detectChanges()
        await fixture.whenStable()

        expect(component.tabs.length).toBe(2)
        expect(component.openedTabs.length).toBe(2)
        expect(component.activeTabIndex).toBe(1)
        expect(component.openedTabs[1].tabType).toBe(TabType.Display)

        expect(dhcpService.updateSubnetDelete).toHaveBeenCalled()
    })

    it('should show error message when transaction canceling fails', fakeAsync(() => {
        component.loadSubnets({})
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
        spyOn(dhcpService, 'createSubnetDelete').and.returnValue(throwError({ status: 404 }))
        spyOn(messageService, 'add')

        paramMapSubject.next(convertToParamMap({ id: 'new' }))
        fixture.detectChanges()
        tick()

        expect(component.openedTabs.length).toBe(2)

        fixture.detectChanges()
        tick()

        expect(dhcpService.createSubnetBegin).toHaveBeenCalled()

        component.onSubnetFormCancel()
        fixture.detectChanges()
        tick()

        expect(dhcpService.createSubnetDelete).toHaveBeenCalled()
        expect(messageService.add).toHaveBeenCalled()
    }))

    it('should show error message when transaction canceling fails', async () => {
        component.loadSubnets({})
        await fixture.whenStable()
        fixture.detectChanges()

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

        spyOn(dhcpService, 'updateSubnetBegin').and.returnValue(of(updateSubnetBeginResp))
        spyOn(dhcpService, 'updateSubnetDelete').and.returnValue(throwError({ status: 404 }))
        spyOn(dhcpService, 'getSubnet').and.returnValue(of({ id: 5 }) as any)
        spyOn(messageService, 'add')

        paramMapSubject.next(convertToParamMap({ id: 5 }))
        fixture.detectChanges()
        await fixture.whenStable()

        expect(component.openedTabs.length).toBe(2)

        component.onSubnetEditBegin({ id: 5 })
        fixture.detectChanges()
        await fixture.whenStable()

        expect(dhcpService.updateSubnetBegin).toHaveBeenCalled()

        component.onSubnetFormCancel(5)
        fixture.detectChanges()
        await fixture.whenStable()

        expect(dhcpService.updateSubnetDelete).toHaveBeenCalled()
        expect(messageService.add).toHaveBeenCalled()
    })

    it('should close subnet tab when subnet is deleted', fakeAsync(() => {
        component.loadSubnets({})
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
        paramMapSubject.next(convertToParamMap({ id: 5 }))
        fixture.detectChanges()
        tick()
        expect(component.openedTabs.length).toBe(2)

        // Simulate the notification that the subnet has been deleted.
        component.onSubnetDelete(subnet)
        fixture.detectChanges()
        tick()

        // The main subnet tab should only be left.
        expect(component.openedTabs.length).toBe(1)
    }))
})
