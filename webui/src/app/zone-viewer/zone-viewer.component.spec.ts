import { ComponentFixture, fakeAsync, TestBed, tick } from '@angular/core/testing'
import { ZoneViewerComponent } from './zone-viewer.component'
import { ZoneRR } from '../backend/model/zoneRR'
import { By } from '@angular/platform-browser'
import { DNSService, ZoneRRs } from '../backend'
import { FilterMetadata, MessageService } from 'primeng/api'
import { of, throwError } from 'rxjs'
import { provideHttpClient, withInterceptorsFromDi } from '@angular/common/http'
import { provideHttpClientTesting } from '@angular/common/http/testing'
import { provideNoopAnimations } from '@angular/platform-browser/animations'

describe('ZoneViewerComponent', () => {
    let component: ZoneViewerComponent
    let fixture: ComponentFixture<ZoneViewerComponent>
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

    const mockRefreshedZoneRRs: ZoneRRs = {
        items: [
            {
                name: 'example.com.',
                ttl: 1800,
                rrClass: 'IN',
                rrType: 'SOA',
                data: 'ns2.example.com. admin.example.com. 2024031501 1800 900 1209600 300',
            } as ZoneRR,
            {
                name: 'www.example.com.',
                ttl: 1800,
                rrClass: 'IN',
                rrType: 'A',
                data: '192.0.2.2',
            } as ZoneRR,
        ],
    }

    beforeEach(async () => {
        await TestBed.compileComponents()

        dnsServiceSpy = jasmine.createSpyObj('DNSService', ['getZoneRRs', 'putZoneRRsCache'])
        dnsServiceSpy.getZoneRRs.and.returnValue(of(mockZoneRRs as any))
        dnsServiceSpy.putZoneRRsCache.and.returnValue(of(mockRefreshedZoneRRs as any))
        const messageSpy = jasmine.createSpyObj('MessageService', ['add'])

        await TestBed.configureTestingModule({
            providers: [
                { provide: DNSService, useValue: dnsServiceSpy },
                { provide: MessageService, useValue: messageSpy },
                provideNoopAnimations(),
                provideHttpClient(withInterceptorsFromDi()),
                provideHttpClientTesting(),
            ],
        }).compileComponents()

        messageServiceSpy = TestBed.inject(MessageService) as jasmine.SpyObj<MessageService>
        fixture = TestBed.createComponent(ZoneViewerComponent)
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

    it('should load data', fakeAsync(() => {
        component.loadRRs({ first: 10, rows: 100 })
        tick()
        expect(dnsServiceSpy.getZoneRRs).toHaveBeenCalledWith(
            component.daemonId,
            component.viewName,
            component.zoneId,
            10,
            100,
            null,
            null
        )
    }))

    it('should refresh zone data', fakeAsync(() => {
        // Set custom pagination values.
        component.table.first = 1
        component.table.rows = 11
        component.loadRRs(component.table.createLazyLoadMetadata())
        tick()
        fixture.detectChanges()

        // Make sure that the correct parameters are passed to the API.
        expect(dnsServiceSpy.getZoneRRs).toHaveBeenCalledWith(
            component.daemonId,
            component.viewName,
            component.zoneId,
            1,
            11,
            null,
            null
        )

        // Make sure that the pagination values are preserved.
        expect(component.table.first).toBe(1)
        expect(component.table.rows).toBe(11)

        // Make sure that the data is loaded correctly.
        expect(component.zoneData.length).toBe(2)
        expect(component.zoneData[0].name).toBe('@')
        expect(component.zoneData[0].data).toBe('ns1.example.com. admin.example.com. 2024031501 3600 900 1209600 300')
        expect(component.zoneData[1].name).toBe('www')
        expect(component.zoneData[1].data).toBe('192.0.2.1')

        // Refresh the data. Make sure that the offset was reset to 0.
        component.refreshRRsFromDNS()
        tick()
        fixture.detectChanges()

        // Make sure that the correct parameters are passed to the API.
        expect(dnsServiceSpy.putZoneRRsCache).toHaveBeenCalledWith(
            component.daemonId,
            component.viewName,
            component.zoneId,
            0,
            11,
            null,
            null
        )

        // Make sure that the data was refreshed correctly.
        expect(component.zoneData.length).toBe(2)
        expect(component.zoneData[0].name).toBe('@')
        expect(component.zoneData[0].data).toBe('ns2.example.com. admin.example.com. 2024031501 1800 900 1209600 300')
        expect(component.zoneData[1].name).toBe('www')
        expect(component.zoneData[1].data).toBe('192.0.2.2')
    }))

    it('should filter zone data by type', fakeAsync(() => {
        component.table.filters['rrType'] = {
            value: ['SOA', 'A'],
            matchMode: 'contains',
        } as FilterMetadata
        component.filterRRsTable(['SOA', 'A'] as any, component.table.filters['rrType'])
        tick(300)
        fixture.detectChanges()

        // Make sure that the correct parameters are passed to the API.
        expect(dnsServiceSpy.getZoneRRs).toHaveBeenCalledWith(
            component.daemonId,
            component.viewName,
            component.zoneId,
            0,
            10,
            ['SOA', 'A'],
            null
        )
    }))

    it('should filter zone data by text', fakeAsync(() => {
        component.table.filters['text'] = {
            value: 'example.com',
            matchMode: 'contains',
        } as FilterMetadata
        component.filterRRsTable('example.com', component.table.filters['text'])
        tick(300)
        fixture.detectChanges()

        // Make sure that the correct parameters are passed to the API.
        expect(dnsServiceSpy.getZoneRRs).toHaveBeenCalledWith(
            component.daemonId,
            component.viewName,
            component.zoneId,
            0,
            10,
            null,
            'example.com'
        )
    }))

    it('should filter zone data by type when refreshing zone data', fakeAsync(() => {
        component.table.filters['rrType'] = {
            value: ['A', 'AAAA'],
            matchMode: 'contains',
        } as FilterMetadata
        component.refreshRRsFromDNS()
        tick()
        fixture.detectChanges()

        // Make sure that the correct parameters are passed to the API.
        expect(dnsServiceSpy.putZoneRRsCache).toHaveBeenCalledWith(
            component.daemonId,
            component.viewName,
            component.zoneId,
            0,
            10,
            ['A', 'AAAA'],
            null
        )
    }))

    it('should filter zone data by text when refreshing zone data', fakeAsync(() => {
        component.table.first = 1
        component.table.filters['text'] = {
            value: 'example.com',
            matchMode: 'contains',
        } as FilterMetadata
        component.refreshRRsFromDNS()
        tick()
        fixture.detectChanges()

        // Make sure that the correct parameters are passed to the API.
        expect(dnsServiceSpy.putZoneRRsCache).toHaveBeenCalledWith(
            component.daemonId,
            component.viewName,
            component.zoneId,
            0,
            10,
            null,
            'example.com'
        )
    }))

    it('should handle API errors while getting zone data', fakeAsync(() => {
        // Set up the error message.
        const errorMessage = 'Failed to load zone data'
        dnsServiceSpy.getZoneRRs.and.returnValue(throwError(() => new Error(errorMessage)))
        // Make sure that the viewer error event is emitted.
        spyOn(component.viewerError, 'emit')

        // Load the data.
        component.loadRRs()
        tick()
        fixture.detectChanges()

        // Make sure that the error message is displayed.
        expect(messageServiceSpy.add).toHaveBeenCalledWith({
            severity: 'error',
            summary: 'Error getting zone contents',
            detail: errorMessage,
            life: 10000,
        })
        // Make sure that the viewer error event is emitted.
        expect(component.viewerError.emit).toHaveBeenCalled()
    }))

    it('should handle API errors while refreshing zone data', fakeAsync(() => {
        // Set up the error message.
        const errorMessage = 'Failed to refresh zone data'
        dnsServiceSpy.putZoneRRsCache.and.returnValue(throwError(() => new Error(errorMessage)))
        // Make sure that the viewer error event is emitted.
        spyOn(component.viewerError, 'emit')

        // Refresh the data.
        component.refreshRRsFromDNS()
        tick()
        fixture.detectChanges()

        // Make sure that the error message is displayed.
        expect(messageServiceSpy.add).toHaveBeenCalledWith({
            severity: 'error',
            summary: 'Error refreshing zone contents from DNS server',
            detail: errorMessage,
            life: 10000,
        })
        // Make sure that the viewer error event is emitted.
        expect(component.viewerError.emit).toHaveBeenCalled()
    }))

    it('should transform SOA record correctly', () => {
        const soaRecord: ZoneRR = {
            name: 'example.com.',
            ttl: 3600,
            rrClass: 'IN',
            rrType: 'SOA',
            data: 'ns1.example.com. admin.example.com. 2024031501 3600 900 1209600 300',
        }

        const result = component['_transformZoneRR'](soaRecord)
        expect(result).toEqual({
            ...soaRecord,
            name: '@',
        })
        expect(component['_zoneName']).toBe('example.com.')
    })

    it('should handle repeated names', () => {
        // Set up the zone name with a SOA record.
        const soaRecord: ZoneRR = {
            name: 'example.com.',
            ttl: 3600,
            rrClass: 'IN',
            rrType: 'SOA',
            data: 'ns1.example.com. admin.example.com. 2024031501 3600 900 1209600 300',
        }
        component['_transformZoneRR'](soaRecord)

        // First record following the SOA record should have the
        // zone name stripped from the name.
        const record1: ZoneRR = {
            name: 'www.example.com.',
            ttl: 3600,
            rrClass: 'IN',
            rrType: 'A',
            data: '192.0.2.1',
        }
        // Second record with the same name should have an empty name.
        const record2: ZoneRR = {
            name: 'www.example.com.',
            ttl: 3600,
            rrClass: 'IN',
            rrType: 'A',
            data: '192.0.2.2',
        }

        const result1 = component['_transformZoneRR'](record1)
        const result2 = component['_transformZoneRR'](record2)

        expect(result1).not.toBeNull()
        expect(result2).not.toBeNull()

        expect(result1.name).toBe('www')
        expect(result2.name).toBe('')
    })

    it('should transform all items on initialization', fakeAsync(() => {
        const items: ZoneRR[] = [
            {
                name: 'example.com.',
                ttl: 3600,
                rrClass: 'IN',
                rrType: 'SOA',
                data: 'ns1.example.com. admin.example.com. 2024031501 3600 900 1209600 300',
            },
            {
                name: 'www.example.com.',
                ttl: 3600,
                rrClass: 'IN',
                rrType: 'A',
                data: '192.0.2.1',
            },
            {
                name: 'www.example.com.',
                ttl: 3600,
                rrClass: 'IN',
                rrType: 'AAAA',
                data: '2001:db8:85a3::8a2e:370:7334',
            },
            {
                name: 'www2.example.com.',
                ttl: 3600,
                rrClass: 'IN',
                rrType: 'A',
                data: '192.0.2.2',
            },
            {
                name: 'example.com.',
                ttl: 3600,
                rrClass: 'IN',
                rrType: 'SOA',
                data: 'ns1.example.com. admin.example.com. 2024031501 3600 900 1209600 300',
            },
        ]

        dnsServiceSpy.getZoneRRs.and.returnValue(of({ items, total: 5 } as any))
        component.loadRRs()
        tick()

        expect(component.zoneData.length).toBe(5)
        expect(component.zoneData[0].name).toBe('@')
        expect(component.zoneData[1].name).toBe('www')
        expect(component.zoneData[2].name).toBe('')
        expect(component.zoneData[3].name).toBe('www2')
        expect(component.zoneData[4].name).toBe('@')
    }))

    it('should display the records in the table', fakeAsync(() => {
        component.loadRRs()
        tick()
        fixture.detectChanges()

        const rows = fixture.debugElement.queryAll(By.css('tr'))
        expect(rows.length).toBe(3)

        // The first row contains a header.
        let cells = rows[0].queryAll(By.css('th'))
        expect(cells.length).toBe(5)

        // First record.
        cells = rows[1].queryAll(By.css('td'))
        expect(cells.length).toBe(5)
        expect(cells[0].nativeElement.innerText).toBe('@')
        expect(cells[1].nativeElement.innerText).toBe('3600')
        expect(cells[2].nativeElement.innerText).toBe('IN')
        expect(cells[3].nativeElement.innerText).toBe('SOA')
        expect(cells[4].nativeElement.innerText).toBe(
            'ns1.example.com. admin.example.com. 2024031501 3600 900 1209600 300'
        )

        // Second record.
        cells = rows[2].queryAll(By.css('td'))
        expect(cells.length).toBe(5)
        expect(cells[0].nativeElement.innerText).toBe('www')
        expect(cells[1].nativeElement.innerText).toBe('3600')
        expect(cells[2].nativeElement.innerText).toBe('IN')
        expect(cells[3].nativeElement.innerText).toBe('A')
        expect(cells[4].nativeElement.innerText).toBe('192.0.2.1')
    }))
})
