import { ComponentFixture, TestBed } from '@angular/core/testing'
import { TableModule } from 'primeng/table'
import { ZoneViewerComponent } from './zone-viewer.component'
import { ZoneRR } from '../backend/model/zoneRR'
import { By } from '@angular/platform-browser'
import { ButtonModule } from 'primeng/button'
import { TooltipModule } from 'primeng/tooltip'
import { DividerModule } from 'primeng/divider'
import { ProgressSpinnerModule } from 'primeng/progressspinner'
import { LocaltimePipe } from '../pipes/localtime.pipe'
import { PlaceholderPipe } from '../pipes/placeholder.pipe'
import { HelpTipComponent } from '../help-tip/help-tip.component'
import { PopoverModule } from 'primeng/popover'

describe('ZoneViewerComponent', () => {
    let component: ZoneViewerComponent
    let fixture: ComponentFixture<ZoneViewerComponent>

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            imports: [ButtonModule, DividerModule, PopoverModule, ProgressSpinnerModule, TableModule, TooltipModule],
            declarations: [HelpTipComponent, LocaltimePipe, PlaceholderPipe, ZoneViewerComponent],
        }).compileComponents()

        fixture = TestBed.createComponent(ZoneViewerComponent)
        component = fixture.componentInstance
        // Override the default flex scroller height with an explicit value.
        // It disables the flex layout. Using the flex layout requires the parent
        // to use flex display. It doesn't work well with the unit tests.
        component.scrollHeight = '400px'
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should transform SOA record correctly', () => {
        const soaRecord: ZoneRR = {
            name: 'example.com.',
            ttl: 3600,
            rrClass: 'IN',
            rrType: 'SOA',
            data: 'ns1.example.com. admin.example.com. 2024031501 3600 900 1209600 300',
        }

        const result = component['_transformZoneRR'](soaRecord, true)
        expect(result).toEqual({
            ...soaRecord,
            name: '@',
        })
        expect(component['_zoneName']).toBe('example.com.')
    })

    it('should skip last SOA record', () => {
        const soaRecord: ZoneRR = {
            name: 'example.com.',
            ttl: 3600,
            rrClass: 'IN',
            rrType: 'SOA',
            data: 'ns1.example.com. admin.example.com. 2024031501 3600 900 1209600 300',
        }
        const result = component['_transformZoneRR'](soaRecord, false)
        expect(result).toBeNull()
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
        component['_transformZoneRR'](soaRecord, true)

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

        const result1 = component['_transformZoneRR'](record1, false)
        const result2 = component['_transformZoneRR'](record2, false)

        expect(result1).not.toBeNull()
        expect(result2).not.toBeNull()

        expect(result1.name).toBe('www')
        expect(result2.name).toBe('')
    })

    it('should transform all items on initialization', () => {
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

        component.data = { items }

        expect(component.data.items.length).toBe(4)
        expect(component.data.items[0].name).toBe('@')
        expect(component.data.items[1].name).toBe('www')
        expect(component.data.items[2].name).toBe('')
        expect(component.data.items[3].name).toBe('www2')
    })

    it('should display the records in the table', () => {
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
        ]

        component.data = { items }
        fixture.detectChanges()

        const rows = fixture.debugElement.queryAll(By.css('tr'))
        expect(rows.length).toBe(items.length)

        // First record.
        let cells = rows[0].queryAll(By.css('td'))
        expect(cells.length).toBe(5)
        expect(cells[0].nativeElement.innerText).toBe('@')
        expect(cells[1].nativeElement.innerText).toBe('3600')
        expect(cells[2].nativeElement.innerText).toBe('IN')
        expect(cells[3].nativeElement.innerText).toBe('SOA')
        expect(cells[4].nativeElement.innerText).toBe(
            'ns1.example.com. admin.example.com. 2024031501 3600 900 1209600 300'
        )

        // Second record.
        cells = rows[1].queryAll(By.css('td'))
        expect(cells.length).toBe(5)
        expect(cells[0].nativeElement.innerText).toBe('www')
        expect(cells[1].nativeElement.innerText).toBe('3600')
        expect(cells[2].nativeElement.innerText).toBe('IN')
        expect(cells[3].nativeElement.innerText).toBe('A')
        expect(cells[4].nativeElement.innerText).toBe('192.0.2.1')
    })

    it('should display the loading spinner when loading is true', () => {
        component.loading = true
        fixture.detectChanges()

        // Spinner should be displayed while loading.
        const spinner = fixture.debugElement.query(By.css('p-progressSpinner'))
        expect(spinner).toBeTruthy()

        // Table should not be displayed while loading.
        const table = fixture.debugElement.query(By.css('p-table'))
        expect(table).toBeFalsy()
    })
})
