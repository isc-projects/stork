import { ComponentFixture, fakeAsync, TestBed, tick } from '@angular/core/testing'
import { ZoneViewerFeederComponent } from './zone-viewer-feeder.component'
import { ZoneViewerComponent } from '../zone-viewer/zone-viewer.component'
import { DNSService } from '../backend/api/dNS.service'
import { MessageService } from 'primeng/api'
import { TableModule } from 'primeng/table'
import { of, throwError } from 'rxjs'
import { ZoneRR } from '../backend/model/zoneRR'
import { ZoneRRs } from '../backend/model/zoneRRs'

describe('ZoneViewerFeederComponent', () => {
    let component: ZoneViewerFeederComponent
    let fixture: ComponentFixture<ZoneViewerFeederComponent>
    let dnsServiceSpy: jasmine.SpyObj<DNSService>
    let messageServiceSpy: jasmine.SpyObj<MessageService>

    const mockZoneRRs: ZoneRRs = {
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
        const dnsSpy = jasmine.createSpyObj('DNSService', ['getZoneRRs'])
        const messageSpy = jasmine.createSpyObj('MessageService', ['add'])

        await TestBed.configureTestingModule({
            imports: [TableModule],
            declarations: [ZoneViewerFeederComponent, ZoneViewerComponent],
            providers: [
                { provide: DNSService, useValue: dnsSpy },
                { provide: MessageService, useValue: messageSpy },
            ],
        }).compileComponents()

        dnsServiceSpy = TestBed.inject(DNSService) as jasmine.SpyObj<DNSService>
        messageServiceSpy = TestBed.inject(MessageService) as jasmine.SpyObj<MessageService>

        fixture = TestBed.createComponent(ZoneViewerFeederComponent)
        component = fixture.componentInstance

        // Set required inputs
        component.daemonId = 1
        component.viewName = 'default'
        component.zoneId = 123

        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should initialize with empty zone data', () => {
        expect(component.zoneData.items).toEqual([])
    })

    it('should not load data when inactive', () => {
        component.active = false
        expect(dnsServiceSpy.getZoneRRs).not.toHaveBeenCalled()
    })

    it('should load data when activated', fakeAsync(() => {
        dnsServiceSpy.getZoneRRs.and.returnValue(of(mockZoneRRs as any))

        component.active = true
        tick()

        expect(dnsServiceSpy.getZoneRRs).toHaveBeenCalledWith(component.daemonId, component.viewName, component.zoneId)
    }))

    it('should load data only once when activated multiple times', fakeAsync(() => {
        dnsServiceSpy.getZoneRRs.and.returnValue(of(mockZoneRRs as any))

        // Load the data when the component is first activated.
        component.active = true
        tick()
        expect(dnsServiceSpy.getZoneRRs).toHaveBeenCalledTimes(1)

        // Deactivate the component.
        component.active = false
        tick()
        expect(dnsServiceSpy.getZoneRRs).toHaveBeenCalledTimes(1)

        // Reactivate the component. It should use the cached data.
        component.active = true
        tick()
        expect(dnsServiceSpy.getZoneRRs).toHaveBeenCalledTimes(1)
    }))

    it('should handle API errors', fakeAsync(() => {
        const errorMessage = 'Failed to load zone data'
        dnsServiceSpy.getZoneRRs.and.returnValue(throwError(() => new Error(errorMessage)))

        component.active = true
        tick()

        expect(messageServiceSpy.add).toHaveBeenCalledWith({
            severity: 'error',
            summary: 'Error getting zone contents',
            detail: errorMessage,
            life: 10000,
        })
    }))
})
