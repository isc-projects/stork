import { ComponentFixture, TestBed } from '@angular/core/testing'
import { By } from '@angular/platform-browser'

import { MessageService } from 'primeng/api'

import { datetimeToLocal } from '../utils'
import { LeaseDetailBoxComponent } from './lease-detail-box.component'
import { JsonTreeRootComponent } from '../json-tree-root/json-tree-root.component'
import { JsonTreeComponent } from '../json-tree/json-tree.component'

describe('LeaseDetailBoxComponent', () => {
    let component: LeaseDetailBoxComponent
    let fixture: ComponentFixture<LeaseDetailBoxComponent>

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            imports: [LeaseDetailBoxComponent],
            providers: [MessageService],
        }).compileComponents()

        fixture = TestBed.createComponent(LeaseDetailBoxComponent)
        component = fixture.componentInstance
        await fixture.whenStable()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should display DHCPv4 lease', () => {
        fixture.componentRef.setInput('lease', {
            id: 0,
            ipAddress: '192.0.2.3',
            state: 0,
            daemonId: 1,
            daemonLabel: 'DHCPv4@localhost',
            hwAddress: '01:02:03:04:05:06',
            clientId: '51:52:53:54',
            hostname: 'faq.example.org',
            fqdnFwd: false,
            fqdnRev: true,
            localSubnetId: 123,
            cltt: 1616149050,
            validLifetime: 3600,
            userContext: { ISC: { 'client-classes': ['ALL', 'HA_primary', 'UNKNOWN'] } },
        })

        fixture.detectChanges()
        // Find the tables holding expanded information.
        const tables = fixture.debugElement.queryAll(By.css('table'))
        expect(tables.length).toBe(3)

        // Find allocation and expiration time.
        const startDate = new Date(1616149050000)
        const endDate = new Date(1616149050000 + 3600000)

        // Expected data in various tables within the expanded row.
        const expectedLeaseData: any = [
            [
                ['HW Address', '01:02:03:04:05:06'],
                ['Client Identifier', 'QRST'],
            ],
            [
                ['Kea Subnet ID', '123'],
                ['Valid Lifetime', '3600 seconds'],
                ['Allocated at', datetimeToLocal(startDate)],
                ['Expires at', datetimeToLocal(endDate)],
            ],
            [
                ['Hostname', 'faq.example.org'],
                ['Forward DDNS', 'no'],
                ['Reverse DDNS', 'yes'],
            ],
        ]

        // Second, third and, fourth tables should contain expanded lease information.
        // For each table check if the data is correct.
        let tableIndex = 0
        for (const expectedDataGroup of expectedLeaseData) {
            const rows = tables[tableIndex].queryAll(By.css('tr'))
            expect(rows.length).toBe(expectedDataGroup.length)

            // For each table row, compare its contents with the expected data.
            let i = 0
            for (const row of rows) {
                expect(row.children.length).toBe(2)
                expect(row.children[0].nativeElement.innerText).toBe(expectedDataGroup[i][0] + ':')
                expect(row.children[1].nativeElement.innerText).toContain(expectedDataGroup[i][1])
                i++
            }
            tableIndex++
        }

        // Test User Context JSON tree content.
        const tree = fixture.debugElement.queryAll(By.directive(JsonTreeRootComponent))
        expect(tree).not.toBeNull()
        expect(Object.keys(tree).length).toBe(1)
        const treeComponent = tree[0].componentInstance as JsonTreeComponent
        expect(treeComponent).not.toBeNull()
        expect(treeComponent.value).not.toBeNull()
        expect(Object.keys(treeComponent.value).length).toBe(1)
        expect(treeComponent.value['ISC']).not.toBeNull()
        expect(Object.keys(treeComponent.value['ISC']).length).toBe(1)
        expect(treeComponent.value['ISC']['client-classes']).not.toBeNull()
        expect(treeComponent.value['ISC']['client-classes'].length).toBe(3)
        expect(treeComponent.value['ISC']['client-classes'][0]).toBe('ALL')
        expect(treeComponent.value['ISC']['client-classes'][1]).toBe('HA_primary')
        expect(treeComponent.value['ISC']['client-classes'][2]).toBe('UNKNOWN')
    })

    // TODO: copy other two tests here
    it('should display declined DHCPv4 lease', () => {
        fixture.componentRef.setInput('lease', {
            id: 0,
            ipAddress: '192.0.2.3',
            state: 1,
            daemonId: 1,
            daemonLabel: 'DHCPv4@localhost',
            localSubnetId: 123,
            cltt: 1616149050,
            validLifetime: 3600,
        })

        fixture.detectChanges()
        // There should be one table holding the expanded information.
        // In particular, there should be no table with client identifiers
        // because they are not present for a declined lease.
        const tables = fixture.debugElement.queryAll(By.css('table'))
        expect(tables.length).toBe(1)

        // Find allocation and expiration time.
        const startDate = new Date(1616149050000)
        const endDate = new Date(1616149050000 + 3600000)

        // Expected data within the expanded row.
        const expectedLeaseData: any = [
            ['Kea Subnet ID', '123'],
            ['Valid Lifetime', '3600 seconds'],
            ['Allocated at', datetimeToLocal(startDate)],
            ['Expires at', datetimeToLocal(endDate)],
        ]

        // Find rows.
        const rows = tables[0].queryAll(By.css('tr'))
        expect(rows.length).toBe(4)

        // For each table row, compare its contents with the expected data.
        let i = 0
        for (const row of rows) {
            console.log(row.children)
            expect(row.children.length).toBe(2)
            expect(row.children[0].nativeElement.innerText).toBe(expectedLeaseData[i][0] + ':')
            expect(row.children[1].nativeElement.innerText).toBe(expectedLeaseData[i][1])
            i++
        }

        // Test User Context JSON tree content.
        const tree = fixture.debugElement.queryAll(By.directive(JsonTreeRootComponent))
        expect(tree).not.toBeNull()
        expect(Object.keys(tree).length).toBe(0)
    })

    it('should display an IA_NA DHCPv6 lease', () => {
        fixture.componentRef.setInput('lease', {
            id: 1,
            ipAddress: '2001:db8:1::1',
            leaseType: 'IA_NA',
            state: 1,
            daemonId: 2,
            daemonLabel: 'DHCPv6@localhost',
            hwAddress: '01:02:03:04:05:06',
            duid: '01:02:03:04',
            hostname: 'faq.example.org',
            fqdnFwd: true,
            fqdnRev: false,
            localSubnetId: 234,
            iaid: 12,
            cltt: 1616149050,
            preferredLifetime: 900,
            validLifetime: 1800,
            userContext: { ISC: { 'client-classes': ['ALL', 'HA_primary', 'UNKNOWN'] } },
        })

        fixture.detectChanges()

        // Find the table holding expanded information.
        const tables = fixture.debugElement.queryAll(By.css('table'))
        expect(tables.length).toBe(3)

        // Find allocation and expiration time.
        const startDate = new Date(1616149050000)
        const endDate = new Date(1616149050000 + 1800000)

        let expectedLeaseData: any = [
            [
                ['HW Address', '01:02:03:04:05:06'],
                ['DUID', '\\0x01\\0x02\\0x03\\0x04'],
            ],
            [
                ['Kea Subnet ID', '234'],
                ['IAID', '12'],
                ['Preferred Lifetime', '900 seconds'],
                ['Valid Lifetime', '1800 seconds'],
                ['Allocated at', datetimeToLocal(startDate)],
                ['Expires at', datetimeToLocal(endDate)],
            ],
            [
                ['Hostname', 'faq.example.org'],
                ['Forward DDNS', 'yes'],
                ['Reverse DDNS', 'no'],
            ],
        ]

        let tableIndex = 0
        for (const expectedDataGroup of expectedLeaseData) {
            const rows = tables[tableIndex].queryAll(By.css('tr'))
            expect(rows.length).toBe(expectedDataGroup.length)

            // For each table row, compare its contents with the expected data.
            let i = 0
            for (const row of rows) {
                expect(row.children.length).toBe(2)
                expect(row.children[0].nativeElement.innerText).toBe(expectedDataGroup[i][0] + ':')
                expect(row.children[1].nativeElement.innerText).toContain(expectedDataGroup[i][1])
                i++
            }
            tableIndex++
        }

        // Test User Context JSON tree content.
        const tree = fixture.debugElement.queryAll(By.directive(JsonTreeRootComponent))
        expect(tree).not.toBeNull()
        expect(Object.keys(tree).length).toBe(1)
        const treeComponent = tree[0].componentInstance as JsonTreeComponent
        expect(treeComponent).not.toBeNull()
        expect(treeComponent.value).not.toBeNull()
        expect(Object.keys(treeComponent.value).length).toBe(1)
        expect(treeComponent.value['ISC']).not.toBeNull()
        expect(Object.keys(treeComponent.value['ISC']).length).toBe(1)
        expect(treeComponent.value['ISC']['client-classes']).not.toBeNull()
        expect(treeComponent.value['ISC']['client-classes'].length).toBe(3)
        expect(treeComponent.value['ISC']['client-classes'][0]).toBe('ALL')
        expect(treeComponent.value['ISC']['client-classes'][1]).toBe('HA_primary')
        expect(treeComponent.value['ISC']['client-classes'][2]).toBe('UNKNOWN')
    })

    it('should display an IA_PD DHCPv6 lease', () => {
        fixture.componentRef.setInput('lease', {
            id: 2,
            ipAddress: '3000::',
            prefixLength: 64,
            leaseType: 'IA_PD',
            state: 2,
            daemonId: 2,
            daemonLabel: 'DHCPv6@localhost',
            duid: '01:02:03:05',
            localSubnetId: 345,
            iaid: 13,
            cltt: 1616149050,
            preferredLifetime: 900,
            validLifetime: 1800,
        })

        fixture.detectChanges()

        // Find the table holding expanded information.
        const tables = fixture.debugElement.queryAll(By.css('table'))
        expect(tables.length).toBe(2)

        // Find allocation and expiration time.
        const startDate = new Date(1616149050000)
        const endDate = new Date(1616149050000 + 1800000)

        let expectedLeaseData: any = [
            [['DUID', '\\0x01\\0x02\\0x03\\0x05']],
            [
                ['Kea Subnet ID', '345'],
                ['IAID', '13'],
                ['Preferred Lifetime', '900 seconds'],
                ['Valid Lifetime', '1800 seconds'],
                ['Allocated at', datetimeToLocal(startDate)],
                ['Expires at', datetimeToLocal(endDate)],
            ],
        ]

        // Fifth and sixth table should contain expanded lease information.
        // For each table check if the data is correct.
        let tableIndex: number = 0
        for (const expectedDataGroup of expectedLeaseData) {
            const rows = tables[tableIndex].queryAll(By.css('tr'))
            expect(rows.length).toBe(expectedDataGroup.length)

            // For each table row, compare its contents with the expected data.
            let i = 0
            for (const row of rows) {
                expect(row.children.length).toBe(2)
                expect(row.children[0].nativeElement.innerText).toBe(expectedDataGroup[i][0] + ':')
                expect(row.children[1].nativeElement.innerText).toContain(expectedDataGroup[i][1])
                i++
            }
            tableIndex++
        }
    })
})
