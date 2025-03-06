import { ComponentFixture, TestBed } from '@angular/core/testing'

import { VersionPageComponent } from './version-page.component'
import { HttpClientTestingModule } from '@angular/common/http/testing'
import { MessageService } from 'primeng/api'
import { BreadcrumbModule } from 'primeng/breadcrumb'
import { BreadcrumbsComponent } from '../breadcrumbs/breadcrumbs.component'
import { PanelModule } from 'primeng/panel'
import { TableModule } from 'primeng/table'
import { HelpTipComponent } from '../help-tip/help-tip.component'
import { OverlayPanelModule } from 'primeng/overlaypanel'
import { BrowserAnimationsModule } from '@angular/platform-browser/animations'
import { ButtonModule } from 'primeng/button'
import { RouterModule } from '@angular/router'
import { Severity, VersionAlert, VersionService } from '../version.service'
import { of } from 'rxjs'
import { AppsVersions, ServicesService } from '../backend'
import { MessagesModule } from 'primeng/messages'
import { BadgeModule } from 'primeng/badge'
import { By } from '@angular/platform-browser'

describe('VersionPageComponent', () => {
    let component: VersionPageComponent
    let fixture: ComponentFixture<VersionPageComponent>
    let versionService: VersionService
    let servicesApi: ServicesService
    let getCurrentDataSpy: jasmine.Spy<any>
    let getDataManufactureDateSpy: jasmine.Spy<any>
    let getDataSourceSpy: jasmine.Spy<any>
    let getMachinesAppsVersionsSpy: jasmine.Spy<any>
    let messageService: MessageService
    let messageAddSpy: jasmine.Spy<any>
    const fakeResponse = {
        bind9: {
            currentStable: [
                {
                    eolDate: '2026-07-01',
                    esv: 'true',
                    major: 9,
                    minor: 18,
                    range: '9.18.x',
                    releaseDate: '2024-09-18',
                    status: 'Current Stable',
                    version: '9.18.30',
                },
                {
                    eolDate: '2028-07-01',
                    major: 9,
                    minor: 20,
                    range: '9.20.x',
                    releaseDate: '2024-09-18',
                    status: 'Current Stable',
                    version: '9.20.2',
                },
            ],
            latestDev: { major: 9, minor: 21, releaseDate: '2024-09-18', status: 'Development', version: '9.21.1' },
            sortedStableVersions: ['9.18.30', '9.20.2'],
        },
        date: '2024-10-03',
        kea: {
            currentStable: [
                {
                    eolDate: '2026-07-01',
                    major: 2,
                    minor: 6,
                    range: '2.6.x',
                    releaseDate: '2024-07-31',
                    status: 'Current Stable',
                    version: '2.6.1',
                },
                {
                    eolDate: '2025-07-01',
                    major: 2,
                    minor: 4,
                    range: '2.4.x',
                    releaseDate: '2023-11-29',
                    status: 'Current Stable',
                    version: '2.4.1',
                },
            ],
            latestDev: { major: 2, minor: 7, releaseDate: '2024-09-25', status: 'Development', version: '2.7.3' },
            sortedStableVersions: ['2.4.1', '2.6.1'],
        },
        stork: {
            currentStable: null,
            latestDev: { major: 1, minor: 19, releaseDate: '2024-10-02', status: 'Development', version: '1.19.0' },
            latestSecure: [
                {
                    major: 1,
                    minor: 15,
                    releaseDate: '2024-03-27',
                    status: 'Security update',
                    version: '1.15.1',
                },
            ],
            sortedStableVersions: null,
        },
    }
    const fakeMachinesResponse = {
        items: [
            {
                address: 'agent-kea', // warn
                agentPort: 8888,
                agentVersion: '1.19.0',
                apps: [
                    {
                        accessPoints: null,
                        details: {
                            daemons: [
                                { backends: null, files: null, hooks: null, id: 12, logTargets: null, name: 'd2' },
                                { backends: null, files: null, hooks: null, id: 14, logTargets: null, name: 'dhcp6' },
                                {
                                    active: true,
                                    backends: null,
                                    files: null,
                                    hooks: null,
                                    id: 13,
                                    logTargets: null,
                                    name: 'dhcp4',
                                    version: '2.7.2',
                                },
                                {
                                    active: true,
                                    backends: null,
                                    files: null,
                                    hooks: null,
                                    id: 11,
                                    logTargets: null,
                                    name: 'ca',
                                    version: '2.7.2',
                                },
                            ],
                        },
                        id: 4,
                        name: 'kea@agent-kea',
                        type: 'kea',
                        version: '2.7.2',
                    },
                ],
                hostname: 'agent-kea',
                id: 4,
            },
            {
                address: 'agent-bind9', // success
                agentPort: 8883,
                agentVersion: '1.19.0',
                apps: [
                    {
                        accessPoints: null,
                        details: { daemons: null },
                        id: 9,
                        name: 'bind9@agent-bind9',
                        type: 'bind9',
                        version: 'BIND 9.18.30 (Extended Support Version) <id:cdc8d69>',
                    },
                ],
                hostname: 'agent-bind9',
                id: 9,
            },
            {
                address: 'agent-kea-ha2', // info
                agentPort: 8885,
                agentVersion: '1.19.0',
                apps: [
                    {
                        accessPoints: null,
                        details: {
                            daemons: [
                                { backends: null, files: null, hooks: null, id: 23, logTargets: null, name: 'd2' },
                                { backends: null, files: null, hooks: null, id: 25, logTargets: null, name: 'dhcp6' },
                                {
                                    active: true,
                                    backends: null,
                                    files: null,
                                    hooks: null,
                                    id: 24,
                                    logTargets: null,
                                    name: 'dhcp4',
                                    version: '2.6.0',
                                },
                                {
                                    active: true,
                                    backends: null,
                                    files: null,
                                    hooks: null,
                                    id: 26,
                                    logTargets: null,
                                    name: 'ca',
                                    version: '2.6.0',
                                },
                            ],
                        },
                        id: 7,
                        name: 'kea@agent-kea-ha2',
                        type: 'kea',
                        version: '2.6.0',
                    },
                ],
                hostname: 'agent-kea-ha2',
                id: 7,
            },
            {
                address: 'agent-kea6', // err
                agentPort: 8887,
                agentVersion: '1.19.0',
                apps: [
                    {
                        accessPoints: null,
                        details: {
                            daemons: [
                                {
                                    active: true,
                                    backends: null,
                                    files: null,
                                    hooks: null,
                                    id: 2,
                                    logTargets: null,
                                    name: 'dhcp6',
                                    version: '2.7.0',
                                },
                                {
                                    active: true,
                                    backends: null,
                                    files: null,
                                    hooks: null,
                                    id: 1,
                                    logTargets: null,
                                    name: 'ca',
                                    version: '2.7.1',
                                },
                            ],
                            mismatchingDaemons: true,
                        },
                        id: 1,
                        name: 'kea@agent-kea6',
                        type: 'kea',
                        version: '2.7.0',
                    },
                ],
                hostname: 'agent-kea6',
                id: 1,
            },
        ],
        total: 4,
    }

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            imports: [
                HttpClientTestingModule,
                PanelModule,
                TableModule,
                BreadcrumbModule,
                OverlayPanelModule,
                BrowserAnimationsModule,
                ButtonModule,
                RouterModule.forRoot([
                    {
                        path: 'versions',
                        component: VersionPageComponent,
                    },
                ]),
                MessagesModule,
                BadgeModule,
            ],
            declarations: [VersionPageComponent, BreadcrumbsComponent, HelpTipComponent],
            providers: [MessageService],
        }).compileComponents()
        fixture = TestBed.createComponent(VersionPageComponent)
        versionService = TestBed.inject(VersionService)
        servicesApi = TestBed.inject(ServicesService)
        messageService = TestBed.inject(MessageService)
        component = fixture.componentInstance
        getCurrentDataSpy = spyOn(versionService, 'getCurrentData')
        getDataManufactureDateSpy = spyOn(versionService, 'getDataManufactureDate').and.returnValue(of('2024-10-03'))
        getDataSourceSpy = spyOn(versionService, 'getDataSource').and.returnValue(
            of(AppsVersions.DataSourceEnum.Offline)
        )
        spyOn(versionService, 'getStorkServerUpdateNotification').and.returnValue(
            of({
                available: true,
                feedback: {
                    severity: Severity.warn,
                    messages: ['Stork server update is available (1.19.0).'],
                },
            })
        )
        getMachinesAppsVersionsSpy = spyOn(servicesApi, 'getMachinesAppsVersions')
        messageAddSpy = spyOn(messageService, 'add').and.callThrough()
    })

    /**
     * Helper function setting mocks of normal version service responses.
     */
    function apisWorkingFine() {
        getCurrentDataSpy.and.returnValue(of(fakeResponse))
        getMachinesAppsVersionsSpy.and.returnValue(of(fakeMachinesResponse as any))
        fixture.detectChanges()
    }

    /**
     * Helper function setting mocks of malfunctioning version service responses.
     */
    function apisThrowingErrors(first = true) {
        if (first) {
            getCurrentDataSpy.and.throwError('version service error1')
            getMachinesAppsVersionsSpy.and.returnValue(of(fakeMachinesResponse as any))
        } else {
            getCurrentDataSpy.and.returnValue(of(fakeResponse))
            getMachinesAppsVersionsSpy.and.throwError('version service error2')
        }

        fixture.detectChanges()
    }

    it('should create', () => {
        apisWorkingFine()
        expect(component).toBeTruthy()
    })

    it('should get daemons versions', () => {
        // Arrange
        apisWorkingFine()
        const app = fakeMachinesResponse.items.filter((m) => m.address === 'agent-kea')[0].apps[0]

        // Act & Assert
        expect(component.getDaemonsVersions(app)).toEqual('dhcp4 2.7.2, ca 2.7.2')
    })

    it('should display offline data info message', () => {
        // Arrange & Act & Assert
        apisWorkingFine()
        expect(getDataManufactureDateSpy).toHaveBeenCalledTimes(1)
        expect(getDataSourceSpy).toHaveBeenCalledTimes(1)
        expect(getCurrentDataSpy).toHaveBeenCalledTimes(1)
        expect(getMachinesAppsVersionsSpy).toHaveBeenCalledTimes(1)

        const de = fixture.debugElement.query(By.css('.header-message .p-messages .p-message-info'))
        expect(de).toBeTruthy()
        expect(de.nativeElement.innerText).toContain(
            'The information below about ISC software versions relies on an' +
                ' offline built-in JSON file that was generated on 2024-10-03.' +
                ' To see up-to-date information, please visit the ISC website'
        )
    })

    it('should display summary table', () => {
        // Arrange & Act & Assert
        apisWorkingFine()
        expect(component.machines.length).toEqual(4)

        // There should be 4 tables.
        const tablesDe = fixture.debugElement.queryAll(By.css('table.p-datatable-table'))
        expect(tablesDe.length).toEqual(4)
        const summaryTableDe = tablesDe[0]

        // There should be 4 group headers, one per error, warn, info and success severity.
        expect(summaryTableDe.queryAll(By.css('tbody tr')).length).toEqual(4)
        expect(component.counters).toEqual([1, 1, 1, 0, 1])
        const groupHeaderMessagesDe = summaryTableDe.queryAll(By.css('.p-message'))
        expect(groupHeaderMessagesDe.length).toEqual(4)
        expect(Object.keys(groupHeaderMessagesDe[0].classes)).toContain('p-message-error')
        expect(Object.keys(groupHeaderMessagesDe[1].classes)).toContain('p-message-warn')
        expect(Object.keys(groupHeaderMessagesDe[2].classes)).toContain('p-message-info')
        expect(Object.keys(groupHeaderMessagesDe[3].classes)).toContain('p-message-success')
    })

    it('should display kea releases table', () => {
        // Arrange & Act & Assert
        apisWorkingFine()
        expect(component.machines.length).toEqual(4)

        // There should be 4 tables.
        const tablesDe = fixture.debugElement.queryAll(By.css('table.p-datatable-table'))
        expect(tablesDe.length).toEqual(4)
        const keaTable = tablesDe[1]

        // There should be 2 rows for stable releases and 1 for development.
        expect(keaTable.queryAll(By.css('tbody tr')).length).toEqual(3)
        expect(keaTable.nativeElement.innerText).toContain('Current Stable')
        expect(keaTable.nativeElement.innerText).toContain('Development')
        expect(keaTable.nativeElement.innerText).toContain('Kea ARM')
        expect(keaTable.nativeElement.innerText).toContain('Release Notes')
    })

    it('should display BIND9 releases table', () => {
        // Arrange & Act & Assert
        apisWorkingFine()
        expect(component.machines.length).toEqual(4)

        // There should be 4 tables.
        const tablesDe = fixture.debugElement.queryAll(By.css('table.p-datatable-table'))
        expect(tablesDe.length).toEqual(4)
        const bindTable = tablesDe[2]

        // There should be 2 rows for stable releases and 1 for development.
        expect(bindTable.queryAll(By.css('tbody tr')).length).toEqual(3)
        expect(bindTable.nativeElement.innerText).toContain('Current Stable')
        expect(bindTable.nativeElement.innerText).toContain('Development')
        expect(bindTable.nativeElement.innerText).toContain('BIND 9.20 ARM')
        expect(bindTable.nativeElement.innerText).toContain('Release Notes')
    })

    it('should display stork releases table', () => {
        // Arrange & Act & Assert
        apisWorkingFine()
        expect(component.machines.length).toEqual(4)

        // There should be 4 tables.
        const tablesDe = fixture.debugElement.queryAll(By.css('table.p-datatable-table'))
        expect(tablesDe.length).toEqual(4)
        const storkTable = tablesDe[3]

        // There is 1 row for development release.
        expect(storkTable.queryAll(By.css('tbody tr')).length).toEqual(1)
        expect(storkTable.nativeElement.innerText).toContain('Development')
        expect(storkTable.nativeElement.innerText).toContain('Stork ARM')
        expect(storkTable.nativeElement.innerText).toContain('Release Notes')
    })

    it('should display version alert dismiss message', () => {
        // Arrange & Act & Assert
        let alert: VersionAlert
        versionService.getVersionAlert().subscribe((a) => (alert = a))
        apisWorkingFine()
        const de = fixture.debugElement.query(By.css('.header-message .p-messages .p-message-warn'))
        expect(de).toBeTruthy()
        expect(de.nativeElement.innerText).toContain('Action required')

        // There is Kea daemons versions mismatch in fakeMachinesResponse, so the highest error severity alert is expected.
        expect(alert).toBeTruthy()
        expect(alert.severity).toEqual(Severity.error)

        // There is a button to dismiss the alert.
        const btn = de.query(By.css('button'))
        expect(btn).toBeTruthy()
        spyOn(versionService, 'dismissVersionAlert').and.callThrough()
        btn.triggerEventHandler('click')
        expect(versionService.dismissVersionAlert).toHaveBeenCalledTimes(1)
        expect(alert.detected).toBeFalse()
    })

    it('should display button to refresh data', () => {
        // Arrange & Act & Assert
        apisWorkingFine()

        // There is a button to refresh the data.
        const de = fixture.debugElement.query(By.css('p-button[label="Refresh Versions"]'))
        expect(de).toBeTruthy()
        expect(de.nativeElement.innerText).toContain('Refresh Versions')

        const btn = de.query(By.css('button'))
        expect(btn).toBeTruthy()
        spyOn(versionService, 'refreshData').and.callThrough()
        btn.triggerEventHandler('click')
        fixture.detectChanges()
        expect(versionService.refreshData).toHaveBeenCalledTimes(1)
    })

    it('should display error when failed to get current data from version service ', () => {
        // Arrange & Act & Assert
        expect(() => {
            apisThrowingErrors(true)
        }).toThrowError('version service error1')
        expect(getCurrentDataSpy).toHaveBeenCalledTimes(1)
        expect(getMachinesAppsVersionsSpy).toHaveBeenCalledTimes(0)
    })

    it('should display error when failed to get machines data from service api', () => {
        // Arrange & Act & Assert
        apisThrowingErrors(false)
        expect(getCurrentDataSpy).toHaveBeenCalledTimes(1)
        expect(getMachinesAppsVersionsSpy).toHaveBeenCalledTimes(1)
        expect(messageAddSpy).toHaveBeenCalledOnceWith({
            severity: 'error',
            summary: 'Error retrieving software version data',
            detail: 'An error occurred when retrieving software version data: version service error2',
            life: 10000,
        })
    })

    it('should display stork server update notification message', () => {
        // Arrange & Act & Assert
        apisWorkingFine()
        const wrappers = fixture.debugElement.queryAll(By.css('.p-panel-content .p-message-wrapper'))
        expect(wrappers).toBeTruthy()
        expect(wrappers.length).toBeGreaterThan(0)

        // This should be first message of all.
        const de = wrappers[0]
        expect(de).toBeTruthy()
        expect(de.nativeElement.innerText).toContain('Stork server update available')
        expect(de.nativeElement.innerText).toContain('Stork server update is available (1.19.0)')
    })
})
