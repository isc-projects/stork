import { ComponentFixture, TestBed } from '@angular/core/testing'

import { ZonesPageComponent } from './zones-page.component'
import { HttpClientTestingModule } from '@angular/common/http/testing'
import { ConfirmationService, MessageService } from 'primeng/api'
import { BreadcrumbsComponent } from '../breadcrumbs/breadcrumbs.component'
import { DialogModule } from 'primeng/dialog'
import { ButtonModule } from 'primeng/button'
import { TableModule } from 'primeng/table'
import { TabViewModule } from 'primeng/tabview'
import { BreadcrumbModule } from 'primeng/breadcrumb'
import { HelpTipComponent } from '../help-tip/help-tip.component'
import { OverlayPanelModule } from 'primeng/overlaypanel'
import { RouterModule } from '@angular/router'
import { DNSService, ZoneInventoryState, ZoneInventoryStates, Zones, ZonesFetchStatus } from '../backend'
import { of } from 'rxjs'
import { HttpEventType, HttpHeaders, HttpResponse, HttpStatusCode } from '@angular/common/http'
import { ConfirmDialogModule } from 'primeng/confirmdialog'
import { MessageModule } from 'primeng/message'
import { ProgressBarModule } from 'primeng/progressbar'
import { SkeletonModule } from 'primeng/skeleton'
import { BrowserAnimationsModule } from '@angular/platform-browser/animations'
import { By } from '@angular/platform-browser'
import { PlaceholderPipe } from '../pipes/placeholder.pipe'
import { LocaltimePipe } from '../pipes/localtime.pipe'
import { TagModule } from 'primeng/tag'
import createSpyObj = jasmine.createSpyObj
import objectContaining = jasmine.objectContaining
import StatusEnum = ZoneInventoryState.StatusEnum
import { FieldsetModule } from 'primeng/fieldset'

describe('ZonesPageComponent', () => {
    let component: ZonesPageComponent
    let fixture: ComponentFixture<ZonesPageComponent>
    let dnsApi: any
    let getZonesFetchSpy: any
    let putZonesFetchSpy: any
    let getZonesSpy: any
    let messageService: any
    let messageAddSpy: any

    const noContent: HttpResponse<ZoneInventoryStates> = {
        body: null,
        status: HttpStatusCode.NoContent,
        type: HttpEventType.Response,
        clone: function (): HttpResponse<ZoneInventoryStates> {
            return null
        },
        headers: new HttpHeaders(),
        statusText: '',
        url: '',
        ok: true,
    }

    const progress1: HttpResponse<ZonesFetchStatus> = {
        body: { appsCount: 3, completedAppsCount: 0 },
        status: HttpStatusCode.Accepted,
        type: HttpEventType.Response,
        clone: function (): HttpResponse<ZonesFetchStatus> {
            return null
        },
        headers: new HttpHeaders(),
        statusText: '',
        url: '',
        ok: true,
    }

    const progress2: HttpResponse<ZonesFetchStatus> = {
        body: { appsCount: 3, completedAppsCount: 1 },
        status: HttpStatusCode.Accepted,
        type: HttpEventType.Response,
        clone: function (): HttpResponse<ZonesFetchStatus> {
            return null
        },
        headers: new HttpHeaders(),
        statusText: '',
        url: '',
        ok: true,
    }

    const progress3: HttpResponse<ZonesFetchStatus> = {
        body: { appsCount: 3, completedAppsCount: 2 },
        status: HttpStatusCode.Accepted,
        type: HttpEventType.Response,
        clone: function (): HttpResponse<ZonesFetchStatus> {
            return null
        },
        headers: new HttpHeaders(),
        statusText: '',
        url: '',
        ok: true,
    }

    const progress4: HttpResponse<ZonesFetchStatus> = {
        body: { appsCount: 3, completedAppsCount: 3 },
        status: HttpStatusCode.Accepted,
        type: HttpEventType.Response,
        clone: function (): HttpResponse<ZonesFetchStatus> {
            return null
        },
        headers: new HttpHeaders(),
        statusText: '',
        url: '',
        ok: true,
    }

    const zoneFetchStates: HttpResponse<ZoneInventoryStates> = {
        body: {
            items: [
                {
                    appId: 30,
                    appName: 'bind9@agent-bind9',
                    createdAt: '2025-03-04T20:37:05.096Z',
                    daemonId: 73,
                    status: 'ok',
                    zoneCount: 105,
                },
                {
                    appId: 31,
                    appName: 'bind9@agent-bind9-2',
                    createdAt: '2025-03-04T20:37:13.106Z',
                    daemonId: 74,
                    status: 'erred',
                    error: 'Fetching custom error',
                    zoneCount: 105,
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
                        zoneType: 'builtin',
                    },
                    {
                        appId: 31,
                        appName: 'bind9@agent-bind9-2',
                        _class: 'IN',
                        daemonId: 74,
                        loadedAt: '2025-03-03T17:36:14.000Z',
                        serial: 0,
                        view: '_default',
                        zoneType: 'builtin',
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
                        zoneType: 'builtin',
                    },
                    {
                        appId: 31,
                        appName: 'bind9@agent-bind9-2',
                        _class: 'IN',
                        daemonId: 74,
                        loadedAt: '2025-03-03T17:36:14.000Z',
                        serial: 0,
                        view: '_default',
                        zoneType: 'builtin',
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
                        zoneType: 'builtin',
                    },
                    {
                        appId: 31,
                        appName: 'bind9@agent-bind9-2',
                        _class: 'IN',
                        daemonId: 74,
                        loadedAt: '2025-03-03T17:36:14.000Z',
                        serial: 0,
                        view: '_default',
                        zoneType: 'builtin',
                    },
                ],
                name: '0.IN-ADDR.ARPA',
                rname: 'ARPA.IN-ADDR.0',
            },
        ],
        total: 3,
    }

    beforeEach(async () => {
        dnsApi = createSpyObj('DNSService', ['getZonesFetch', 'putZonesFetch', 'getZones'])
        getZonesFetchSpy = dnsApi.getZonesFetch
        putZonesFetchSpy = dnsApi.putZonesFetch
        getZonesSpy = dnsApi.getZones

        // Returns replies in order: 204 No content, 200 Ok ZoneInventoryStates, 202 Accepted ZonesFetchStatus 0/3, 202 Accepted progress 1/3,
        // 202 Accepted progress 2/3, 202 Accepted progress 3/3, 200 Ok ZoneInventoryStates.
        getZonesFetchSpy.and.returnValues(
            of(noContent),
            of(zoneFetchStates),
            of(progress1),
            of(progress2),
            of(progress3),
            of(progress4),
            of(zoneFetchStates)
        )
        // By default, emits null response and completes without any error.
        putZonesFetchSpy.and.returnValue(of(null))
        // Returns replies in order: no Zones, 3 Zones.
        getZonesSpy.and.returnValues(of(noZones), of(fakeZones))

        messageService = createSpyObj('MessageService', ['add'])
        messageAddSpy = messageService.add

        await TestBed.configureTestingModule({
            imports: [
                HttpClientTestingModule,
                DialogModule,
                ButtonModule,
                TableModule,
                TabViewModule,
                BreadcrumbModule,
                OverlayPanelModule,
                RouterModule.forRoot([]),
                ConfirmDialogModule,
                MessageModule,
                ProgressBarModule,
                SkeletonModule,
                BrowserAnimationsModule,
                TagModule,
                FieldsetModule,
            ],
            declarations: [ZonesPageComponent, BreadcrumbsComponent, HelpTipComponent, PlaceholderPipe, LocaltimePipe],
            providers: [
                { provide: MessageService, useValue: messageService },
                { provide: DNSService, useValue: dnsApi },
                ConfirmationService,
            ],
        }).compileComponents()

        fixture = TestBed.createComponent(ZonesPageComponent)
        component = fixture.componentInstance

        // By default, fake that wasZoneFetchSent returns false from session storage.
        spyOn(component, 'wasZoneFetchSent').and.returnValue(false)

        fixture.detectChanges()

        // Do not save table state between tests, because that makes tests unstable.
        spyOn(component.zonesTable, 'saveState').and.callFake(() => {})

        // Let's wait for async actions that happen on component init:
        // 1. Zones Fetch Status table data is fetched on every init
        // 2. Zones table data is lazily loaded on every init

        // onLazyLoadZones() is manually triggering change detection cycle in order to solve NG0100: ExpressionChangedAfterItHasBeenCheckedError
        // Call await fixture.whenStable() two times to wait for another round of change detection.
        await fixture.whenStable()
        await fixture.whenStable()

        expect(component.zonesLoading).withContext('zones data loads on init').toBeTrue()
        expect(component.zonesFetchStatesLoading).withContext('zones fetch status data loads on init').toBeTrue()

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
        expect(getZonesSpy).toHaveBeenCalledOnceWith(0, 10)
        expect(getZonesFetchSpy).toHaveBeenCalledTimes(1)
        expect(putZonesFetchSpy).toHaveBeenCalledTimes(0)
        expect(messageAddSpy).toHaveBeenCalledOnceWith(
            objectContaining({ summary: 'Zones not fetched', severity: 'info' })
        )
    })

    it('should retrieve list of zones', async () => {
        // Arrange + Act
        expect(component.zonesLoading).withContext('Zones table data loading should be done').toBeFalse()
        const refreshBtnDe = fixture.debugElement.query(By.css('#refresh-zones-data button'))
        expect(refreshBtnDe).toBeTruthy()
        // Click on Refresh List button
        refreshBtnDe.nativeElement.click()
        fixture.detectChanges()
        expect(component.zonesLoading).withContext('zones data loads').toBeTrue()
        await fixture.whenStable()

        // Assert
        expect(component.zonesLoading).withContext('Zones table data loading should be done').toBeFalse()
        expect(getZonesSpy).toHaveBeenCalledTimes(2)
        expect(getZonesSpy).toHaveBeenCalledWith(0, 10)
        expect(getZonesFetchSpy).toHaveBeenCalledTimes(1)
        expect(putZonesFetchSpy).toHaveBeenCalledTimes(0)
        expect(component.zones).toEqual(fakeZones.items)
        expect(component.zonesTotal).toEqual(fakeZones.total)
        fixture.detectChanges()

        // There should be no notification about zones that were not fetched yet.
        const messageDe = fixture.debugElement.query(By.css('#zones-table .p-inline-message'))
        expect(messageDe).toBeNull()

        // There are all 3 zones listed.
        const tableRows = fixture.debugElement.queryAll(By.css('#zones-table tbody tr'))
        expect(tableRows).toBeTruthy()
        expect(tableRows.length).toEqual(3)
        expect(tableRows[0].nativeElement.innerText).toContain(fakeZones.items[0].name)
        expect(tableRows[1].nativeElement.innerText).toContain(fakeZones.items[1].name)
        expect(tableRows[2].nativeElement.innerText).toContain(fakeZones.items[2].name)

        // Try to expand the row.
        const expandRowBtns = tableRows[0].queryAll(By.css('button'))
        expect(expandRowBtns).toBeTruthy()
        // There are 2 buttons per row: 1. expand/collapse row; 2. anchor to detailed zone view
        expect(expandRowBtns.length).toEqual(2)
        expandRowBtns[0].nativeElement.click()
        fixture.detectChanges()

        const innerRows = fixture.debugElement.queryAll(By.css('#zones-table tbody tbody tr'))
        expect(innerRows).toBeTruthy()
        expect(innerRows.length).toEqual(2)
        expect(innerRows[0].nativeElement.innerText).toContain(fakeZones.items[0].localZones[0].appName)
        expect(innerRows[1].nativeElement.innerText).toContain(fakeZones.items[0].localZones[1].appName)
    })

    it('should display explanation message and fetch zones button', async () => {
        // Arrange + Act
        expect(component.zonesLoading).withContext('Zones table data loading should be done').toBeFalse()
        fixture.detectChanges()
        const onlyCellDe = fixture.debugElement.query(By.css('#zones-table tbody td'))

        // Assert
        expect(onlyCellDe).toBeTruthy()
        const messageDe = onlyCellDe.query(By.css('.p-inline-message'))
        const buttonDe = onlyCellDe.query(By.css('button'))
        expect(messageDe).toBeTruthy()
        expect(buttonDe).toBeTruthy()
        expect(messageDe.nativeElement.innerText).toContain('Zones were not fetched yet')
        expect(buttonDe.nativeElement.innerText).toContain('Fetch Zones')
        expect(getZonesSpy).toHaveBeenCalledOnceWith(0, 10)
    })

    it('should get severity', () => {
        // Arrange + Act + Assert
        expect(component.getSeverity(StatusEnum.Busy)).toEqual('warning')
        expect(component.getSeverity(StatusEnum.Ok)).toEqual('success')
        expect(component.getSeverity(StatusEnum.Erred)).toEqual('danger')
        expect(component.getSeverity(StatusEnum.Uninitialized)).toEqual('secondary')
        expect(component.getSeverity(<StatusEnum>'foo')).toEqual('info')
    })

    it('should get tooltip', () => {
        // Arrange + Act + Assert
        expect(component.getTooltip(StatusEnum.Busy)).toContain('Zone inventory on the agent is busy')
        expect(component.getTooltip(StatusEnum.Ok)).toContain('successfully fetched all zones')
        expect(component.getTooltip(StatusEnum.Erred)).toContain('Error when communicating with a zone inventory')
        expect(component.getTooltip(StatusEnum.Uninitialized)).toContain(
            'Zone inventory on the agent was not initialized'
        )
        expect(component.getTooltip(<StatusEnum>'foo')).toBeNull()
    })

    it('should open and close tabs', async () => {
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

        // There are all 3 zones listed.
        const tableRows = fixture.debugElement.queryAll(By.css('#zones-table tbody tr'))
        expect(tableRows).toBeTruthy()
        expect(tableRows.length).toEqual(3)

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
        expect(component.activeTabIdx).toEqual(1)
        expect(component.openTabs.length).toEqual(1)
        expect(component.openTabs).toContain(component.zones[0])

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
        expect(component.activeTabIdx).toEqual(2)
        expect(component.openTabs.length).toEqual(2)
        expect(component.openTabs).toContain(component.zones[1])
        expect(component.openTabs).toContain(component.zones[0])
        expect(component.openTabs).not.toContain(component.zones[2])

        // Go back to first tab.
        zonesTabDe.nativeElement.click()
        fixture.detectChanges()

        // Try to open tab 1 again.
        firstRowBtns[1].nativeElement.click()
        fixture.detectChanges()
        expect(component.activeTabIdx).toEqual(1)
        expect(component.openTabs.length).toEqual(2)
        expect(component.openTabs).toContain(component.zones[0])
        expect(component.openTabs).toContain(component.zones[1])
        expect(component.openTabs).not.toContain(component.zones[2])

        const closeTabBtns = fixture.debugElement.queryAll(By.css('ul.p-tabview-nav .p-icon-wrapper'))
        expect(closeTabBtns).toBeTruthy()
        expect(closeTabBtns.length).toEqual(2)

        // Close tab 2.
        closeTabBtns[1].nativeElement.click()
        await fixture.whenStable()
        fixture.detectChanges()
        expect(component.activeTabIdx).toEqual(1)
        expect(component.openTabs.length).toEqual(1)
        expect(component.openTabs).not.toContain(component.zones[1])
        expect(component.openTabs).toContain(component.zones[0])

        // Close tab 1.
        closeTabBtns[0].nativeElement.click()
        await fixture.whenStable()
        fixture.detectChanges()
        expect(component.activeTabIdx).toEqual(0)
        expect(component.openTabs.length).toEqual(0)
        expect(component.openTabs).not.toContain(component.zones[0])
        expect(component.openTabs).not.toContain(component.zones[1])
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
        const confirmDialog = fixture.debugElement.query(By.css('.p-confirm-dialog'))
        expect(confirmDialog).toBeTruthy()
        expect(confirmDialog.nativeElement.innerText).toContain('Confirm Fetching Zones')
        expect(confirmDialog.nativeElement.innerText).toContain('Are you sure you want to continue?')

        // cancel
        const rejectBtnDe = confirmDialog.query(By.css('button.p-confirm-dialog-reject'))
        expect(rejectBtnDe).toBeTruthy()
        rejectBtnDe.nativeElement.click()
        fixture.detectChanges()
        expect(putZonesFetchSpy).toHaveBeenCalledTimes(0)
    })
})
