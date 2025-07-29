import { ComponentFixture, fakeAsync, TestBed, tick, waitForAsync } from '@angular/core/testing'

import { DashboardComponent } from './dashboard.component'
import { PanelModule } from 'primeng/panel'
import { ButtonModule } from 'primeng/button'
import {
    AppsStats,
    AppsVersions,
    DhcpOverview,
    DHCPService,
    DNSService,
    ServicesService,
    SettingsService,
    UsersService,
} from '../backend'
import { provideHttpClientTesting } from '@angular/common/http/testing'
import { MessageService } from 'primeng/api'
import { LocationStrategy, PathLocationStrategy } from '@angular/common'
import { of } from 'rxjs'
import { By } from '@angular/platform-browser'
import { ServerDataService } from '../server-data.service'
import { NoopAnimationsModule } from '@angular/platform-browser/animations'
import { EventsPanelComponent } from '../events-panel/events-panel.component'
import { PaginatorModule } from 'primeng/paginator'
import { HelpTipComponent } from '../help-tip/help-tip.component'
import { SubnetBarComponent } from '../subnet-bar/subnet-bar.component'
import { OverlayPanelModule } from 'primeng/overlaypanel'
import { TooltipModule } from 'primeng/tooltip'
import { TableModule } from 'primeng/table'
import { ConfirmDialogModule } from 'primeng/confirmdialog'
import { HumanCountPipe } from '../pipes/human-count.pipe'
import { SurroundPipe } from '../pipes/surround.pipe'
import { EntityLinkComponent } from '../entity-link/entity-link.component'
import { ServerSentEventsService, ServerSentEventsTestingService } from '../server-sent-events.service'
import { SettingService } from '../setting.service'
import { VersionStatusComponent } from '../version-status/version-status.component'
import { Severity, VersionService } from '../version.service'
import { provideHttpClient, withInterceptorsFromDi } from '@angular/common/http'
import { TagModule } from 'primeng/tag'
import { provideRouter, RouterModule } from '@angular/router'
import { UtilizationBarComponent } from '../utilization-bar/utilization-bar.component'

describe('DashboardComponent', () => {
    let component: DashboardComponent
    let fixture: ComponentFixture<DashboardComponent>
    let dhcpService: DHCPService
    let dataService: ServerDataService
    let settingService: SettingService
    let versionServiceStub: Partial<VersionService>
    let dnsService: DNSService
    let servicesApi: ServicesService

    beforeEach(waitForAsync(() => {
        versionServiceStub = {
            sanitizeSemver: () => '2.0.0',
            getCurrentData: () => of({} as AppsVersions),
            getSoftwareVersionFeedback: () => ({ severity: Severity.success, messages: ['test feedback'] }),
        }

        TestBed.configureTestingModule({
            declarations: [
                DashboardComponent,
                EventsPanelComponent,
                HelpTipComponent,
                SubnetBarComponent,
                HumanCountPipe,
                SurroundPipe,
                EntityLinkComponent,
                VersionStatusComponent,
                UtilizationBarComponent,
            ],
            imports: [
                NoopAnimationsModule,
                PanelModule,
                OverlayPanelModule,
                PaginatorModule,
                TooltipModule,
                ButtonModule,
                RouterModule,
                TableModule,
                TagModule,
                ConfirmDialogModule,
            ],
            providers: [
                ServicesService,
                LocationStrategy,
                DHCPService,
                MessageService,
                UsersService,
                SettingsService,
                { provide: LocationStrategy, useClass: PathLocationStrategy },
                { provide: ServerSentEventsService, useClass: ServerSentEventsTestingService },
                { provide: VersionService, useValue: versionServiceStub },
                provideHttpClient(withInterceptorsFromDi()),
                provideHttpClientTesting(),
                provideRouter([]),
            ],
        })

        dhcpService = TestBed.inject(DHCPService)
        dataService = TestBed.inject(ServerDataService)
        settingService = TestBed.inject(SettingService)
        TestBed.inject(VersionService)
        dnsService = TestBed.inject(DNSService)
        servicesApi = TestBed.inject(ServicesService)
    }))

    beforeEach(() => {
        const fakeOverview: DhcpOverview = {
            dhcp4Stats: {
                assignedAddresses: '6553',
                declinedAddresses: '100',
                totalAddresses: '65530',
            },
            dhcp6Stats: {
                assignedNAs: '20',
                assignedPDs: '1',
                declinedNAs: '10',
                totalNAs: '100',
                totalPDs: '2',
            },
            dhcpDaemons: [
                {
                    active: true,
                    agentCommErrors: 6,
                    appId: 27,
                    appName: 'kea@localhost',
                    appVersion: '2.0.0',
                    haOverview: [],
                    machine: 'pc',
                    machineId: 15,
                    monitored: true,
                    name: 'dhcp4',
                    uptime: 3652,
                    rps1: 1.5212,
                    rps2: 0.3458,
                },
            ],
            sharedNetworks4: {
                items: [
                    {
                        id: 5,
                        addrUtilization: 40,
                        name: 'frog',
                        subnets: [],
                    },
                ],
            },
            sharedNetworks6: {
                items: [
                    {
                        id: 6,
                        addrUtilization: 50,
                        name: 'mouse',
                        subnets: [],
                    },
                ],
            },
            subnets4: {
                items: [
                    {
                        clientClass: 'class-00-00',
                        id: 5,
                        localSubnets: [
                            {
                                appId: 27,
                                appName: 'kea@localhost',
                                id: 41,
                                machineAddress: 'localhost',
                                machineHostname: 'pc',
                                pools: [
                                    {
                                        pool: '1.0.0.4-1.0.255.254',
                                    },
                                ],
                            },
                        ],
                        subnet: '1.0.0.0/16',
                        stats: {
                            'total-addresses': '65530',
                            'assigned-addresses': '6553',
                            'declined-addresses': '100',
                        },
                        statsCollectedAt: '2022-01-19T12:10:22.513Z',
                        addrUtilization: 10.000000000000002,
                    },
                ],
                total: 10496,
            },
            subnets6: {
                items: [
                    {
                        clientClass: 'class-00-00',
                        id: 6,
                        localSubnets: [
                            {
                                appId: 27,
                                appName: 'kea@localhost',
                                machineAddress: 'localhost',
                                machineHostname: 'pc',
                                pools: [
                                    {
                                        pool: '10.3::1-10.3::100',
                                    },
                                ],
                            },
                        ],
                        stats: {
                            'total-nas': '100',
                            'assigned-nas': '20',
                            'declined-nas': '10',
                            'total-pds': '2',
                            'assigned-pds': '1',
                        },
                        statsCollectedAt: '2022-01-19T12:10:22.513Z',
                        subnet: '10:3::/64',
                        addrUtilization: 20,
                    },
                ],
            },
        }

        spyOn(dhcpService, 'getDhcpOverview').and.returnValues(of({} as any), of(fakeOverview as any))
        spyOn(dataService, 'getAppsStats').and.returnValue(
            of({
                keaAppsTotal: 1,
                bind9AppsNotOk: 0,
                bind9AppsTotal: 1,
                keaAppsNotOk: 0,
            } as AppsStats)
        )

        const zonesFetchStatusResponse = {
            items: [
                {
                    appId: 11,
                    appName: 'bind9@agent-bind9',
                    builtinZoneCount: 104,
                    createdAt: '2025-04-14T13:09:06.460Z',
                    daemonId: 27,
                    distinctZoneCount: 106,
                    status: 'ok',
                    zoneCount: 206,
                },
            ],
            total: 1,
        }
        spyOn(dnsService, 'getZonesFetch').and.returnValue(of(zonesFetchStatusResponse as any))
        const appsResponse = {
            items: [
                {
                    accessPoints: [
                        { address: '127.0.0.1', port: 953, type: 'control' },
                        { address: '127.0.0.1', port: 8053, type: 'statistics' },
                    ],
                    details: {
                        daemons: [],
                        daemon: {
                            active: true,
                            agentCommErrors: 200,
                            autoZoneCount: 200,
                            id: 27,
                            monitored: true,
                            name: 'named',
                            reloadedAt: '2025-04-12T11:58:41.000Z',
                            uptime: 5356,
                            version: 'BIND 9.18.35 (Extended Support Version) <id:f506f80>',
                            views: [
                                { name: 'guest', queryHits: 0, queryMisses: 0 },
                                { name: 'trusted', queryHits: 0, queryMisses: 0 },
                            ],
                            zoneCount: 4,
                        },
                    },
                    id: 11,
                    machine: { address: 'agent-bind9', hostname: 'agent-bind9', id: 15 },
                    name: 'bind9@agent-bind9',
                    type: 'bind9',
                    version: 'BIND 9.18.35 (Extended Support Version) <id:f506f80>',
                },
            ],
            total: 1,
        }
        spyOn(servicesApi, 'getApps').and.returnValue(of(appsResponse as any))

        fixture = TestBed.createComponent(DashboardComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should fetch grafana url and dashboard IDs', fakeAsync(() => {
        spyOn(settingService, 'getSettings').and.returnValue(
            of({
                grafanaUrl: 'http://localhost:3000',
                grafanaDhcp4DashboardId: 'dhcp4-dashboard-id',
                grafanaDhcp6DashboardId: 'dhcp6-dashboard-id',
            } as any)
        )

        component.ngOnInit()
        tick()
        expect(component.grafanaUrl).toBe('http://localhost:3000')
        expect(component.grafanaDhcp4DashboardId).toBe('dhcp4-dashboard-id')
        expect(component.grafanaDhcp6DashboardId).toBe('dhcp6-dashboard-id')

        fixture.detectChanges()
        const grafanaIcons = fixture.debugElement.queryAll(By.css('i.pi-chart-line'))
        expect(grafanaIcons?.length).toBe(2)
    }))

    it('should indicate that HA is not enabled', () => {
        // This test doesn't check that the state is rendered correctly
        // as HTML, because the table listing daemons is dynamic and
        // finding the right table cell is going to be involved. Instead
        // we test it indirectly by making sure that the functions used
        // to render the content return expected values.
        const daemon = {
            haOverview: [
                {
                    haState: 'load-balancing',
                    haFailureAt: '2014-06-01T12:00:00Z',
                },
            ],
        }
        expect(component.showHAState(daemon)).toBe('not configured')
        expect(component.showHAFailureTime(daemon)).toBe('')
        expect(component.haStateIcon(daemon)).toBe('ban')

        const daemon2 = {
            haEnabled: false,
            haOverview: [
                {
                    haState: 'load-balancing',
                    haFailureAt: '2014-06-01T12:00:00Z',
                },
            ],
        }
        expect(component.showHAState(daemon2)).toBe('not configured')
        expect(component.showHAFailureTime(daemon2)).toBe('')
        expect(component.haStateIcon(daemon2)).toBe('ban')

        const daemon3 = {
            haEnabled: true,
            haOverview: [
                {
                    haState: '',
                    haFailureAt: null,
                },
            ],
        }
        expect(component.showHAState(daemon3)).toBe('fetching...')
        expect(component.showHAFailureTime(daemon3)).toBe('')
        expect(component.haStateIcon(daemon3)).toBe('spin pi-spinner')

        const daemon4 = {
            haEnabled: true,
            haOverview: [{}],
        }
        expect(component.showHAState(daemon4)).toBe('fetching...')
        expect(component.showHAFailureTime(daemon4)).toBe('')
        expect(component.haStateIcon(daemon4)).toBe('spin pi-spinner')

        const daemon5 = {
            haEnabled: true,
            haOverview: [
                {
                    haState: null,
                    haFailureAt: null,
                },
            ],
        }
        expect(component.showHAState(daemon5)).toBe('fetching...')
        expect(component.showHAFailureTime(daemon5)).toBe('')
        expect(component.haStateIcon(daemon5)).toBe('spin pi-spinner')
    })

    it('should display HA time or placeholder', () => {
        let daemon = {
            haEnabled: true,
            haOverview: [
                {
                    haState: 'load-balancing',
                    haFailureAt: null,
                },
            ],
        }
        expect(component.showHAFailureTime(daemon)).toBe('never')

        daemon = {
            haEnabled: true,
            haOverview: [
                {
                    haState: 'load-balancing',
                    haFailureAt: '2014-06-01T12:00:00Z',
                },
            ],
        }
        expect(component.showHAFailureTime(daemon)).not.toBe('never')
        expect(component.showHAFailureTime(daemon)).not.toBe('')
    })

    it('should select the most important state to display', () => {
        // The second state is partner-down, so it is more important and
        // it should be shown.
        const daemon = {
            haEnabled: true,
            haOverview: [
                {
                    haState: 'load-balancing',
                    haFailureAt: '2014-06-01T12:00:00Z',
                },
                {
                    haState: 'partner-down',
                    haFailureAt: '2014-06-02T12:00:00Z',
                },
            ],
        }
        expect(component.showHAState(daemon)).toBe('partner-down')
        expect(component.showHAFailureTime(daemon)).not.toBe('')
        expect(component.haStateIcon(daemon)).toBe('exclamation-triangle')

        // Swap the states. Still the partner-down state should be shown.
        const daemon1 = {
            haEnabled: true,
            haOverview: [
                {
                    haState: 'partner-down',
                    haFailureAt: '2014-06-02T12:00:00Z',
                },
                {
                    haState: 'load-balancing',
                    haFailureAt: '2014-06-01T12:00:00Z',
                },
            ],
        }
        expect(component.showHAState(daemon1)).toBe('partner-down')
        expect(component.showHAFailureTime(daemon1)).not.toBe('')
        expect(component.haStateIcon(daemon1)).toBe('exclamation-triangle')

        const daemon2 = {
            haEnabled: true,
            haOverview: [
                {
                    haState: 'partner-down',
                    haFailureAt: '2014-06-02T12:00:00Z',
                },
                {
                    haState: 'unavailable',
                    haFailureAt: null,
                },
            ],
        }
        expect(component.showHAState(daemon2)).toBe('unavailable')
        expect(component.showHAFailureTime(daemon2)).toBe('never')
        expect(component.haStateIcon(daemon2)).toBe('times')
    })

    it('should parse integer statistics', async () => {
        await component.refreshDhcpOverview()
        expect(component.overview.subnets4.items[0].stats['total-addresses']).toBe(BigInt(65530))
        expect(component.overview.subnets6.items[0].stats['assigned-nas']).toBe(BigInt(20))
    })

    it('should display utilizations', async () => {
        await component.refreshDhcpOverview()
        fixture.detectChanges()
        await fixture.whenRenderingDone()

        // DHCPv4
        let rows = fixture.debugElement.queryAll(
            By.css('#dashboard-dhcp4 .dashboard-dhcp__subnets .dashboard-section__data .utilization-row')
        )
        expect(rows.length).toBe(1)
        let row = rows[0]
        let cell = row.query(By.css('.utilization-row__value'))
        expect(cell).not.toBeNull()
        let utilization = (cell.nativeElement as HTMLElement).textContent
        expect(utilization.trim()).toBe('10% used')

        // DHCPv6
        rows = fixture.debugElement.queryAll(
            By.css('#dashboard-dhcp6 .dashboard-dhcp__shared-networks .dashboard-section__data .utilization-row')
        )
        expect(rows.length).toBe(1)
        row = rows[0]
        cell = row.query(By.css('.utilization-row__value'))
        expect(cell).not.toBeNull()
        utilization = (cell.nativeElement as HTMLElement).textContent
        expect(utilization.trim()).toBe('50% used')
    })

    it('should display global statistics', async () => {
        await component.refreshDhcpOverview()
        fixture.detectChanges()
        await fixture.whenRenderingDone()

        const rows = fixture.debugElement.queryAll(
            By.css('.dashboard-dhcp__globals .dashboard-section__data .statistics-row')
        )
        const expected = [
            ['Addresses', '6.6k / 65.5k (10% used)'],
            ['Declined', '100'],
            ['Addresses', '20 / 100 (20% used)'],
            ['Prefixes', '1 / 2 (50% used)'],
            ['Declined', '10'],
        ]

        expect(rows.length).toBe(expected.length)

        for (let i = 0; i < expected.length; i++) {
            const [expectedLabel, expectedValue] = expected[i]
            const rowElement = rows[i].nativeElement as HTMLElement
            const labelElement = rowElement.querySelector('.statistics-row__label')
            const valueElement = rowElement.querySelector('.statistics-row__value')
            const labelText = labelElement.textContent.trim()
            const valueText = valueElement.textContent.trim()
            expect(labelText).toBe(expectedLabel)
            expect(valueText).toBe(expectedValue)
        }
    })

    it('should display Kea subnet ID', async () => {
        await component.refreshDhcpOverview()
        fixture.detectChanges()
        await fixture.whenRenderingDone()

        const cells = fixture.debugElement.queryAll(By.css('.dashboard-dhcp__subnets .utilization-row__id'))
        expect(cells.length).toBe(2)
        const values = cells.map((c) => (c.nativeElement as HTMLElement).textContent.trim())
        expect(values).toContain('[41]')
        expect(values).toContain('')
    })

    it('should display rps statistics', async () => {
        await component.refreshDhcpOverview()
        fixture.detectChanges()
        await fixture.whenRenderingDone()

        const table = fixture.debugElement.query(By.css('p-table'))
        expect(table).toBeTruthy()

        const rows = table.queryAll(By.css('tr'))
        expect(rows.length).toBe(2)

        expect(rows[1].nativeElement.innerText).toContain('1.52')
        expect(rows[1].nativeElement.innerText).toContain('0.35')
    })

    it('should display version status component', async () => {
        await component.refreshDhcpOverview()
        fixture.detectChanges()
        await fixture.whenRenderingDone()

        // One VersionStatus for Kea dhcp4 daemon and one for Bind9 daemon.
        const versionStatus = fixture.debugElement.queryAll(By.directive(VersionStatusComponent))
        expect(versionStatus).toBeTruthy()
        expect(versionStatus.length).toEqual(2)
        // Stubbed success icon for kea 2.0.0 is expected.
        expect(versionStatus[0].properties.outerHTML).toContain('2.0.0')
        expect(versionStatus[0].properties.outerHTML).toContain('kea')
        expect(versionStatus[0].properties.outerHTML).toContain('text-green-500')
        expect(versionStatus[0].properties.outerHTML).toContain('test feedback')
    })

    it('should return whether both dhcp and dns apps exist', () => {
        component.appsStats.keaAppsTotal = 0
        component.appsStats.bind9AppsTotal = 0
        expect(component.appsStats.keaAppsTotal).toBe(0)
        expect(component.appsStats.bind9AppsTotal).toBe(0)
        expect(component.bothDHCPAndDNSAppsExist)
            .withContext('in the beginning there are no dhcp nor dns apps')
            .toBeFalse()
        component.appsStats.bind9AppsTotal = 2
        expect(component.bothDHCPAndDNSAppsExist).withContext('only dns apps exist').toBeFalse()
        component.appsStats.bind9AppsTotal = 0
        component.appsStats.keaAppsTotal = 2
        expect(component.bothDHCPAndDNSAppsExist).withContext('only dhcp apps exist').toBeFalse()
        component.appsStats.bind9AppsTotal = 2
        expect(component.bothDHCPAndDNSAppsExist).withContext('both dhcp and dns apps exist').toBeTrue()
    })

    it('should return that both dhcp and dns dashboards are hidden', () => {
        localStorage.clear()
        expect(component.appsStats.keaAppsTotal).toBe(1)
        expect(component.appsStats.bind9AppsTotal).toBe(1)
        expect(localStorage.getItem('dns-dashboard-hidden'))
            .withContext('there should be no state stored in local storage yet')
            .toBeNull()
        expect(localStorage.getItem('dhcp-dashboard-hidden'))
            .withContext('there should be no state stored in local storage yet')
            .toBeNull()
        expect(component.isDNSDashboardHidden()).toBeTrue()
        expect(component.isDHCPDashboardHidden()).toBeTrue()
    })

    it('should return that one dashboard is not hidden', () => {
        localStorage.clear()
        component.appsStats.keaAppsTotal = 2
        component.appsStats.bind9AppsTotal = 0
        expect(localStorage.getItem('dns-dashboard-hidden'))
            .withContext('there should be no state stored in local storage yet')
            .toBeNull()
        expect(localStorage.getItem('dhcp-dashboard-hidden'))
            .withContext('there should be no state stored in local storage yet')
            .toBeNull()
        expect(component.isDHCPDashboardHidden()).toBeFalse()
        fixture.detectChanges()
        let expandedDashboardPanels = fixture.debugElement.queryAll(
            By.css('.p-panel.p-panel-toggleable.p-panel-expanded')
        )
        expect(expandedDashboardPanels).toBeTruthy()
        expect(expandedDashboardPanels.length).toBe(1)
        expect(expandedDashboardPanels[0].nativeElement.innerText).toContain('DHCP Dashboard')

        component.appsStats.keaAppsTotal = 0
        component.appsStats.bind9AppsTotal = 2
        expect(component.isDNSDashboardHidden()).toBeFalse()
        fixture.detectChanges()
        expandedDashboardPanels = fixture.debugElement.queryAll(By.css('.p-panel.p-panel-toggleable.p-panel-expanded'))
        expect(expandedDashboardPanels).toBeTruthy()
        expect(expandedDashboardPanels.length).toBe(1)
        expect(expandedDashboardPanels[0].nativeElement.innerText).toContain('DNS Dashboard')
    })

    it('should store dashboard hidden state', () => {
        localStorage.clear()
        component.storeDHCPDashboardHidden(true)
        expect(localStorage.getItem('dhcp-dashboard-hidden')).toBe('true')
        expect(localStorage.getItem('dns-dashboard-hidden')).toBeNull()
        component.storeDNSDashboardHidden(true)
        expect(localStorage.getItem('dns-dashboard-hidden')).toBe('true')
        component.storeDHCPDashboardHidden(false)
        expect(localStorage.getItem('dhcp-dashboard-hidden')).toBe('false')
        component.storeDNSDashboardHidden(false)
        expect(localStorage.getItem('dns-dashboard-hidden')).toBe('false')
    })

    it('should display dns dashboard', () => {
        const dashboardPanels = fixture.debugElement.queryAll(By.css('.p-panel.p-panel-toggleable'))
        expect(dashboardPanels).toBeTruthy()
        expect(dashboardPanels.length).toBe(2)
        expect(dashboardPanels[1].nativeElement.innerText).toContain('DNS Dashboard')
    })
})
