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
import createSpyObj = jasmine.createSpyObj
import { DNSService, ZoneInventoryStates, Zones } from '../backend'
import { of } from 'rxjs'
import { HttpEventType, HttpHeaders, HttpResponse, HttpStatusCode } from '@angular/common/http'
import { ConfirmDialogModule } from 'primeng/confirmdialog'
import objectContaining = jasmine.objectContaining
import { MessageModule } from 'primeng/message'
import { ProgressBarModule } from 'primeng/progressbar'
import { SkeletonModule } from 'primeng/skeleton'
import { BrowserAnimationsModule } from '@angular/platform-browser/animations'
import { By } from '@angular/platform-browser'
import { PlaceholderPipe } from '../pipes/placeholder.pipe'

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
        ok: false,
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

        // By default, reply with 204 No content.
        getZonesFetchSpy.withArgs('response').and.returnValue(of(noContent))
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
            ],
            declarations: [ZonesPageComponent, BreadcrumbsComponent, HelpTipComponent, PlaceholderPipe],
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
            objectContaining({ summary: 'No Zones Fetch information', severity: 'info' })
        )
    })

    it('should fetch list of zones', async () => {
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
})
