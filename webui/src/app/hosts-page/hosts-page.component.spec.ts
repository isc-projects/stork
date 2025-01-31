import { ComponentFixture, fakeAsync, flush, TestBed, tick, waitForAsync } from '@angular/core/testing'

import { HostsPageComponent } from './hosts-page.component'
import { EntityLinkComponent } from '../entity-link/entity-link.component'
import { UntypedFormBuilder, FormsModule, ReactiveFormsModule } from '@angular/forms'
import { ConfirmationService, MessageService } from 'primeng/api'
import { TableModule } from 'primeng/table'
import { DHCPService, Host, LocalHost } from '../backend'
import { HttpClientTestingModule } from '@angular/common/http/testing'
import {
    ActivatedRoute,
    ActivatedRouteSnapshot,
    convertToParamMap,
    NavigationEnd,
    Router,
    RouterModule,
} from '@angular/router'
import { By } from '@angular/platform-browser'
import { of, throwError, BehaviorSubject } from 'rxjs'
import { BreadcrumbsComponent } from '../breadcrumbs/breadcrumbs.component'
import { TabMenu, TabMenuModule } from 'primeng/tabmenu'
import { HelpTipComponent } from '../help-tip/help-tip.component'
import { HostTabComponent } from '../host-tab/host-tab.component'
import { BreadcrumbModule } from 'primeng/breadcrumb'
import { OverlayPanelModule } from 'primeng/overlaypanel'
import { NoopAnimationsModule } from '@angular/platform-browser/animations'
import { TooltipModule } from 'primeng/tooltip'
import { FieldsetModule } from 'primeng/fieldset'
import { ProgressSpinnerModule } from 'primeng/progressspinner'
import { ToggleButtonModule } from 'primeng/togglebutton'
import { IdentifierComponent } from '../identifier/identifier.component'
import { ButtonModule } from 'primeng/button'
import { CheckboxModule } from 'primeng/checkbox'
import { DropdownModule } from 'primeng/dropdown'
import { MultiSelectModule } from 'primeng/multiselect'
import { ConfirmDialogModule } from 'primeng/confirmdialog'
import { HostFormComponent } from '../host-form/host-form.component'
import { DhcpOptionFormComponent } from '../dhcp-option-form/dhcp-option-form.component'
import { DhcpOptionSetFormComponent } from '../dhcp-option-set-form/dhcp-option-set-form.component'
import { DhcpOptionSetViewComponent } from '../dhcp-option-set-view/dhcp-option-set-view.component'
import { TreeModule } from 'primeng/tree'
import { DhcpClientClassSetFormComponent } from '../dhcp-client-class-set-form/dhcp-client-class-set-form.component'
import { DhcpClientClassSetViewComponent } from '../dhcp-client-class-set-view/dhcp-client-class-set-view.component'
import { ChipsModule } from 'primeng/chips'
import { DividerModule } from 'primeng/divider'
import { HostDataSourceLabelComponent } from '../host-data-source-label/host-data-source-label.component'
import { TagModule } from 'primeng/tag'
import { MessagesModule } from 'primeng/messages'
import { InputNumberModule } from 'primeng/inputnumber'
import { PluralizePipe } from '../pipes/pluralize.pipe'
import { HostsTableComponent } from '../hosts-table/hosts-table.component'
import { PanelModule } from 'primeng/panel'

describe('HostsPageComponent', () => {
    let component: HostsPageComponent
    let fixture: ComponentFixture<HostsPageComponent>
    let route: ActivatedRoute
    let router: Router
    let dhcpApi: DHCPService
    let messageService: MessageService
    let routerEventSubject: BehaviorSubject<NavigationEnd>

    beforeEach(waitForAsync(() => {
        TestBed.configureTestingModule({
            providers: [DHCPService, UntypedFormBuilder, ConfirmationService, MessageService],
            imports: [
                ButtonModule,
                ChipsModule,
                DividerModule,
                FormsModule,
                TableModule,
                HttpClientTestingModule,
                RouterModule.forRoot([
                    {
                        path: 'dhcp/hosts',
                        pathMatch: 'full',
                        redirectTo: 'dhcp/hosts/all',
                    },
                    {
                        path: 'dhcp/hosts/:id',
                        component: HostsPageComponent,
                    },
                ]),
                TabMenuModule,
                BreadcrumbModule,
                OverlayPanelModule,
                NoopAnimationsModule,
                TooltipModule,
                FieldsetModule,
                ProgressSpinnerModule,
                ToggleButtonModule,
                CheckboxModule,
                DropdownModule,
                MultiSelectModule,
                ReactiveFormsModule,
                ConfirmDialogModule,
                TreeModule,
                TagModule,
                MessagesModule,
                InputNumberModule,
                PanelModule,
            ],
            declarations: [
                EntityLinkComponent,
                HostsPageComponent,
                BreadcrumbsComponent,
                HelpTipComponent,
                HostTabComponent,
                IdentifierComponent,
                HostFormComponent,
                DhcpClientClassSetFormComponent,
                DhcpClientClassSetViewComponent,
                DhcpOptionFormComponent,
                DhcpOptionSetFormComponent,
                DhcpOptionSetViewComponent,
                HostDataSourceLabelComponent,
                PluralizePipe,
                HostsTableComponent,
            ],
        }).compileComponents()
    }))

    beforeEach(() => {
        fixture = TestBed.createComponent(HostsPageComponent)
        component = fixture.componentInstance
        route = fixture.debugElement.injector.get(ActivatedRoute)
        router = fixture.debugElement.injector.get(Router)
        dhcpApi = fixture.debugElement.injector.get(DHCPService)
        messageService = fixture.debugElement.injector.get(MessageService)

        route.snapshot = {
            paramMap: convertToParamMap({}),
            queryParamMap: convertToParamMap({}),
        } as ActivatedRouteSnapshot

        routerEventSubject = new BehaviorSubject(new NavigationEnd(1, 'dhcp/hosts', 'dhcp/hosts/all'))
        spyOnProperty(router, 'events').and.returnValue(routerEventSubject)

        fixture.detectChanges()

        // PrimeNG TabMenu is using setTimeout() logic when scrollable property is set to true.
        // This makes testing in fakeAsync zone unexpected, so disable 'scrollable' feature in tests.
        const m = fixture.debugElement.query(By.directive(TabMenu))
        if (m?.context) {
            m.context.scrollable = false
        }

        // PrimeNG table is stateful in the component, so clear stored filter between tests.
        component.table.table.clearFilterValues()

        fixture.detectChanges()
    })

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
        routerEventSubject.next(new NavigationEnd(eid, `dhcp/hosts/${params.id}`, `dhcp/hosts/${params.id}`))

        flush()
        fixture.detectChanges()
    }

    it('should create', () => {
        expect(component).toBeTruthy()
        expect(component.tabs.length).toBe(1)
        expect(component.activeTabIndex).toBe(0)
    })

    it('host table should have valid app name and app link', () => {
        component.table.hosts = [{ id: 1, localHosts: [{ appId: 1, appName: 'frog', dataSource: 'config' }] }]
        fixture.detectChanges()
        // Table rows have ids created by appending host id to the host-row- string.
        const row = fixture.debugElement.query(By.css('#host-row-1'))
        // There should be 6 table cells in the row.
        expect(row.children.length).toBe(6)
        // The last one includes the app name.
        const appNameTd = row.children[5]
        // The cell includes a link to the app.
        expect(appNameTd.children.length).toBe(1)
        const appLink = appNameTd.children[0]
        expect(appLink.nativeElement.textContent).toBe('frog config')
        // Verify that the link to the app is correct.
        const appLinkAnchor = appLink.query(By.css('a'))
        expect(appLinkAnchor.properties.hasOwnProperty('pathname')).toBeTrue()
        expect(appLinkAnchor.properties.pathname).toBe('/apps/kea/1')
    })

    it('should open and close host tabs', fakeAsync(() => {
        // Create a list with two hosts.
        component.table.hosts = [
            {
                id: 1,
                hostIdentifiers: [
                    {
                        idType: 'duid',
                        idHexValue: '01:02:03:04',
                    },
                ],
                addressReservations: [
                    {
                        address: '192.0.2.1',
                    },
                ],
                localHosts: [
                    {
                        appId: 1,
                        appName: 'frog',
                        dataSource: 'config',
                    },
                ],
            },
            {
                id: 2,
                hostIdentifiers: [
                    {
                        idType: 'duid',
                        idHexValue: '11:12:13:14',
                    },
                ],
                addressReservations: [
                    {
                        address: '192.0.2.2',
                    },
                ],
                localHosts: [
                    {
                        appId: 2,
                        appName: 'mouse',
                        dataSource: 'config',
                    },
                ],
            },
        ]
        fixture.detectChanges()

        // Ensure that we don't fetch the host information from the server upon
        // opening a new tab. We should use the information available in the
        // hosts structure.
        spyOn(dhcpApi, 'getHost')

        // Open tab with host with id 1.
        navigate({ id: 1 })
        expect(component.tabs.length).toBe(2)
        expect(component.activeTabIndex).toBe(1)

        // Open the tab for creating a host.
        navigate({ id: 'new' })
        expect(component.tabs.length).toBe(3)
        expect(component.activeTabIndex).toBe(2)

        // Open tab with host with id 2.
        navigate({ id: 2 })
        expect(component.tabs.length).toBe(4)
        expect(component.activeTabIndex).toBe(3)

        // Navigate back to the hosts list in the first tab.
        navigate({})
        expect(component.tabs.length).toBe(4)
        expect(component.activeTabIndex).toBe(0)

        // Navigate to the existing tab with host with id 1.
        navigate({ id: 1 })
        expect(component.tabs.length).toBe(4)
        expect(component.activeTabIndex).toBe(1)

        // navigate to the existing tab for adding new host.
        navigate({ id: 'new' })
        expect(component.tabs.length).toBe(4)
        expect(component.activeTabIndex).toBe(2)

        // Close the second tab.
        component.closeHostTab(null, 1)
        fixture.detectChanges()
        expect(component.tabs.length).toBe(3)
        expect(component.activeTabIndex).toBe(1)

        // Close the tab for adding new host.
        component.closeHostTab(null, 1)
        fixture.detectChanges()
        expect(component.tabs.length).toBe(2)
        expect(component.activeTabIndex).toBe(0)

        // Close the remaining tab.
        component.closeHostTab(null, 1)
        fixture.detectChanges()
        expect(component.tabs.length).toBe(1)
        expect(component.activeTabIndex).toBe(0)
    }))

    it('should emit error message when there is an error deleting transaction for new host', fakeAsync(() => {
        // Open the tab for creating a host.
        navigate({ id: 'new' })
        expect(component.tabs.length).toBe(2)
        expect(component.activeTabIndex).toBe(1)

        // Ensure an error is emitted when transaction is deleted.
        component.openedTabs[0].state.transactionId = 123
        spyOn(dhcpApi, 'createHostDelete').and.returnValue(throwError({ status: 404 }))
        spyOn(messageService, 'add')

        // Close the tab for adding new host.
        component.closeHostTab(null, 1)
        tick()
        fixture.detectChanges()
        expect(component.tabs.length).toBe(1)
        expect(component.activeTabIndex).toBe(0)
        expect(messageService.add).toHaveBeenCalled()
    }))

    it('should switch a tab to host editing mode', fakeAsync(() => {
        // Create a list with two hosts.
        component.table.hosts = [
            {
                id: 1,
                hostIdentifiers: [
                    {
                        idType: 'duid',
                        idHexValue: '01:02:03:04',
                    },
                ],
                addressReservations: [
                    {
                        address: '192.0.2.1',
                    },
                ],
                localHosts: [
                    {
                        appId: 1,
                        appName: 'frog',
                        dataSource: 'config',
                    },
                ],
            },
            {
                id: 2,
                hostIdentifiers: [
                    {
                        idType: 'duid',
                        idHexValue: '11:12:13:14',
                    },
                ],
                addressReservations: [
                    {
                        address: '192.0.2.2',
                    },
                ],
                localHosts: [
                    {
                        appId: 2,
                        appName: 'mouse',
                        dataSource: 'config',
                    },
                ],
            },
        ]
        fixture.detectChanges()

        // Ensure that we don't fetch the host information from the server upon
        // opening a new tab. We should use the information available in the
        // hosts structure.
        spyOn(dhcpApi, 'getHost')

        // Open tab with host with id 1.
        navigate({ id: 1 })
        expect(component.tabs.length).toBe(2)
        expect(component.activeTabIndex).toBe(1)

        component.onHostEditBegin(component.table.hosts[0])
        fixture.detectChanges()
        expect(component.tabs.length).toBe(2)
        expect(component.activeTabIndex).toBe(1)

        // Make sure the tab includes the host reservation form.
        expect(component.tabs[1].icon).toBe('pi pi-pencil')
        let form = fixture.debugElement.query(By.css('form'))
        expect(form).toBeTruthy()

        // Open tab with host with id 2.
        navigate({ id: 2 })
        expect(component.tabs.length).toBe(3)
        expect(component.activeTabIndex).toBe(2)
        // This tab should have no form.
        form = fixture.debugElement.query(By.css('form'))
        expect(form).toBeFalsy()

        // Return to the previous tab and make sure the form is still open.
        navigate({ id: 1 })
        expect(component.tabs.length).toBe(3)
        expect(component.activeTabIndex).toBe(1)
        form = fixture.debugElement.query(By.css('form'))
        expect(form).toBeTruthy()
    }))

    it('should emit an error when deleting transaction for updating a host fails', fakeAsync(() => {
        // Create a list with two hosts.
        component.table.hosts = [
            {
                id: 1,
                hostIdentifiers: [
                    {
                        idType: 'duid',
                        idHexValue: '01:02:03:04',
                    },
                ],
                addressReservations: [
                    {
                        address: '192.0.2.1',
                    },
                ],
                localHosts: [
                    {
                        appId: 1,
                        appName: 'frog',
                        dataSource: 'config',
                    },
                ],
            },
        ]
        fixture.detectChanges()

        // Ensure that we don't fetch the host information from the server upon
        // opening a new tab. We should use the information available in the
        // hosts structure.
        spyOn(dhcpApi, 'getHost')

        // Open tab with host with id 1.
        navigate({ id: 1 })
        expect(component.tabs.length).toBe(2)
        expect(component.activeTabIndex).toBe(1)

        // Simulate clicking on Edit.
        component.onHostEditBegin(component.table.hosts[0])
        fixture.detectChanges()
        expect(component.tabs.length).toBe(2)
        expect(component.activeTabIndex).toBe(1)

        // Make sure an error is returned when closing the tab.
        component.openedTabs[0].state.transactionId = 123
        spyOn(dhcpApi, 'updateHostDelete').and.returnValue(throwError({ status: 404 }))
        spyOn(messageService, 'add')

        // Close the tab.
        component.closeHostTab(null, 1)
        tick()
        fixture.detectChanges()
        expect(component.tabs.length).toBe(1)
        expect(component.activeTabIndex).toBe(0)
        expect(messageService.add).toHaveBeenCalled()
    }))

    it('should open a tab when hosts have not been loaded', fakeAsync(() => {
        const host: any = {
            id: 1,
            hostIdentifiers: [
                {
                    idType: 'duid',
                    idHexValue: '01:02:03:04',
                },
            ],
            addressReservations: [
                {
                    address: '192.0.2.1',
                },
            ],
            localHosts: [
                {
                    appId: 1,
                    appName: 'frog',
                    dataSource: 'config',
                },
            ],
        }
        // Do not initialize the hosts list. Instead, simulate returning the
        // host information from the server. The component should send the
        // request to the server to get the host.
        spyOn(dhcpApi, 'getHost').and.returnValue(of(host))
        navigate({ id: 1 })
        // There should be two tabs opened. One with the list of hosts and one
        // with the host details.
        expect(component.tabs.length).toBe(2)
    }))

    it('should not open a tab when getting host information erred', fakeAsync(() => {
        // Simulate the getHost call to return an error.
        spyOn(dhcpApi, 'getHost').and.returnValue(throwError({ status: 404 }))
        spyOn(messageService, 'add')
        navigate({ id: 1 })
        // There should still be one tab open with a list of hosts.
        expect(component.tabs.length).toBe(1)
        // Ensure that the error message was displayed.
        expect(messageService.add).toHaveBeenCalled()
    }))

    it('should generate a label from host information', () => {
        const host0 = {
            id: 1,
            hostIdentifiers: [
                {
                    idType: 'duid',
                    idHexValue: '01:02:03:04',
                },
            ],
            addressReservations: [
                {
                    address: '192.0.2.1',
                },
            ],
            prefixReservations: [
                {
                    address: '2001:db8::',
                },
            ],
            hostname: 'mouse.example.org',
        }

        expect(component.getHostLabel(host0)).toBe('192.0.2.1')

        const host1 = {
            id: 1,
            hostIdentifiers: [
                {
                    idType: 'duid',
                    idHexValue: '01:02:03:04',
                },
            ],
            prefixReservations: [
                {
                    address: '2001:db8::',
                },
            ],
            hostname: 'mouse.example.org',
        }

        expect(component.getHostLabel(host1)).toBe('2001:db8::')

        const host2 = {
            id: 1,
            hostIdentifiers: [
                {
                    idType: 'duid',
                    idHexValue: '01:02:03:04',
                },
            ],
            hostname: 'mouse.example.org',
        }
        expect(component.getHostLabel(host2)).toBe('mouse.example.org')

        const host3 = {
            id: 1,
            hostIdentifiers: [
                {
                    idType: 'duid',
                    idHexValue: '01:02:03:04',
                },
            ],
        }
        expect(component.getHostLabel(host3)).toBe('duid=01:02:03:04')

        const host4 = {
            id: 1,
        }
        expect(component.getHostLabel(host4)).toBe('[1]')
    })

    it('should display well formatted host identifiers', () => {
        // Create a list with three hosts. One host uses a duid convertible
        // to a textual format. Another host uses a hw-address which is
        // by default displayed in the hex format. Third host uses a
        // flex-id which is not convertible to a textual format.
        component.table.hosts = [
            {
                id: 1,
                hostIdentifiers: [
                    {
                        idType: 'duid',
                        idHexValue: '61:62:63:64',
                    },
                ],
                addressReservations: [
                    {
                        address: '192.0.2.1',
                    },
                ],
                localHosts: [
                    {
                        appId: 1,
                        appName: 'frog',
                        dataSource: 'config',
                    },
                ],
            },
            {
                id: 2,
                hostIdentifiers: [
                    {
                        idType: 'hw-address',
                        idHexValue: '51:52:53:54:55:56',
                    },
                ],
                addressReservations: [
                    {
                        address: '192.0.2.2',
                    },
                ],
                localHosts: [
                    {
                        appId: 2,
                        appName: 'mouse',
                        dataSource: 'config',
                    },
                ],
            },
            {
                id: 3,
                hostIdentifiers: [
                    {
                        idType: 'flex-id',
                        idHexValue: '10:20:30:40:50',
                    },
                ],
                addressReservations: [
                    {
                        address: '192.0.2.2',
                    },
                ],
                localHosts: [
                    {
                        appId: 3,
                        appName: 'lion',
                        dataSource: 'config',
                    },
                ],
            },
        ]
        fixture.detectChanges()

        // There should be 3 hosts listed.
        const identifierEl = fixture.debugElement.queryAll(By.css('app-identifier'))
        expect(identifierEl.length).toBe(3)

        // Each host identifier should be a link.
        const firstIdEl = identifierEl[0].query(By.css('a'))
        expect(firstIdEl).toBeTruthy()
        // The DUID is convertible to text.
        expect(firstIdEl.nativeElement.textContent).toContain('duid=(abcd)')
        expect(firstIdEl.attributes.href).toBe('/dhcp/hosts/1')

        const secondIdEl = identifierEl[1].query(By.css('a'))
        expect(secondIdEl).toBeTruthy()
        // The HW address is convertible but by default should be in hex format.
        expect(secondIdEl.nativeElement.textContent).toContain('hw-address=(51:52:53:54:55:56)')
        expect(secondIdEl.attributes.href).toBe('/dhcp/hosts/2')

        const thirdIdEl = identifierEl[2].query(By.css('a'))
        expect(thirdIdEl).toBeTruthy()
        // The flex-id is not convertible to text so should be in hex format.
        expect(thirdIdEl.nativeElement.textContent).toContain('flex-id=(10:20:30:40:50)')
        expect(thirdIdEl.attributes.href).toBe('/dhcp/hosts/3')
    })

    it('should close new host tab when form is submitted', fakeAsync(() => {
        const createHostBeginResp: any = {
            id: 123,
            subnets: [
                {
                    id: 1,
                    subnet: '192.0.2.0/24',
                    localSubnets: [
                        {
                            daemonId: 1,
                        },
                    ],
                },
            ],
            daemons: [
                {
                    id: 1,
                    name: 'dhcp4',
                    app: {
                        name: 'first',
                    },
                },
            ],
            clientClasses: ['router', 'cable-modem'],
        }
        const okResp: any = {
            status: 200,
        }

        spyOn(dhcpApi, 'createHostBegin').and.returnValue(of(createHostBeginResp))
        spyOn(dhcpApi, 'createHostDelete').and.returnValue(of(okResp))

        navigate({ id: 'new' })
        navigate({})

        expect(component.openedTabs.length).toBe(1)
        expect(component.openedTabs[0].state.hasOwnProperty('transactionId')).toBeTrue()
        expect(component.openedTabs[0].state.transactionId).toBe(123)

        component.onHostFormSubmit(component.openedTabs[0].state)
        tick()
        expect(component.tabs.length).toBe(1)
        expect(component.activeTabIndex).toBe(0)

        expect(dhcpApi.createHostDelete).not.toHaveBeenCalled()
    }))

    it('should cancel transaction when a tab is closed', fakeAsync(() => {
        const createHostBeginResp: any = {
            id: 123,
            subnets: [
                {
                    id: 1,
                    subnet: '192.0.2.0/24',
                    localSubnets: [
                        {
                            daemonId: 1,
                        },
                    ],
                },
            ],
            daemons: [
                {
                    id: 1,
                    name: 'dhcp4',
                    app: {
                        name: 'first',
                    },
                },
            ],
            clientClasses: ['router', 'cable-modem'],
        }
        const okResp: any = {
            status: 200,
        }
        spyOn(dhcpApi, 'createHostBegin').and.returnValue(of(createHostBeginResp))
        spyOn(dhcpApi, 'createHostDelete').and.returnValue(of(okResp))

        navigate({ id: 'new' })
        navigate({})

        expect(component.openedTabs.length).toBe(1)
        expect(component.openedTabs[0].state.hasOwnProperty('transactionId')).toBeTrue()
        expect(component.openedTabs[0].state.transactionId).toBe(123)

        component.closeHostTab(null, 1)
        tick()
        fixture.detectChanges()
        expect(component.tabs.length).toBe(1)
        expect(component.activeTabIndex).toBe(0)

        expect(dhcpApi.createHostDelete).toHaveBeenCalled()
    }))

    it('should cancel transaction when cancel button is clicked', fakeAsync(() => {
        const createHostBeginResp: any = {
            id: 123,
            subnets: [
                {
                    id: 1,
                    subnet: '192.0.2.0/24',
                    localSubnets: [
                        {
                            daemonId: 1,
                        },
                    ],
                },
            ],
            daemons: [
                {
                    id: 1,
                    name: 'dhcp4',
                    app: {
                        name: 'first',
                    },
                },
            ],
            clientClasses: ['router', 'cable-modem'],
        }
        const okResp: any = {
            status: 200,
        }
        spyOn(dhcpApi, 'createHostBegin').and.returnValue(of(createHostBeginResp))
        spyOn(dhcpApi, 'createHostDelete').and.returnValue(of(okResp))

        navigate({ id: 'new' })
        navigate({})

        expect(component.openedTabs.length).toBe(1)
        expect(component.openedTabs[0].state.hasOwnProperty('transactionId')).toBeTrue()
        expect(component.openedTabs[0].state.transactionId).toBe(123)

        // Cancel editing. It should close the tab and the transaction should be deleted.
        component.onHostFormCancel(0)
        tick()
        fixture.detectChanges()
        expect(component.tabs.length).toBe(1)
        expect(component.activeTabIndex).toBe(0)
        expect(dhcpApi.createHostDelete).toHaveBeenCalled()
    }))

    it('should cancel update transaction when a tab is closed', fakeAsync(() => {
        component.table.hosts = [
            {
                id: 1,
                subnetId: 1,
                subnetPrefix: '192.0.2.0/24',
                hostIdentifiers: [
                    {
                        idType: 'duid',
                        idHexValue: '01:02:03:04',
                    },
                ],
                addressReservations: [
                    {
                        address: '192.0.2.1',
                    },
                ],
                prefixReservations: [],
                localHosts: [
                    {
                        daemonId: 1,
                        dataSource: 'api',
                        options: [],
                    },
                ],
            },
        ]
        fixture.detectChanges()

        // Ensure that we don't fetch the host information from the server upon
        // opening a new tab. We should use the information available in the
        // hosts structure.
        spyOn(dhcpApi, 'getHost')

        // Open tab with host with id 1.
        navigate({ id: 1 })
        expect(component.tabs.length).toBe(2)
        expect(component.activeTabIndex).toBe(1)
        expect(component.openedTabs.length).toBe(1)

        const updateHostBeginResp: any = {
            id: 123,
            subnets: [
                {
                    id: 1,
                    subnet: '192.0.2.0/24',
                    localSubnets: [
                        {
                            daemonId: 1,
                        },
                    ],
                },
            ],
            daemons: [
                {
                    id: 1,
                    name: 'dhcp4',
                    app: {
                        name: 'first',
                    },
                },
            ],
            clientClasses: ['router', 'cable-modem'],
            host: component.table.hosts[0],
        }
        const okResp: any = {
            status: 200,
        }
        spyOn(dhcpApi, 'updateHostBegin').and.returnValue(of(updateHostBeginResp))
        spyOn(dhcpApi, 'updateHostDelete').and.returnValue(of(okResp))

        component.onHostEditBegin(component.table.hosts[0])
        fixture.detectChanges()
        tick()

        expect(dhcpApi.updateHostBegin).toHaveBeenCalled()
        expect(dhcpApi.updateHostDelete).not.toHaveBeenCalled()

        expect(component.openedTabs.length).toBe(1)
        expect(component.openedTabs[0].state.hasOwnProperty('transactionId')).toBeTrue()
        expect(component.openedTabs[0].state.transactionId).toBe(123)

        component.closeHostTab(null, 1)
        tick()
        fixture.detectChanges()
        expect(component.tabs.length).toBe(1)
        expect(component.activeTabIndex).toBe(0)

        expect(dhcpApi.updateHostDelete).toHaveBeenCalled()
    }))

    it('should cancel update transaction cancel button is clicked', fakeAsync(() => {
        component.table.hosts = [
            {
                id: 1,
                subnetId: 1,
                subnetPrefix: '192.0.2.0/24',
                hostIdentifiers: [
                    {
                        idType: 'duid',
                        idHexValue: '01:02:03:04',
                    },
                ],
                addressReservations: [
                    {
                        address: '192.0.2.1',
                    },
                ],
                prefixReservations: [],
                localHosts: [
                    {
                        daemonId: 1,
                        dataSource: 'api',
                        options: [],
                    },
                ],
            },
        ]
        fixture.detectChanges()

        // Ensure that we don't fetch the host information from the server upon
        // opening a new tab. We should use the information available in the
        // hosts structure.
        spyOn(dhcpApi, 'getHost')

        // Open tab with host with id 1.
        navigate({ id: 1 })
        expect(component.tabs.length).toBe(2)
        expect(component.activeTabIndex).toBe(1)
        expect(component.openedTabs.length).toBe(1)

        const updateHostBeginResp: any = {
            id: 123,
            subnets: [
                {
                    id: 1,
                    subnet: '192.0.2.0/24',
                    localSubnets: [
                        {
                            daemonId: 1,
                        },
                    ],
                },
            ],
            daemons: [
                {
                    id: 1,
                    name: 'dhcp4',
                    app: {
                        name: 'first',
                    },
                },
            ],
            clientClasses: ['router', 'cable-modem'],
            host: component.table.hosts[0],
        }
        const okResp: any = {
            status: 200,
        }
        spyOn(dhcpApi, 'updateHostBegin').and.returnValue(of(updateHostBeginResp))
        spyOn(dhcpApi, 'updateHostDelete').and.returnValue(of(okResp))

        component.onHostEditBegin(component.table.hosts[0])
        fixture.detectChanges()
        tick()

        expect(dhcpApi.updateHostBegin).toHaveBeenCalled()
        expect(dhcpApi.updateHostDelete).not.toHaveBeenCalled()

        expect(component.openedTabs.length).toBe(1)
        expect(component.openedTabs[0].state.hasOwnProperty('transactionId')).toBeTrue()
        expect(component.openedTabs[0].state.transactionId).toBe(123)

        component.onHostFormCancel(component.table.hosts[0].id)
        tick()
        fixture.detectChanges()
        expect(component.tabs.length).toBe(2)
        expect(component.activeTabIndex).toBe(1)
        expect(dhcpApi.updateHostDelete).toHaveBeenCalled()

        // Ensure that the form was closed and the tab now shows the host
        // reservation view.
        expect(fixture.debugElement.query(By.css('app-host-tab'))).toBeTruthy()
    }))

    it('should close a tab after deleting a host', fakeAsync(() => {
        // Create a list with two hosts.
        component.table.hosts = [
            {
                id: 1,
                hostIdentifiers: [
                    {
                        idType: 'duid',
                        idHexValue: '01:02:03:04',
                    },
                ],
                addressReservations: [
                    {
                        address: '192.0.2.1',
                    },
                ],
                localHosts: [
                    {
                        appId: 1,
                        appName: 'frog',
                        dataSource: 'config',
                    },
                ],
            },
            {
                id: 2,
                hostIdentifiers: [
                    {
                        idType: 'duid',
                        idHexValue: '11:12:13:14',
                    },
                ],
                addressReservations: [
                    {
                        address: '192.0.2.2',
                    },
                ],
                localHosts: [
                    {
                        appId: 2,
                        appName: 'mouse',
                        dataSource: 'config',
                    },
                ],
            },
        ]
        fixture.detectChanges()

        // Open tab with host with id 1.
        navigate({ id: 1 })
        expect(component.tabs.length).toBe(2)
        expect(component.activeTabIndex).toBe(1)

        // Open tab with host with id 2.
        navigate({ id: 2 })
        expect(component.tabs.length).toBe(3)
        expect(component.activeTabIndex).toBe(2)

        // Ensure that the component reloads hosts after deleting one of them
        // and switching to the hosts list.
        spyOn(dhcpApi, 'getHosts').and.callThrough()

        // Delete an existing host. The tab should be closed and the tab for
        // the other host should remain open.
        component.onHostDelete({ id: 2 })
        fixture.detectChanges()
        expect(component.tabs.length).toBe(2)
        expect(component.activeTabIndex).toBe(1)
        expect(dhcpApi.getHosts).toHaveBeenCalledTimes(0)

        // Closing a host without specifying its ID should not throw.
        expect(() => {
            component.onHostDelete({})
        }).not.toThrow()
        fixture.detectChanges()
        expect(component.tabs.length).toBe(2)
        expect(component.activeTabIndex).toBe(1)
        expect(dhcpApi.getHosts).toHaveBeenCalledTimes(0)

        // Closing a host with invalid ID should not throw too.
        expect(() => {
            component.onHostDelete({ id: 5 })
        }).not.toThrow()
        fixture.detectChanges()
        expect(component.tabs.length).toBe(2)
        expect(component.activeTabIndex).toBe(1)
        expect(dhcpApi.getHosts).toHaveBeenCalledTimes(0)

        // Mock router.navigate(['/dhcp/hosts/all']) to call fake navigation.
        // The navigation happens when closing current Host tab.
        spyOn(router, 'navigate')
            .withArgs(['/dhcp/hosts/all'])
            .and.callFake(() => {
                navigate({ id: 'all' })
                return Promise.resolve(true)
            })

        // Closing the existing host should result in closing its tab.
        // It should select the first tab and reload the hosts.
        component.onHostDelete({ id: 1 })
        fixture.detectChanges()
        expect(component.tabs.length).toBe(1)
        expect(component.activeTabIndex).toBe(0)
        expect(router.navigate).toHaveBeenCalledTimes(1)
        expect(dhcpApi.getHosts).toHaveBeenCalledTimes(1)
    }))

    it('should contain a refresh button', () => {
        const refreshBtn = fixture.debugElement.query(By.css('[label="Refresh List"]'))
        expect(refreshBtn).toBeTruthy()

        spyOn(dhcpApi, 'getHosts').and.returnValue(throwError({ status: 404 }))
        refreshBtn.componentInstance.onClick.emit(new Event('click'))
        fixture.detectChanges()
        expect(dhcpApi.getHosts).toHaveBeenCalled()
    })

    it('should have breadcrumbs', () => {
        const breadcrumbsElement = fixture.debugElement.query(By.directive(BreadcrumbsComponent))
        expect(breadcrumbsElement).not.toBeNull()
        const breadcrumbsComponent = breadcrumbsElement.componentInstance as BreadcrumbsComponent
        expect(breadcrumbsComponent).not.toBeNull()
        expect(breadcrumbsComponent.items).toHaveSize(2)
        expect(breadcrumbsComponent.items[0].label).toEqual('DHCP')
        expect(breadcrumbsComponent.items[1].label).toEqual('Host Reservations')
    })

    it('hosts list should be filtered by appId', fakeAsync(() => {
        component.table.hosts = [{ id: 1, localHosts: [{ appId: 1, appName: 'frog', dataSource: 'config' }] }]
        fixture.detectChanges()

        spyOn(dhcpApi, 'getHosts').and.callThrough()

        component.table.filter$.next({ appId: 2 })
        tick()
        fixture.detectChanges()

        expect(dhcpApi.getHosts).toHaveBeenCalledWith(0, 10, 2, null, null, null, null, null)

        expect(fixture.debugElement.query(By.css('.p-error'))).toBeFalsy()
    }))

    it('should display error message when appId is invalid', fakeAsync(() => {
        component.table.hosts = [{ id: 1, localHosts: [{ appId: 1, appName: 'frog', dataSource: 'config' }] }]
        fixture.detectChanges()

        spyOn(dhcpApi, 'getHosts').and.callThrough()

        component.table.updateFilterFromQueryParameters(convertToParamMap({ appId: 'abc' }))
        tick()
        fixture.detectChanges()

        // Invalid filter should not be applied, so dhcpApi.getHosts should be called with default params.
        expect(dhcpApi.getHosts).toHaveBeenCalledWith(0, 10, null, null, null, null, null, null)

        const errMsg = fixture.debugElement.query(By.css('.p-error'))
        expect(errMsg).toBeTruthy()
        expect(errMsg.nativeElement.innerText).toBe('Please specify appId as a number (e.g., appId=4).')
    }))

    it('hosts list should be filtered by subnetId', fakeAsync(() => {
        component.table.hosts = [{ id: 1, localHosts: [{ appId: 1, appName: 'frog', dataSource: 'config' }] }]
        fixture.detectChanges()

        spyOn(dhcpApi, 'getHosts').and.callThrough()

        component.table.filter$.next({ subnetId: 89 })
        tick()
        fixture.detectChanges()

        expect(dhcpApi.getHosts).toHaveBeenCalledWith(0, 10, null, 89, null, null, null, null)

        expect(fixture.debugElement.query(By.css('.p-error'))).toBeFalsy()
    }))

    it('should display error message when subnetId is invalid', fakeAsync(() => {
        component.table.hosts = [{ id: 1, localHosts: [{ appId: 1, appName: 'frog', dataSource: 'config' }] }]
        fixture.detectChanges()

        spyOn(dhcpApi, 'getHosts').and.callThrough()

        component.table.queryParamNumericKeys = ['subnetId']
        component.table.updateFilterFromQueryParameters(convertToParamMap({ subnetId: 'abc' }))
        tick()
        fixture.detectChanges()

        // Invalid filter should not be applied, so dhcpApi.getHosts should be called with default params.
        expect(dhcpApi.getHosts).toHaveBeenCalledWith(0, 10, null, null, null, null, null, null)

        const errMsg = fixture.debugElement.query(By.css('.p-error'))
        expect(errMsg).toBeTruthy()
        expect(errMsg.nativeElement.innerText).toBe('Please specify subnetId as a number (e.g., subnetId=4).')
    }))

    it('hosts list should be filtered by conflicts', fakeAsync(() => {
        component.table.hosts = [{ id: 1, localHosts: [{ appId: 1, appName: 'frog', dataSource: 'config' }] }]
        fixture.detectChanges()

        spyOn(dhcpApi, 'getHosts').and.callThrough()

        component.table.filter$.next({ conflict: true })
        tick()
        fixture.detectChanges()

        expect(dhcpApi.getHosts).toHaveBeenCalledWith(0, 10, null, null, null, null, null, true)

        expect(fixture.debugElement.query(By.css('.p-error'))).toBeFalsy()
    }))

    it('hosts list should be filtered by non-conflicts', fakeAsync(() => {
        component.table.hosts = [{ id: 1, localHosts: [{ appId: 1, appName: 'frog', dataSource: 'config' }] }]
        fixture.detectChanges()

        spyOn(dhcpApi, 'getHosts').and.callThrough()

        component.table.filter$.next({ conflict: false })
        tick()
        fixture.detectChanges()

        expect(dhcpApi.getHosts).toHaveBeenCalledWith(0, 10, null, null, null, null, null, false)

        expect(fixture.debugElement.query(By.css('.p-error'))).toBeFalsy()
    }))

    it('hosts list should be filtered by keaSubnetId', fakeAsync(() => {
        component.table.hosts = [{ id: 1, localHosts: [{ appId: 1, appName: 'frog', dataSource: 'config' }] }]
        fixture.detectChanges()

        spyOn(dhcpApi, 'getHosts').and.callThrough()

        component.table.filter$.next({ keaSubnetId: 101 })
        tick()
        fixture.detectChanges()

        expect(dhcpApi.getHosts).toHaveBeenCalledWith(0, 10, null, null, 101, null, null, null)

        expect(fixture.debugElement.query(By.css('.p-error'))).toBeFalsy()
    }))

    it('should display error message when keaSubnetId is invalid', fakeAsync(() => {
        component.table.hosts = [{ id: 1, localHosts: [{ appId: 1, appName: 'frog', dataSource: 'config' }] }]
        fixture.detectChanges()

        spyOn(dhcpApi, 'getHosts').and.callThrough()

        component.table.queryParamNumericKeys = ['keaSubnetId']
        component.table.updateFilterFromQueryParameters(convertToParamMap({ keaSubnetId: 'abc' }))
        tick()
        fixture.detectChanges()

        // Invalid filter should not be applied, so dhcpApi.getHosts should be called with default params.
        expect(dhcpApi.getHosts).toHaveBeenCalledWith(0, 10, null, null, null, null, null, null)

        const errMsg = fixture.debugElement.query(By.css('.p-error'))
        expect(errMsg).toBeTruthy()
        expect(errMsg.nativeElement.innerText).toBe('Please specify keaSubnetId as a number (e.g., keaSubnetId=4).')
    }))

    it('should display multiple error message for each invalid value', fakeAsync(() => {
        component.table.hosts = [{ id: 1, localHosts: [{ appId: 1, appName: 'frog', dataSource: 'config' }] }]
        fixture.detectChanges()

        spyOn(dhcpApi, 'getHosts').and.callThrough()

        component.table.queryParamNumericKeys = ['subnetId']
        component.table.queryParamBooleanKeys = ['isGlobal']
        component.table.updateFilterFromQueryParameters(
            convertToParamMap({ appId: 'foo', subnetId: 'bar', isGlobal: 'tru' })
        )

        tick()
        fixture.detectChanges()

        // Invalid filter should not be applied, so dhcpApi.getHosts should be called with default params.
        expect(dhcpApi.getHosts).toHaveBeenCalledWith(0, 10, null, null, null, null, null, null)

        const errMsgs = fixture.debugElement.queryAll(By.css('.p-error'))
        expect(errMsgs.length).toBe(3)
    }))

    it('should group the local hosts by appId', () => {
        const host = {
            id: 42,
            localHosts: [
                {
                    appId: 3,
                    daemonId: 31,
                },
                {
                    appId: 3,
                    daemonId: 32,
                },
                {
                    appId: 3,
                    daemonId: 33,
                },
                {
                    appId: 2,
                    daemonId: 21,
                },
                {
                    appId: 2,
                    daemonId: 22,
                },
                {
                    appId: 1,
                    daemonId: 11,
                },
            ],
        } as Host

        component.table.hosts = [host]
        const groups = component.table.localHostsGroupedByApp[host.id]

        expect(groups.length).toBe(3)
        for (let group of groups) {
            expect(group.length).toBeGreaterThanOrEqual(1)
            const appId = group[0].appId
            expect(group.length).toBe(appId)
            for (let item of group) {
                expect(item.daemonId).toBeGreaterThan(10 * appId)
                expect(item.daemonId).toBeLessThan(10 * (appId + 1))
            }
        }
    })

    it('should recognize the state of local hosts', () => {
        // Conflict
        let localHosts = [
            {
                appId: 1,
                daemonId: 1,
                nextServer: 'foo',
            },
            {
                appId: 1,
                daemonId: 2,
                nextServer: 'bar',
            },
        ] as LocalHost[]

        let state = component.table.getLocalHostsState(localHosts)
        expect(state).toBe('conflict')

        // Duplicate
        localHosts = [
            {
                appId: 1,
                daemonId: 1,
                nextServer: 'foo',
            },
            {
                appId: 1,
                daemonId: 2,
                nextServer: 'foo',
            },
        ] as LocalHost[]

        state = component.table.getLocalHostsState(localHosts)
        expect(state).toBe('duplicate')

        // Null
        localHosts = [
            {
                appId: 1,
                daemonId: 1,
                nextServer: 'foo',
            },
        ] as LocalHost[]

        state = component.table.getLocalHostsState(localHosts)
        expect(state).toBeNull()
    })
})
