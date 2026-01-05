import { ComponentFixture, fakeAsync, TestBed, tick } from '@angular/core/testing'

import { ZonesPageComponent } from './zones-page.component'
import { provideHttpClientTesting } from '@angular/common/http/testing'
import { ConfirmationService, MessageService, TableState } from 'primeng/api'
import { Router, provideRouter } from '@angular/router'
import { DNSClass, DNSService, ZoneInventoryState, ZoneInventoryStates, ZoneRR, ZoneRRs, Zones, Zone } from '../backend'
import { Observable, of } from 'rxjs'
import {
    HttpEventType,
    HttpHeaders,
    HttpResponse,
    HttpStatusCode,
    provideHttpClient,
    withInterceptorsFromDi,
} from '@angular/common/http'

import { provideNoopAnimations } from '@angular/platform-browser/animations'
import { By } from '@angular/platform-browser'
import createSpyObj = jasmine.createSpyObj
import objectContaining = jasmine.objectContaining
import { take } from 'rxjs/operators'
import { NgZone } from '@angular/core'
import { AuthService } from '../auth.service'

describe('ZonesPageComponent', () => {
    let component: ZonesPageComponent
    let fixture: ComponentFixture<ZonesPageComponent>
    let dnsApi: jasmine.SpyObj<DNSService>
    let putZonesFetchSpy: any
    let getZonesSpy: any
    let getZoneRRsSpy: any
    let messageService: jasmine.SpyObj<MessageService>
    let messageAddSpy: any
    let getZonesFetchWithStatusSpy: any
    let authService: AuthService
    let router: Router

    const noContent = {
        status: HttpStatusCode.NoContent,
    }

    const progress1 = {
        appsCount: 3,
        completedAppsCount: 0,
        status: HttpStatusCode.Accepted,
    }

    const progress2 = {
        appsCount: 3,
        completedAppsCount: 1,
        status: HttpStatusCode.Accepted,
    }

    const progress3 = {
        appsCount: 3,
        completedAppsCount: 2,
        status: HttpStatusCode.Accepted,
    }

    const progress4 = {
        appsCount: 3,
        completedAppsCount: 3,
        status: HttpStatusCode.Accepted,
    }

    const zoneFetchStates = {
        items: [
            {
                appId: 30,
                appName: 'bind9@agent-bind9',
                createdAt: '2025-03-04T20:37:05.096Z',
                daemonId: 73,
                status: 'ok',
                zoneConfigsCount: 105,
            },
            {
                appId: 31,
                appName: 'bind9@agent-bind9-2',
                createdAt: '2025-03-04T20:37:13.106Z',
                daemonId: 74,
                status: 'erred',
                error: 'Fetching custom error',
                zoneConfigsCount: 105,
            },
        ],
        total: 2,
        status: HttpStatusCode.Ok,
    }

    const zoneFetchStatesHttpResp: HttpResponse<ZoneInventoryStates> = {
        body: {
            items: [
                {
                    appId: 30,
                    appName: 'bind9@agent-bind9',
                    createdAt: '2025-03-04T20:37:05.096Z',
                    daemonId: 73,
                    status: 'ok',
                    zoneConfigsCount: 105,
                },
                {
                    appId: 31,
                    appName: 'bind9@agent-bind9-2',
                    createdAt: '2025-03-04T20:37:13.106Z',
                    daemonId: 74,
                    status: 'erred',
                    error: 'Fetching custom error',
                    zoneConfigsCount: 105,
                },
            ],
            total: 2,
        },
        status: HttpStatusCode.Ok,
        type: HttpEventType.Response,
        clone: function (): HttpResponse<ZoneInventoryStates> {
            return null
        },
        headers: new HttpHeaders(),
        statusText: '',
        url: '',
        ok: true,
    }

    const noZones: Zones = { items: null }
    const fakeZones: Zones = {
        items: [
            {
                id: 21320,
                localZones: [
                    {
                        appId: 30,
                        appName: 'bind9@agent-bind9',
                        _class: 'IN',
                        daemonId: 73,
                        loadedAt: '2025-03-03T17:36:14.000Z',
                        serial: 0,
                        view: '_default',
                        zoneType: 'primary',
                    },
                    {
                        appId: 31,
                        appName: 'bind9@agent-bind9-2',
                        _class: 'IN',
                        daemonId: 74,
                        loadedAt: '2025-03-03T17:36:14.000Z',
                        serial: 0,
                        view: '_default',
                        zoneType: 'primary',
                    },
                ],
                name: 'EMPTY.AS112.ARPA',
                rname: 'ARPA.AS112.EMPTY',
            },
            {
                id: 21321,
                localZones: [
                    {
                        appId: 30,
                        appName: 'bind9@agent-bind9',
                        _class: 'IN',
                        daemonId: 73,
                        loadedAt: '2025-03-03T17:36:14.000Z',
                        serial: 0,
                        view: '_default',
                        zoneType: 'primary',
                    },
                    {
                        appId: 31,
                        appName: 'bind9@agent-bind9-2',
                        _class: 'IN',
                        daemonId: 74,
                        loadedAt: '2025-03-03T17:36:14.000Z',
                        serial: 0,
                        view: '_default',
                        zoneType: 'primary',
                    },
                ],
                name: 'HOME.ARPA',
                rname: 'ARPA.HOME',
            },
            {
                id: 21322,
                localZones: [
                    {
                        appId: 30,
                        appName: 'bind9@agent-bind9',
                        _class: 'IN',
                        daemonId: 73,
                        loadedAt: '2025-03-03T17:36:14.000Z',
                        serial: 0,
                        view: '_default',
                        zoneType: 'primary',
                    },
                    {
                        appId: 31,
                        appName: 'bind9@agent-bind9-2',
                        _class: 'IN',
                        daemonId: 74,
                        loadedAt: '2025-03-03T17:36:14.000Z',
                        serial: 0,
                        view: '_default',
                        zoneType: 'primary',
                    },
                ],
                name: '0.IN-ADDR.ARPA',
                rname: 'ARPA.IN-ADDR.0',
            },
            {
                id: 21323,
                localZones: [
                    {
                        appId: 30,
                        appName: 'bind9@agent-bind9',
                        _class: 'IN',
                        daemonId: 73,
                        loadedAt: '2025-03-03T17:36:14.000Z',
                        serial: 0,
                        view: '_default',
                        zoneType: 'primary',
                        rpz: true,
                    },
                    {
                        appId: 31,
                        appName: 'bind9@agent-bind9-2',
                        _class: 'IN',
                        daemonId: 74,
                        loadedAt: '2025-03-03T17:36:14.000Z',
                        serial: 0,
                        view: '_default',
                        zoneType: 'primary',
                        rpz: true,
                    },
                ],
                name: 'rpz.example.com',
                rname: 'com.example.rpz',
            },
        ],
        total: 3,
    }

    const zoneRRs: ZoneRRs = {
        items: [
            {
                name: 'example.com.',
                ttl: 3600,
                rrClass: 'IN',
                rrType: 'SOA',
                data: 'ns1.example.com. admin.example.com. 2024031501 3600 900 1209600 300',
            } as ZoneRR,
            {
                name: 'www.example.com.',
                ttl: 3600,
                rrClass: 'IN',
                rrType: 'A',
                data: '192.0.2.1',
            } as ZoneRR,
        ],
    }

    beforeEach(async () => {
        dnsApi = createSpyObj('DNSService', ['getZonesFetch', 'putZonesFetch', 'getZones', 'getZoneRRs'])
        putZonesFetchSpy = dnsApi.putZonesFetch
        getZonesSpy = dnsApi.getZones
        getZoneRRsSpy = dnsApi.getZoneRRs

        // By default, emits null response and completes without any error.
        putZonesFetchSpy.and.returnValue(of(null))
        // Returns replies in order: no Zones, 3 Zones.
        getZonesSpy.and.returnValues(of(noZones), of(fakeZones))
        // Returns zone RRs.
        getZoneRRsSpy.and.returnValue(of(zoneRRs))

        messageService = createSpyObj('MessageService', ['add'])
        messageAddSpy = messageService.add

        await TestBed.configureTestingModule({
            providers: [
                { provide: MessageService, useValue: messageService },
                { provide: DNSService, useValue: dnsApi },
                ConfirmationService,
                provideNoopAnimations(),
                provideHttpClient(withInterceptorsFromDi()),
                provideHttpClientTesting(),
                provideRouter([]),
            ],
        }).compileComponents()

        fixture = TestBed.createComponent(ZonesPageComponent)
        component = fixture.componentInstance
        authService = fixture.debugElement.injector.get(AuthService)
        router = fixture.debugElement.injector.get(Router)
        spyOn(router, 'navigate')
        spyOn(authService, 'superAdmin').and.returnValue(true)

        // By default, fake that wasZoneFetchSent returns false from session storage.
        spyOn(component, 'wasZoneFetchSent').and.returnValue(false)

        // Returns replies in order: 204 No content, 200 Ok ZoneInventoryStates, 202 Accepted ZonesFetchStatus 0/3, 202 Accepted progress 1/3,
        // 202 Accepted progress 2/3, 202 Accepted progress 3/3, 200 Ok ZoneInventoryStates.
        getZonesFetchWithStatusSpy = spyOn(component, 'getZonesFetchWithStatus')
        getZonesFetchWithStatusSpy.and.returnValues(
            of(noContent),
            of(zoneFetchStates) as Observable<any>,
            of(progress1),
            of(progress2),
            of(progress3),
            of(progress4),
            of(zoneFetchStates) as Observable<any>
        )

        fixture.detectChanges()
        expect(component.zonesLoading).withContext('zones data does not load on init').toBeFalse()
        expect(component.zonesFetchStatesLoading).withContext('zones fetch status data loads on init').toBeTrue()

        // Do not save table state between tests, because that makes tests unstable.
        spyOn(component.zonesTable, 'saveState').and.callFake(() => {})

        // Let's wait for async actions that happen on component init:
        // 1. Zones Fetch Status table data is fetched on every init
        // 2. Zones table data is lazily loaded on every init

        // onLazyLoadZones() is manually triggering change detection cycle in order to solve NG0100: ExpressionChangedAfterItHasBeenCheckedError
        // Call await fixture.whenStable() two times to wait for another round of change detection.
        await fixture.whenStable()
        await fixture.whenStable()

        // Wait for getZones and getZonesFetch async responses
        await fixture.whenStable()

        expect(component.zonesLoading).withContext('Zones table data loading should be done').toBeFalse()
        expect(component.zonesFetchStatesLoading)
            .withContext('Zones Fetch Status table data loading should be done')
            .toBeFalse()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should call dns apis on init', async () => {
        // Arrange + Act + Assert
        expect(getZonesSpy).not.toHaveBeenCalled()
        expect(component.getZonesFetchWithStatus).toHaveBeenCalledTimes(1)
        expect(putZonesFetchSpy).toHaveBeenCalledTimes(0)
        expect(messageAddSpy).toHaveBeenCalledOnceWith(
            objectContaining({ summary: 'Zones not fetched', severity: 'info' })
        )
    })

    it('should display explanation message and fetch zones button', async () => {
        // Arrange + Act
        expect(component.zonesLoading).withContext('Zones table data loading should be done').toBeFalse()
        fixture.detectChanges()
        const onlyCellDe = fixture.debugElement.query(By.css('#zones-table tbody td'))

        // Assert
        expect(onlyCellDe).toBeTruthy()
        const messageDe = onlyCellDe.query(By.css('.p-message'))
        const buttonDe = onlyCellDe.query(By.css('button'))
        expect(messageDe).toBeTruthy()
        expect(buttonDe).toBeTruthy()
        expect(messageDe.nativeElement.innerText).toContain('Zones were not fetched yet')
        expect(buttonDe.nativeElement.innerText).toContain('Fetch Zones')
    })

    xit('should open and close tabs', async () => {
        // TODO: this test should be moved away from Karma tests.
        // Arrange
        expect(component.zonesLoading).withContext('Zones table data loading should be done').toBeFalse()
        const refreshBtnDe = fixture.debugElement.query(By.css('#refresh-zones-data button'))
        expect(refreshBtnDe).toBeTruthy()
        // Click on Refresh List button
        refreshBtnDe.nativeElement.click()
        fixture.detectChanges()
        expect(component.zonesLoading).withContext('zones data loads').toBeTrue()
        await fixture.whenStable()
        expect(component.zonesLoading).withContext('Zones table data loading should be done').toBeFalse()
        expect(component.zones).toEqual(fakeZones.items)
        expect(component.zonesTotal).toEqual(fakeZones.total)
        fixture.detectChanges()

        // There are all zones listed.
        const tableRows = fixture.debugElement.queryAll(By.css('#zones-table tbody tr'))
        expect(tableRows).toBeTruthy()
        expect(tableRows.length).toEqual(4)

        // Act + Assert
        const zonesTabDe = fixture.debugElement.query(By.css('ul.p-tabview-nav li:first-child a'))
        expect(zonesTabDe).toBeTruthy()

        // Try to open tab 1.
        const firstRowBtns = tableRows[0].queryAll(By.css('button'))
        expect(firstRowBtns).toBeTruthy()
        // There are 2 buttons per row: 1. expand/collapse row; 2. anchor to detailed zone view
        expect(firstRowBtns.length).toEqual(2)
        firstRowBtns[1].nativeElement.click()
        await fixture.whenStable()
        fixture.detectChanges()
        // expect(component.activeTabIdx).toEqual(1)
        // expect(component.openTabs.length).toEqual(1)
        // expect(component.openTabs).toContain(component.zones[0])

        // Go back to first tab.
        zonesTabDe.nativeElement.click()
        fixture.detectChanges()

        // Try to open tab 2.
        const secondRowBtns = tableRows[1].queryAll(By.css('button'))
        expect(secondRowBtns).toBeTruthy()
        // There are 2 buttons per row: 1. expand/collapse row; 2. anchor to detailed zone view
        expect(secondRowBtns.length).toEqual(2)
        secondRowBtns[1].nativeElement.click()
        await fixture.whenStable()
        fixture.detectChanges()
        // expect(component.activeTabIdx).toEqual(2)
        // expect(component.openTabs.length).toEqual(2)
        // expect(component.openTabs).toContain(component.zones[1])
        // expect(component.openTabs).toContain(component.zones[0])
        // expect(component.openTabs).not.toContain(component.zones[2])

        // Go back to first tab.
        zonesTabDe.nativeElement.click()
        fixture.detectChanges()

        // Try to open tab 1 again.
        firstRowBtns[1].nativeElement.click()
        fixture.detectChanges()
        // expect(component.activeTabIdx).toEqual(1)
        // expect(component.openTabs.length).toEqual(2)
        // expect(component.openTabs).toContain(component.zones[0])
        // expect(component.openTabs).toContain(component.zones[1])
        // expect(component.openTabs).not.toContain(component.zones[2])

        const closeTabBtns = fixture.debugElement.queryAll(By.css('ul.p-tabview-nav .p-icon-wrapper'))
        expect(closeTabBtns).toBeTruthy()
        expect(closeTabBtns.length).toEqual(2)

        // Close tab 2.
        closeTabBtns[1].nativeElement.click()
        await fixture.whenStable()
        fixture.detectChanges()
        // expect(component.activeTabIdx).toEqual(1)
        // expect(component.openTabs.length).toEqual(1)
        // expect(component.openTabs).not.toContain(component.zones[1])
        // expect(component.openTabs).toContain(component.zones[0])

        // Close tab 1.
        closeTabBtns[0].nativeElement.click()
        await fixture.whenStable()
        fixture.detectChanges()
        // expect(component.activeTabIdx).toEqual(0)
        // expect(component.openTabs.length).toEqual(0)
        // expect(component.openTabs).not.toContain(component.zones[0])
        // expect(component.openTabs).not.toContain(component.zones[1])
    })

    it('should display confirmation dialog when fetch zones clicked', async () => {
        // Arrange + Act
        expect(component.zonesLoading).withContext('Zones table data loading should be done').toBeFalse()
        fixture.detectChanges()
        const fetchZonesBtn = fixture.debugElement.query(By.css('#fetch-zones button'))
        expect(fetchZonesBtn).toBeTruthy()
        fetchZonesBtn.nativeElement.click()
        fixture.detectChanges()

        // Assert
        const confirmDialog = fixture.debugElement.query(By.css('.p-confirmdialog'))
        expect(confirmDialog).toBeTruthy()
        expect(confirmDialog.nativeElement.innerText).toContain('Confirm Fetching Zones')
        expect(confirmDialog.nativeElement.innerText).toContain('Are you sure you want to continue?')

        // cancel
        const rejectBtnDe = confirmDialog.query(By.css('button.p-confirmdialog-reject-button'))
        expect(rejectBtnDe).toBeTruthy()
        rejectBtnDe.nativeElement.click()
        fixture.detectChanges()
        expect(putZonesFetchSpy).toHaveBeenCalledTimes(0)
    })

    it('should retrieve list of zone fetch states', async () => {
        // Arrange
        expect(component.zonesFetchStatesLoading)
            .withContext('Zones Fetch Status table data loading should be done')
            .toBeFalse()

        // Display Fetch Status dialog
        const fetchStatusBtnDe = fixture.debugElement.query(By.css('#fetch-status button'))
        expect(fetchStatusBtnDe).toBeTruthy()
        fetchStatusBtnDe.nativeElement.click()
        fixture.detectChanges()
        expect(component.fetchStatusVisible).toBeTrue()

        // Locate the Refresh List button
        const refreshBtnDe = fixture.debugElement.query(By.css('#refresh-fetch-status-data button'))
        expect(refreshBtnDe).toBeTruthy()

        // Act
        refreshBtnDe.nativeElement.click()
        fixture.detectChanges()

        // Assert
        expect(component.zonesFetchStatesLoading).withContext('data should be loading').toBeTrue()
        await fixture.whenStable()
        fixture.detectChanges()
        expect(component.zonesFetchStatesLoading).withContext('data loading should be done').toBeFalse()
        expect(component.getZonesFetchWithStatus).toHaveBeenCalledTimes(2)
        expect(component.zonesFetchStates).toEqual(zoneFetchStates.items as Array<ZoneInventoryState>)
        expect(component.zonesFetchStatesTotal).toEqual(zoneFetchStates.total)
    })

    it('should return get zones fetch with http status observable', () => {
        // Arrange
        getZonesFetchWithStatusSpy.and.callThrough()
        dnsApi.getZonesFetch.and.returnValue(of(zoneFetchStatesHttpResp))
        let resp

        // Act
        component
            .getZonesFetchWithStatus()
            .pipe(take(1))
            .subscribe((r) => (resp = r))

        // Assert
        expect(resp.status).toEqual(HttpStatusCode.Ok)
        expect(resp.items).toEqual(zoneFetchStatesHttpResp.body.items)
        expect(resp.total).toEqual(zoneFetchStatesHttpResp.body.total)
        expect(resp.appsCount).toBeUndefined()
        expect(resp.completedAppsCount).toBeUndefined()
    })

    it('should init filter dropdowns and multiselect', () => {
        // Arrange + Act + Assert
        expect(component.zoneTypes.length).toBeGreaterThan(0)
        expect(component.zoneClasses.length).toBeGreaterThan(0)
        expect(component.appTypes.length).toBeGreaterThan(0)
        expect(component.appTypes[0].value).toBeTruthy()
        expect(component.appTypes[0].name).toBeTruthy()
        expect(component.zoneClasses).not.toContain(DNSClass.Any)
    })

    xit('should activate first tab', () => {
        // TODO: this test should be moved away from Karma tests.
        // Arrange
        // component.activeTabIdx = 1
        // Act
        // component.activateFirstTab()
        // Assert
        // expect(component.activeTabIdx).toBe(0)
    })

    it('should store rows per page for zones table', () => {
        // Arrange + Act
        component.storeZonesTableRowsPerPage(10)

        // Expect
        const stateString = localStorage.getItem('zones-table-state')
        expect(stateString).toBeTruthy()
        const state: TableState = JSON.parse(stateString)
        expect('rows' in state).toBeTrue()
        expect(state.rows).toEqual(10)
    })

    it('should filter zones table by serial', fakeAsync(() => {
        // Arrange
        const filterInput = fixture.debugElement.query(By.css('#zone-serial'))
        expect(filterInput).toBeTruthy()
        filterInput.nativeElement.value = '1'

        // Act
        filterInput.nativeElement.dispatchEvent(new Event('input'))
        tick(300)
        fixture.detectChanges()

        // Assert
        expect(router.navigate).toHaveBeenCalledWith([], {
            queryParams: {
                zoneType: null,
                rpz: null,
                appId: null,
                zoneSerial: '1',
                zoneClass: null,
                appType: null,
                text: null,
            },
        })
    }))

    it('should filter zones table by app id', fakeAsync(() => {
        // Arrange
        const inputNumber = fixture.debugElement.query(By.css('[inputId="app-id"]'))
        expect(inputNumber).toBeTruthy()

        // Act
        inputNumber.componentInstance.handleOnInput(new InputEvent('input'), '', 9)
        tick(300)
        fixture.detectChanges()

        // Assert
        expect(router.navigate).toHaveBeenCalledWith([], {
            queryParams: {
                zoneType: null,
                rpz: null,
                appId: 9,
                zoneSerial: null,
                zoneClass: null,
                appType: null,
                text: null,
            },
        })
    }))

    it('should filter zones table by text', fakeAsync(() => {
        // Arrange
        const filterInput = fixture.debugElement.query(By.css('#text-filter'))
        expect(filterInput).toBeTruthy()
        filterInput.nativeElement.value = 'test'

        // Act
        filterInput.nativeElement.dispatchEvent(new Event('input'))
        tick(300)
        fixture.detectChanges()

        // Assert
        expect(router.navigate).toHaveBeenCalledWith([], {
            queryParams: {
                zoneType: null,
                rpz: null,
                appId: null,
                zoneSerial: null,
                zoneClass: null,
                appType: null,
                text: 'test',
            },
        })
    }))

    xit('should display feedback when wrong filter in query params', async () => {
        // TODO: this test should be moved away from Karma tests.
        // Arrange + Act + Assert
        getZonesSpy.and.returnValue(of(noZones))
        const zone = fixture.debugElement.injector.get(NgZone)
        const r = fixture.debugElement.injector.get(Router)

        await zone.run(() => r.navigate([], { queryParams: { foo: 'bar' } }))
        fixture.detectChanges()
        expect(messageAddSpy).toHaveBeenCalledWith(
            jasmine.objectContaining({
                severity: 'error',
                summary: 'Wrong URL parameter value',
                detail: jasmine.stringContaining('parameter foo not supported'),
            })
        )

        await zone.run(() => r.navigate([], { queryParams: { appId: 'bar' } }))
        fixture.detectChanges()
        expect(messageAddSpy).toHaveBeenCalledWith(
            jasmine.objectContaining({
                severity: 'error',
                summary: 'Wrong URL parameter value',
                detail: jasmine.stringContaining('requires numeric value'),
            })
        )

        await zone.run(() => r.navigate([], { queryParams: { appType: 'bar' } }))
        fixture.detectChanges()
        expect(messageAddSpy).toHaveBeenCalledWith(
            jasmine.objectContaining({
                severity: 'error',
                summary: 'Wrong URL parameter value',
                detail: jasmine.stringContaining('appType requires one of the values'),
            })
        )

        await zone.run(() => r.navigate([], { queryParams: { zoneType: 'bar' } }))
        fixture.detectChanges()
        expect(messageAddSpy).toHaveBeenCalledWith(
            jasmine.objectContaining({
                severity: 'error',
                summary: 'Wrong URL parameter value',
                detail: jasmine.stringContaining('zoneType requires one of the values'),
            })
        )

        await zone.run(() => r.navigate([], { queryParams: { zoneClass: 'bar' } }))
        fixture.detectChanges()
        expect(messageAddSpy).toHaveBeenCalledWith(
            jasmine.objectContaining({
                severity: 'error',
                summary: 'Wrong URL parameter value',
                detail: jasmine.stringContaining('zoneClass requires one of the values'),
            })
        )
    })

    xit('should filter zones when correct filter in query params', async () => {
        // TODO: this test should be moved away from Karma tests.
        // Arrange
        const zone = fixture.debugElement.injector.get(NgZone)
        const r = fixture.debugElement.injector.get(Router)

        // Act
        await zone.run(() =>
            r.navigate([], {
                queryParams: {
                    appId: 2,
                    zoneSerial: '123',
                    zoneType: ['builtin', 'primary'],
                    zoneClass: 'IN',
                    appType: 'bind9',
                    text: 'test',
                },
            })
        )
        fixture.detectChanges()

        // Assert
        expect(messageAddSpy).not.toHaveBeenCalledWith(
            jasmine.objectContaining({
                severity: 'error',
            })
        )
        expect(getZonesSpy).toHaveBeenCalledTimes(2)
        expect(getZonesSpy).toHaveBeenCalledWith(
            0,
            10,
            'bind9',
            jasmine.arrayContaining(['primary', 'builtin']),
            'IN',
            'test',
            2,
            '123',
            null
        )
    })

    it('should have non empty enum values for all enum type of supported query param filters', () => {
        // Arrange + Act + Assert
        for (const paramKey in component.supportedQueryParamFilters) {
            if (component.supportedQueryParamFilters[paramKey].type === 'enum') {
                expect(component.supportedQueryParamFilters[paramKey].enumValues).toBeTruthy()
                expect(component.supportedQueryParamFilters[paramKey].enumValues.length).toBeGreaterThan(0)
            }
        }
    })

    it('should return unique zone types', () => {
        // Arrange
        const zone: Zone = {
            name: 'example.org',
            id: 10,
            localZones: [
                {
                    zoneType: 'builtin',
                },
                {
                    zoneType: 'builtin',
                },
                {
                    zoneType: 'builtin',
                },
                {
                    zoneType: 'primary',
                },
                {
                    zoneType: 'primary',
                },
                {
                    zoneType: 'secondary',
                },
            ],
        }
        const expectedTypes = ['primary', 'secondary', 'builtin']

        // Act
        const result = component.getUniqueZoneTypes(zone)

        // Assert
        expect(result).toEqual(jasmine.arrayWithExactContents(expectedTypes))
    })

    it('should return empty unique zone types', () => {
        // Arrange
        const zone: Zone = {
            name: 'example.org',
            id: 10,
            localZones: [],
        }

        // Act
        const result = component.getUniqueZoneTypes(zone)

        // Assert
        expect(result).toBeTruthy()
        expect(result.length).toEqual(0)
    })

    it('should get zone serial info when there is no mismatch', () => {
        // Arrange
        const zone: Zone = {
            name: 'example.org',
            id: 10,
            localZones: [
                {
                    serial: 12345,
                },
                {
                    serial: 12345,
                },
                {
                    serial: 12345,
                },
            ],
        }

        // Act
        const result = component.getZoneSerialInfo(zone)

        // Assert
        expect(result).toEqual(jasmine.objectContaining({ serial: '12345', hasMismatch: false }))
    })

    it('should get zone serial info when there is mismatch', () => {
        // Arrange
        const zone: Zone = {
            name: 'example.org',
            id: 10,
            localZones: [
                {
                    serial: 12345,
                },
                {
                    serial: 12345,
                },
                {
                    serial: 12344,
                },
            ],
        }

        // Act
        const result = component.getZoneSerialInfo(zone)

        // Assert
        expect(result).toEqual(jasmine.objectContaining({ serial: '12345', hasMismatch: true }))
    })

    it('should get zone serial info when there are no local zones', () => {
        // Arrange
        const zone: Zone = {
            name: 'example.org',
            id: 10,
            localZones: [],
        }

        // Act
        const result = component.getZoneSerialInfo(zone)

        // Assert
        expect(result).toEqual(jasmine.objectContaining({ serial: 'N/A', hasMismatch: false }))
    })

    xit('should open zone viewer dialog', async () => {
        // TODO: this test should be moved away from Karma tests.
        // Arrange the zones list.
        expect(component.zonesLoading).withContext('Zones table data loading should be done').toBeFalse()
        const refreshBtnDe = fixture.debugElement.query(By.css('#refresh-zones-data button'))
        expect(refreshBtnDe).toBeTruthy()
        // Click on Refresh List button
        refreshBtnDe.nativeElement.click()
        fixture.detectChanges()
        expect(component.zonesLoading).withContext('zones data loads').toBeTrue()
        await fixture.whenStable()
        expect(component.zonesLoading).withContext('Zones table data loading should be done').toBeFalse()
        expect(component.zones).toEqual(fakeZones.items)
        expect(component.zonesTotal).toEqual(fakeZones.total)
        fixture.detectChanges()

        // There are all zones listed.
        const tableRows = fixture.debugElement.queryAll(By.css('#zones-table tbody tr'))
        expect(tableRows).toBeTruthy()
        expect(tableRows.length).toEqual(4)

        const zonesTabDe = fixture.debugElement.query(By.css('ul.p-tabview-nav li:first-child a'))
        expect(zonesTabDe).toBeTruthy()

        // Try to open a tab.
        const firstRowBtns = tableRows[0].queryAll(By.css('button'))
        expect(firstRowBtns).toBeTruthy()
        // There are 2 buttons per row: 1. expand/collapse row; 2. anchor to detailed zone view
        expect(firstRowBtns.length).toEqual(2)
        firstRowBtns[1].nativeElement.click()
        await fixture.whenStable()
        fixture.detectChanges()
        // expect(component.activeTabIdx).toEqual(1)
        // expect(component.openTabs.length).toEqual(1)
        // expect(component.openTabs).toContain(component.zones[0])

        // Initially, the zone details should not be loaded.
        expect(dnsApi.getZoneRRs).toHaveBeenCalledTimes(0)

        // Get the table of DNS servers associated with the zone.
        const dnsServersFieldset = fixture.debugElement.query(By.css('[legend="DNS Views Associated with the Zone"]'))
        expect(dnsServersFieldset).toBeTruthy()

        // Get the button to show the zone viewer dialog.
        const showViewerBtns = dnsServersFieldset.queryAll(By.css('button'))
        expect(showViewerBtns.length).toEqual(2)
        showViewerBtns[1].nativeElement.click()
        await fixture.whenStable()
        fixture.detectChanges()

        // It should result in getting the zone RRs.
        expect(dnsApi.getZoneRRs).toHaveBeenCalledOnceWith(
            fakeZones.items[0].localZones[1].daemonId,
            fakeZones.items[0].localZones[1].view,
            fakeZones.items[0].id
        )

        // Wait for the zone viewer dialog to be displayed.
        expect(
            component.zoneViewerDialogVisible()[
                `${fakeZones.items[0].localZones[1].daemonId}:${fakeZones.items[0].localZones[1].view}:${fakeZones.items[0].id}`
            ]
        ).toBeTrue()
        await fixture.whenStable()
        fixture.detectChanges()

        const zoneViewer = fixture.debugElement.query(By.css('app-zone-viewer'))
        expect(zoneViewer).toBeTruthy()
        expect(zoneViewer.nativeElement.innerText).toContain(
            'ns1.example.com. admin.example.com. 2024031501 3600 900 1209600 300'
        )
    })

    xit('should not filter zones table by app id value zero', fakeAsync(() => {
        // TODO: this test should be moved away from Karma tests.
        // Arrange
        const inputNumber = fixture.debugElement.query(By.css('[inputId="app-id"]'))
        expect(inputNumber).toBeTruthy()

        // Act
        inputNumber.componentInstance.handleOnInput(new InputEvent('input'), '', 0)
        tick(300)
        fixture.detectChanges()

        // Assert
        expect(getZonesSpy).toHaveBeenCalledTimes(2)
        // Since zero is forbidden filter value for numeric inputs, we expect that minimum allowed value (i.e. 1) will be used.
        expect(getZonesSpy).toHaveBeenCalledWith(0, 10, null, null, null, null, 1, null, null)
    }))

    it('should disable show zone button', () => {
        // Arrange
        let shouldDisableShowZone = component['_shouldDisableShowZone']
        expect(shouldDisableShowZone).toBeDefined()

        // Act + Assert
        expect(shouldDisableShowZone({ zoneType: 'primary' })).toBeFalse()
        expect(shouldDisableShowZone({ zoneType: 'secondary' })).toBeFalse()
        expect(shouldDisableShowZone({ zoneType: 'master' })).toBeFalse()
        expect(shouldDisableShowZone({ zoneType: 'slave' })).toBeFalse()
        expect(shouldDisableShowZone({ zoneType: 'mirror' })).toBeFalse()
        expect(shouldDisableShowZone({ zoneType: 'builtin' })).toBeTrue()
        expect(shouldDisableShowZone({ zoneType: 'delegation-only' })).toBeTrue()
        expect(shouldDisableShowZone({ zoneType: 'forward' })).toBeTrue()
        expect(shouldDisableShowZone({ zoneType: 'hint' })).toBeTrue()
        expect(shouldDisableShowZone({ zoneType: 'redirect' })).toBeTrue()
        expect(shouldDisableShowZone({ zoneType: 'static-stub' })).toBeTrue()
        expect(shouldDisableShowZone({ zoneType: 'stub' })).toBeTrue()
        expect(shouldDisableShowZone({ zoneType: undefined })).toBeTrue()
    })
})
