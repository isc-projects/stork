import { ComponentFixture, fakeAsync, TestBed, tick } from '@angular/core/testing'
import { TableModule } from 'primeng/table'
import { ZoneViewerComponent } from './zone-viewer.component'
import { ZoneRR } from '../backend/model/zoneRR'
import { By } from '@angular/platform-browser'
import { ButtonModule } from 'primeng/button'
import { TooltipModule } from 'primeng/tooltip'
import { LocaltimePipe } from '../pipes/localtime.pipe'
import { PlaceholderPipe } from '../pipes/placeholder.pipe'
import { HelpTipComponent } from '../help-tip/help-tip.component'
import { PopoverModule } from 'primeng/popover'
import { DNSService, ZoneRRs } from '../backend'
import { MessageService } from 'primeng/api'
import { of, throwError } from 'rxjs'

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
        const dnsSpy = jasmine.createSpyObj('DNSService', ['getZoneRRs', 'putZoneRRsCache'])
        const messageSpy = jasmine.createSpyObj('MessageService', ['add'])

        await TestBed.configureTestingModule({
            imports: [ButtonModule, PopoverModule, TableModule, TooltipModule],
            declarations: [HelpTipComponent, LocaltimePipe, PlaceholderPipe, ZoneViewerComponent],
            providers: [
                { provide: DNSService, useValue: dnsSpy },
                { provide: MessageService, useValue: messageSpy },
            ],
        }).compileComponents()

        dnsServiceSpy = TestBed.inject(DNSService) as jasmine.SpyObj<DNSService>
        messageServiceSpy = TestBed.inject(MessageService) as jasmine.SpyObj<MessageService>

        fixture = TestBed.createComponent(ZoneViewerComponent)
        component = fixture.componentInstance

        dnsServiceSpy.getZoneRRs.and.returnValue(of(mockZoneRRs as any))
        dnsServiceSpy.putZoneRRsCache.and.returnValue(of(mockRefreshedZoneRRs as any))

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
            100
        )
    }))

    it('should refresh zone data', fakeAsync(() => {
        component.loadRRs({ first: 1, rows: 11 })
        tick()

        expect(dnsServiceSpy.getZoneRRs).toHaveBeenCalledWith(
            component.daemonId,
            component.viewName,
            component.zoneId,
            1,
            11
        )

        expect(component.zoneData.length).toBe(2)
        expect(component.zoneData[0].name).toBe('@')
        expect(component.zoneData[0].data).toBe('ns1.example.com. admin.example.com. 2024031501 3600 900 1209600 300')
        expect(component.zoneData[1].name).toBe('www')
        expect(component.zoneData[1].data).toBe('192.0.2.1')

        // Refresh the data.
        component.refreshRRsFromDNS()
        tick()
        expect(dnsServiceSpy.putZoneRRsCache).toHaveBeenCalledWith(
            component.daemonId,
            component.viewName,
            component.zoneId,
            0,
            10
        )

        expect(component.zoneData.length).toBe(2)
        expect(component.zoneData[0].name).toBe('@')
        expect(component.zoneData[0].data).toBe('ns2.example.com. admin.example.com. 2024031501 1800 900 1209600 300')
        expect(component.zoneData[1].name).toBe('www')
        expect(component.zoneData[1].data).toBe('192.0.2.2')
    }))

    it('should handle API errors', fakeAsync(() => {
        const errorMessage = 'Failed to load zone data'
        dnsServiceSpy.getZoneRRs.and.returnValue(throwError(() => new Error(errorMessage)))

        component.loadRRs()
        tick()

        expect(messageServiceSpy.add).toHaveBeenCalledWith({
            severity: 'error',
            summary: 'Error getting zone contents',
            detail: errorMessage,
            life: 10000,
        })
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
