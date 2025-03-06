import { ComponentFixture, TestBed, fakeAsync, waitForAsync, tick } from '@angular/core/testing'
import { HaStatusComponent } from './ha-status.component'
import { PanelModule } from 'primeng/panel'
import { TooltipModule } from 'primeng/tooltip'
import { MessageModule } from 'primeng/message'
import { LocaltimePipe } from '../pipes/localtime.pipe'
import { ServicesService, ServicesStatus } from '../backend'
import { HttpClientTestingModule } from '@angular/common/http/testing'
import { ProgressSpinner, ProgressSpinnerModule } from 'primeng/progressspinner'
import { of, throwError } from 'rxjs'
import { HttpErrorResponse, HttpEvent } from '@angular/common/http'
import { By } from '@angular/platform-browser'
import { MessageService } from 'primeng/api'
import { EntityLinkComponent } from '../entity-link/entity-link.component'
import { TableModule } from 'primeng/table'
import { ButtonModule } from 'primeng/button'
import { RouterModule } from '@angular/router'

describe('HaStatusComponent', () => {
    let component: HaStatusComponent
    let fixture: ComponentFixture<HaStatusComponent>
    let servicesApi: ServicesService

    beforeEach(waitForAsync(() => {
        TestBed.configureTestingModule({
            imports: [
                ButtonModule,
                HttpClientTestingModule,
                MessageModule,
                PanelModule,
                ProgressSpinnerModule,
                RouterModule.forRoot([]),
                TableModule,
                TooltipModule,
            ],
            declarations: [EntityLinkComponent, HaStatusComponent, LocaltimePipe],
            providers: [ServicesService, MessageService],
        }).compileComponents()

        servicesApi = TestBed.inject(ServicesService)
    }))

    beforeEach(() => {
        fixture = TestBed.createComponent(HaStatusComponent)
        component = fixture.componentInstance
        component.appId = 4
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should present a waiting indicator during initial loading', fakeAsync(() => {
        // Mock the API response.
        spyOn(servicesApi, 'getAppServicesStatus').and.returnValue(
            of({
                items: [],
            } as ServicesStatus & HttpEvent<ServicesStatus>)
        )
        spyOn(component, 'setCountdownTimer')

        tick()
        fixture.detectChanges()

        expect(component.loadedOnce).toBeFalse()
        let spinner = fixture.debugElement.query(By.directive(ProgressSpinner))
        expect(spinner).not.toBeNull()

        // Execute ngOnInit hook.
        fixture.detectChanges()

        // Check if the component is in the loading state.
        expect(component.loadedOnce).toBeFalse()

        // Check if the waiting indicator is presented.
        spinner = fixture.debugElement.query(By.directive(ProgressSpinner))
        expect(spinner).not.toBeNull()
    }))

    it('should present a placeholder when loaded data contain no statuses', fakeAsync(() => {
        // Mock the API response.
        spyOn(servicesApi, 'getAppServicesStatus').and.returnValue(
            of({
                items: [],
            } as ServicesStatus & HttpEvent<ServicesStatus>)
        )
        spyOn(component, 'setCountdownTimer')

        // Execute ngOnInit hook.
        fixture.detectChanges()

        // Continue the API response processing.
        tick()

        // Check if the initial data loading is done.
        expect(component.loadedOnce).toBeTrue()
        // Render the updated data.
        fixture.detectChanges()

        // Check if there is no waiting indicator.
        const spinner = fixture.debugElement.query(By.directive(ProgressSpinner))
        expect(spinner).toBeNull()

        // Check if there is the empty data placeholder.
        expect(fixture.debugElement.nativeElement.textContent).toContain(
            'High Availability is not enabled on this server.'
        )
    }))

    it('should not present a placeholder on the initial data loading failure', fakeAsync(() => {
        // Mock the API response.
        spyOn(servicesApi, 'getAppServicesStatus').and.returnValue(throwError(new HttpErrorResponse({ status: 500 })))
        spyOn(component, 'setCountdownTimer')

        // Execute ngOnInit hook.
        fixture.detectChanges()

        // Continue the API response processing.
        tick()

        // Check if the data aren't marked as loaded.
        expect(component.loadedOnce).toBeFalse()
        // Render the updated data.
        fixture.detectChanges()

        // Check if there still is a waiting indicator.
        const spinner = fixture.debugElement.query(By.directive(ProgressSpinner))
        expect(spinner).not.toBeNull()

        // Check if there isn't the empty data placeholder.
        expect(fixture.debugElement.nativeElement.textContent).not.toContain(
            'High Availability is not enabled on this server.'
        )
    }))

    it('should present a hub and spoke configuration state', fakeAsync(() => {
        // Mock the API response.
        spyOn(servicesApi, 'getAppServicesStatus').and.returnValue(
            of({
                items: [
                    {
                        status: {
                            daemon: 'dhcp4',
                            haServers: {
                                relationship: 'server1',
                                primaryServer: {
                                    age: 0,
                                    appId: 234,
                                    controlAddress: '192.0.2.1:8080',
                                    failoverTime: null,
                                    id: 1,
                                    inTouch: true,
                                    role: 'primary',
                                    scopes: ['server1'],
                                    state: 'hot-standby',
                                    statusTime: '2024-02-16 13:54:23',
                                    commInterrupted: 0,
                                    connectingClients: 0,
                                    unackedClients: 0,
                                    unackedClientsLeft: 0,
                                    analyzedPackets: 0,
                                },
                                secondaryServer: {
                                    age: 0,
                                    appId: 123,
                                    controlAddress: '192.0.2.2:8080',
                                    failoverTime: null,
                                    id: 1,
                                    inTouch: true,
                                    role: 'standby',
                                    scopes: [],
                                    state: 'hot-standby',
                                    statusTime: '2024-02-16 12:01:02',
                                    commInterrupted: 0,
                                    connectingClients: 0,
                                    unackedClients: 0,
                                    unackedClientsLeft: 0,
                                    analyzedPackets: 0,
                                },
                            },
                        },
                    },
                    {
                        status: {
                            daemon: 'dhcp4',
                            haServers: {
                                relationship: 'server3',
                                primaryServer: {
                                    age: 0,
                                    appId: 345,
                                    controlAddress: '192.0.2.3:8080',
                                    failoverTime: null,
                                    id: 1,
                                    inTouch: true,
                                    role: 'primary',
                                    scopes: ['server3'],
                                    state: 'hot-standby',
                                    statusTime: '2024-02-16 13:54:23',
                                    commInterrupted: 0,
                                    connectingClients: 0,
                                    unackedClients: 0,
                                    unackedClientsLeft: 0,
                                    analyzedPackets: 0,
                                },
                                secondaryServer: {
                                    age: 0,
                                    appId: 123,
                                    controlAddress: '192.0.2.2:8081',
                                    failoverTime: '2024-02-16 11:11:12',
                                    id: 1,
                                    inTouch: true,
                                    role: 'standby',
                                    scopes: [],
                                    state: 'hot-standby',
                                    statusTime: '2024-02-16 12:01:02',
                                    commInterrupted: 1,
                                    connectingClients: 5,
                                    unackedClients: 1,
                                    unackedClientsLeft: 4,
                                    analyzedPackets: 2,
                                },
                            },
                        },
                    },
                ],
            } as ServicesStatus & HttpEvent<ServicesStatus>)
        )
        spyOn(component, 'setCountdownTimer')

        component.appId = 123
        component.daemonName = 'dhcp4'
        component.ngOnInit()
        tick()
        fixture.detectChanges()

        expect(servicesApi.getAppServicesStatus).toHaveBeenCalled()
        expect(component.setCountdownTimer).toHaveBeenCalled()

        expect(component.status.length).toBe(24)

        for (let i = 0; i < 12; i++) {
            expect(component.status[i].relationship.name).toBe('Relationship #1')
            expect(component.status[i].relationship.cells).toBeTruthy()
            expect(component.status[i].relationship.cells.length).toBe(2)
            expect(component.status[i].relationship.cells[0].appId).toBeFalsy()
            expect(component.status[i].relationship.cells[0].appName).toBeFalsy()
            expect(component.status[i].relationship.cells[0].iconType).toBeFalsy()
            expect(component.status[i].relationship.cells[0].progress).toBeFalsy()
            expect(component.status[i].relationship.cells[0].value).toBe('standby')
            expect(component.status[i].relationship.cells[1].appId).toBe(234)
            expect(component.status[i].relationship.cells[1].appName).toBe('Kea@192.0.2.1:8080')
            expect(component.status[i].relationship.cells[1].iconType).toBeFalsy()
            expect(component.status[i].relationship.cells[1].progress).toBeFalsy()
            expect(component.status[i].relationship.cells[1].value).toBe('primary')
        }

        for (let i = 12; i < 24; i++) {
            expect(component.status[i].relationship.name).toBe('Relationship #2')
            expect(component.status[i].relationship.cells).toBeTruthy()
            expect(component.status[i].relationship.cells.length).toBe(2)
            expect(component.status[i].relationship.cells.length).toBe(2)
            expect(component.status[i].relationship.cells[0].appId).toBeFalsy()
            expect(component.status[i].relationship.cells[0].appName).toBeFalsy()
            expect(component.status[i].relationship.cells[0].iconType).toBe('error')
            expect(component.status[i].relationship.cells[0].progress).toBeFalsy()
            expect(component.status[i].relationship.cells[0].value).toBe('standby')
            expect(component.status[i].relationship.cells[1].appId).toBe(345)
            expect(component.status[i].relationship.cells[1].appName).toBe('Kea@192.0.2.3:8080')
            expect(component.status[i].relationship.cells[1].iconType).toBeFalsy()
            expect(component.status[i].relationship.cells[1].progress).toBeFalsy()
            expect(component.status[i].relationship.cells[1].value).toBe('primary')
        }

        // Verify that each relationship data rows contain expected titles.
        const expectedTitles = [
            'Heartbeat status',
            'Control status',
            'State',
            'Scopes',
            'Status time',
            'Status age',
            'Last in partner-down',
            'Unacked clients',
            'Connecting clients',
            'Analyzed packets',
            'Failover progress',
            'Summary',
        ]
        component.status.forEach((row, index) => {
            expect(row.title).toBe(expectedTitles.at(index % 12))
            expect(row.cells.length).toBe(2)
        })

        expect(component.status[0].cells[0].value).toBe('ok')
        expect(component.status[0].cells[1].value).toBe('ok')
        expect(component.status[1].cells[0].value).toBe('online')
        expect(component.status[1].cells[1].value).toBe('online')
        expect(component.status[2].cells[0].value).toBe('hot-standby')
        expect(component.status[2].cells[1].value).toBe('hot-standby')
        expect(component.status[3].cells[0].value).toBe('none (standby server)')
        expect(component.status[3].cells[1].value).toBe('server1')
        expect(component.status[4].cells[0].value).toBe('2024-02-16 12:01:02')
        expect(component.status[4].cells[1].value).toBe('2024-02-16 13:54:23')
        expect(component.status[5].cells[0].value).toBe('just now')
        expect(component.status[5].cells[1].value).toBe('just now')
        expect(component.status[6].cells[0].value).toBe('never')
        expect(component.status[6].cells[1].value).toBe('never')
        expect(component.status[7].cells[0].value).toBe('n/a')
        expect(component.status[7].cells[1].value).toBe('n/a')
        expect(component.status[8].cells[0].value).toBe('n/a')
        expect(component.status[8].cells[1].value).toBe('n/a')
        expect(component.status[9].cells[0].value).toBe('n/a')
        expect(component.status[9].cells[1].value).toBe('n/a')
        expect(component.status[10].cells[0].value).toBe('n/a')
        expect(component.status[10].cells[1].value).toBe('n/a')
        expect(component.status[11].cells[0].value).toBe('Server is responding to no DHCP traffic.')
        expect(component.status[11].cells[1].value).toBe('Server is responding to all DHCP traffic.')

        expect(component.status[12].cells[0].value).toBe('failed')
        expect(component.status[12].cells[0].iconType).toBe('error')
        expect(component.status[12].cells[1].value).toBe('ok')
        expect(component.status[13].cells[0].value).toBe('online')
        expect(component.status[13].cells[1].value).toBe('online')
        expect(component.status[14].cells[0].value).toBe('hot-standby')
        expect(component.status[14].cells[1].value).toBe('hot-standby')
        expect(component.status[15].cells[0].value).toBe('none (standby server)')
        expect(component.status[15].cells[1].value).toBe('server3')
        expect(component.status[16].cells[0].value).toBe('2024-02-16 12:01:02')
        expect(component.status[16].cells[1].value).toBe('2024-02-16 13:54:23')
        expect(component.status[17].cells[0].value).toBe('just now')
        expect(component.status[17].cells[1].value).toBe('just now')
        expect(component.status[18].cells[0].value).toBe('2024-02-16 11:11:12')
        expect(component.status[18].cells[1].value).toBe('never')
        expect(component.status[19].cells[0].value).toBe('1 of 6')
        expect(component.status[19].cells[1].value).toBe('n/a')
        expect(component.status[20].cells[0].value).toBe(5)
        expect(component.status[20].cells[1].value).toBe('n/a')
        expect(component.status[21].cells[0].value).toBe(2)
        expect(component.status[21].cells[1].value).toBe('n/a')
        expect(component.status[22].cells[0].value).toBeFalsy()
        expect(component.status[22].cells[0].progress).toBe(16)
        expect(component.status[22].cells[1].value).toBe('n/a')
        expect(component.status[23].cells[0].value).toBe('Server has started the failover procedure.')
        expect(component.status[23].cells[1].value).toBe('Server is responding to all DHCP traffic.')
    }))

    it('should present a passive-backup configuration state', fakeAsync(() => {
        // Mock the API response.
        spyOn(servicesApi, 'getAppServicesStatus').and.returnValue(
            of({
                items: [
                    {
                        status: {
                            daemon: 'dhcp4',
                            haServers: {
                                relationship: 'server1',
                                primaryServer: {
                                    age: 7,
                                    appId: 234,
                                    controlAddress: '192.0.2.1:8080',
                                    failoverTime: null,
                                    id: 1,
                                    inTouch: true,
                                    role: 'primary',
                                    scopes: ['server1'],
                                    state: 'passive-backup',
                                    statusTime: '2024-02-16 01:06:57',
                                },
                            },
                        },
                    },
                ],
            } as ServicesStatus & HttpEvent<ServicesStatus>)
        )
        spyOn(component, 'setCountdownTimer')

        component.appId = 234
        component.daemonName = 'dhcp4'
        component.ngOnInit()
        tick()
        fixture.detectChanges()

        expect(servicesApi.getAppServicesStatus).toHaveBeenCalled()
        expect(component.setCountdownTimer).toHaveBeenCalled()

        expect(component.status.length).toBe(5)

        component.status.forEach((row) => {
            expect(row.relationship).toBeTruthy()
            expect(row.relationship.name).toBe('Relationship #1')
            expect(row.relationship.cells).toBeTruthy()
            expect(row.relationship.cells.length).toBe(1)
            expect(row.relationship.cells[0].appId).toBeFalsy()
            expect(row.relationship.cells[0].appName).toBeFalsy()
            expect(row.relationship.cells[0].iconType).toBeFalsy()
            expect(row.relationship.cells[0].progress).toBeFalsy()
            expect(row.relationship.cells[0].value).toBe('primary')
        })

        const expectedTitles = ['Control status', 'State', 'Scopes', 'Status time', 'Status age']

        component.status.forEach((row, index) => {
            expect(row.title).toBe(expectedTitles.at(index))
            expect(row.cells.length).toBe(1)
        })

        expect(component.status[0].cells[0].value).toBe('online')
        expect(component.status[1].cells[0].value).toBe('passive-backup')
        expect(component.status[2].cells[0].value).toBe('server1')
        expect(component.status[3].cells[0].value).toBe('2024-02-16 01:06:57')
        expect(component.status[4].cells[0].value).toBe('7 seconds ago')
    }))

    it('should not present HA state when the appId is not matching', fakeAsync(() => {
        // Mock the API response.
        spyOn(servicesApi, 'getAppServicesStatus').and.returnValue(
            of({
                items: [
                    {
                        status: {
                            daemon: 'dhcp4',
                            haServers: {
                                relationship: 'server1',
                                primaryServer: {
                                    age: 7,
                                    appId: 234,
                                    controlAddress: '192.0.2.1:8080',
                                    failoverTime: null,
                                    id: 1,
                                    inTouch: true,
                                    role: 'primary',
                                    scopes: ['server1'],
                                    state: 'passive-backup',
                                    statusTime: '2024-02-16 01:06:57',
                                },
                            },
                        },
                    },
                ],
            } as ServicesStatus & HttpEvent<ServicesStatus>)
        )
        spyOn(component, 'setCountdownTimer')

        component.appId = 345
        component.daemonName = 'dhcp4'
        component.ngOnInit()
        tick()
        fixture.detectChanges()

        expect(servicesApi.getAppServicesStatus).toHaveBeenCalled()
        expect(component.setCountdownTimer).toHaveBeenCalled()

        expect(component.status.length).toBe(0)
    }))

    it('should not present HA state when the daemon is not matching', fakeAsync(() => {
        // Mock the API response.
        spyOn(servicesApi, 'getAppServicesStatus').and.returnValue(
            of({
                items: [
                    {
                        status: {
                            daemon: 'dhcp4',
                            haServers: {
                                relationship: 'server1',
                                primaryServer: {
                                    age: 7,
                                    appId: 234,
                                    controlAddress: '192.0.2.1:8080',
                                    failoverTime: null,
                                    id: 1,
                                    inTouch: true,
                                    role: 'primary',
                                    scopes: ['server1'],
                                    state: 'passive-backup',
                                    statusTime: '2024-02-16 01:06:57',
                                },
                            },
                        },
                    },
                ],
            } as ServicesStatus & HttpEvent<ServicesStatus>)
        )
        spyOn(component, 'setCountdownTimer')

        component.appId = 234
        component.daemonName = 'dhcp6'
        component.ngOnInit()
        tick()
        fixture.detectChanges()

        expect(servicesApi.getAppServicesStatus).toHaveBeenCalled()
        expect(component.setCountdownTimer).toHaveBeenCalled()

        expect(component.status.length).toBe(0)
    }))

    it('should format age', () => {
        expect(component.formatAge(null)).toBe('just now')
        expect(component.formatAge(0)).toBe('just now')
        expect(component.formatAge(45)).toBe('45 seconds ago')
        expect(component.formatAge(121)).toBe('2 minutes ago')
    })

    it('should format control status', () => {
        expect(component.formatControlStatus({ inTouch: null })).toBe('unknown')
        expect(component.formatControlStatus({ inTouch: true })).toBe('online')
        expect(component.formatControlStatus({ inTouch: false })).toBe('offline')
    })

    it('should format failover number', () => {
        expect(
            component.formatFailoverNumber(
                {
                    unackedClients: 0,
                    unackedClientsLeft: 0,
                },
                3
            )
        ).toBe('n/a')

        expect(
            component.formatFailoverNumber(
                {
                    unackedClients: null,
                    unackedClientsLeft: null,
                },
                3
            )
        ).toBe('n/a')

        expect(
            component.formatFailoverNumber(
                {
                    unackedClients: 2,
                    unackedClientsLeft: 5,
                },
                3
            )
        ).toBe(3)

        expect(
            component.formatFailoverNumber(
                {
                    unackedClients: 2,
                    unackedClientsLeft: 5,
                },
                0
            )
        ).toBe(0)

        expect(
            component.formatFailoverNumber(
                {
                    unackedClients: 2,
                    unackedClientsLeft: 5,
                },
                -1
            )
        ).toBe(0)
    })

    it('should format heartbeat status', () => {
        expect(component.formatHeartbeatStatus({ commInterrupted: -1 })).toBe('unknown')
        expect(component.formatHeartbeatStatus({ commInterrupted: 0 })).toBe('ok')
        expect(component.formatHeartbeatStatus({ commInterrupted: 1 })).toBe('failed')
    })

    it('should format scopes', () => {
        expect(component.formatScopes({})).toBe('none')
        expect(component.formatScopes({ scopes: [] })).toBe('none')
        expect(component.formatScopes({ scopes: ['server1', 'server2'] })).toBe('server1, server2')
        expect(component.formatScopes({ scopes: [], role: 'standby', state: 'hot-standby' })).toBe(
            'none (standby server)'
        )
    })

    it('should format state', () => {
        expect(component.formatState({})).toBe('fetching...')
        expect(component.formatState({ state: '' })).toBe('fetching...')
        expect(component.formatState({ state: 'waiting' })).toBe('waiting')
    })

    it('should format unacked clients', () => {
        expect(component.formatUnackedClients({})).toBe('n/a')
        expect(component.formatUnackedClients({ commInterrupted: 2 })).toBe('n/a')
        expect(component.formatUnackedClients({ commInterrupted: 2, unackedClients: 2, unackedClientsLeft: 3 })).toBe(
            '2 of 6'
        )
    })

    it('should create summary', () => {
        expect(component.createSummary({ unackedClients: 2, unackedClientsLeft: 3 }, {})).toBe(
            'Server has started the failover procedure.'
        )
        expect(component.createSummary({ scopes: [] }, {})).toBe('Server is responding to no DHCP traffic.')
        expect(component.createSummary({ scopes: ['server1'] }, { scopes: [] })).toBe(
            'Server is responding to all DHCP traffic.'
        )
        expect(component.createSummary({ scopes: ['server1'] }, { scopes: ['server2'] })).toBe(
            'Server is responding to DHCP traffic.'
        )
    })

    it('should calculate failover progress', () => {
        expect(component.calculateServerFailoverProgress({})).toBe(-1)
        expect(component.calculateServerFailoverProgress({ unackedClientsLeft: 5 })).toBe(0)
        expect(component.calculateServerFailoverProgress({ unackedClients: 3, unackedClientsLeft: 5 })).toBe(33)
    })

    it('should return state icon type', () => {
        expect(component.getStateIconType({})).toBe('pending')
        expect(component.getStateIconType({ state: '' })).toBe('pending')
        expect(component.getStateIconType({ state: 'load-balancing' })).toBe('ok')
        expect(component.getStateIconType({ state: 'hot-standby' })).toBe('ok')
        expect(component.getStateIconType({ state: 'passive-backup' })).toBe('ok')
        expect(component.getStateIconType({ state: 'terminated' })).toBe('warn')
    })
})
