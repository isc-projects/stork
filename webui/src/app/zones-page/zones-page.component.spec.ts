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

describe('ZonesPageComponent', () => {
    let component: ZonesPageComponent
    let fixture: ComponentFixture<ZonesPageComponent>
    let dnsApi: any
    let getZonesFetchSpy: any
    let putZonesFetchSpy: any
    let getZonesSpy: any

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

    beforeEach(async () => {
        dnsApi = createSpyObj('DNSService', ['getZonesFetch', 'putZonesFetch', 'getZones'])
        getZonesFetchSpy = dnsApi.getZonesFetch
        putZonesFetchSpy = dnsApi.putZonesFetch
        getZonesSpy = dnsApi.getZones

        // By default, reply with 204 No content.
        getZonesFetchSpy.withArgs('response').and.returnValue(of(noContent))
        // By default, emits null response and completes without any error.
        putZonesFetchSpy.and.returnValue(of(null))
        // By default, returns no Zones.
        getZonesSpy.and.returnValue(of(noZones))

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
            ],
            declarations: [ZonesPageComponent, BreadcrumbsComponent, HelpTipComponent],
            providers: [MessageService, { provide: DNSService, useValue: dnsApi }, ConfirmationService],
        }).compileComponents()

        fixture = TestBed.createComponent(ZonesPageComponent)
        component = fixture.componentInstance
        fixture.detectChanges()

        // Do not save table state between tests, because that makes tests unstable.
        spyOn(component.zonesTable, 'saveState').and.callFake(() => {})

        spyOn(component, 'wasZoneFetchSent').and.returnValue(true)
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should call dns apis on init', async () => {
        expect(component.zonesLoading).toBeTrue()
        expect(component.zonesFetchStatesLoading).toBeTrue()
        await fixture.whenStable()
        fixture.detectChanges()
        await fixture.whenStable()
        fixture.detectChanges()
        expect(component.zonesLoading).withContext('Zones table data loading should be done').toBeFalse()
        expect(component.zonesFetchStatesLoading)
            .withContext('Zones Fetch Status table data loading should be done')
            .toBeFalse()
        expect(getZonesSpy).toHaveBeenCalledTimes(1)
        expect(getZonesFetchSpy).toHaveBeenCalledTimes(1)
        expect(putZonesFetchSpy).toHaveBeenCalledTimes(0)
    })
})
