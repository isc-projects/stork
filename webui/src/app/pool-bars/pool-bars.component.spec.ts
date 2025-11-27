import { ComponentFixture, TestBed } from '@angular/core/testing'

import { PoolBarsComponent } from './pool-bars.component'
import { DelegatedPrefixPool, Pool } from '../backend'
import { SubnetWithUniquePools } from '../subnets'

describe('PoolBarsComponent', () => {
    let component: PoolBarsComponent
    let fixture: ComponentFixture<PoolBarsComponent>

    beforeEach(async () => {
        await TestBed.compileComponents()

        fixture = TestBed.createComponent(PoolBarsComponent)
        component = fixture.componentInstance
        component.source = {}
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should group address pools by their IDs and families and sort pools in groups', () => {
        // Arrange
        const addressPools: Pool[] = [
            // The IPv4 pools with a default pool ID.
            { pool: '10.0.0.21-10.0.0.30' },
            { pool: '10.0.0.1-10.0.0.10' },
            { pool: '10.0.0.11-10.0.0.20' },
            // An IPv4 pool with a unique pool ID.
            { pool: '11.2.0.1-11.2.0.10', keaConfigPoolParameters: { poolID: 2 } },
            // The IPv4 pools with a custom pool ID.
            { pool: '9.1.0.1-9.1.0.10', keaConfigPoolParameters: { poolID: 1 } },
            { pool: '9.1.0.21-9.1.0.30', keaConfigPoolParameters: { poolID: 1 } },
            { pool: '9.1.0.11-9.1.0.20', keaConfigPoolParameters: { poolID: 1 } },
            // The IPv6 pools with a default pool ID.
            { pool: '2001:db8:0::1-2001:db8:0::10' },
            { pool: '2001:db8:0::11-2001:db8:0::20' },
            { pool: '2001:db8:0::21-2001:db8:0::30' },
            // An IPv6 pool with a unique pool ID.
            { pool: '2001:db8:2::1-2001:db8:2::10', keaConfigPoolParameters: { poolID: 2 } },
            // The IPv6 pools with a custom pool ID.
            { pool: '2001:db8:1::1-2001:db8:1::10', keaConfigPoolParameters: { poolID: 1 } },
            { pool: '2001:db8:1::11-2001:db8:1::20', keaConfigPoolParameters: { poolID: 1 } },
            { pool: '2001:db8:1::21-2001:db8:1::30', keaConfigPoolParameters: { poolID: 1 } },
        ]

        // Act
        component.source = { pools: addressPools }
        component.ngOnInit()

        // Assert
        expect(component.addressPoolsGrouped.length).toBe(6)
        // The IPv4 pools take precedence over the IPv6 pools.
        // The group with a single pool takes precedence over the group with multiple pools.
        expect(component.addressPoolsGrouped[0][0]).toBe(2)
        expect(component.addressPoolsGrouped[0][1].length).toBe(1)
        expect(component.addressPoolsGrouped[0][1][0].pool).toBe('11.2.0.1-11.2.0.10')
        // The groups with multiple pools are sorted by their pool ID.
        // The first group is the default pool ID.
        // The pools are sorted by their ranges.
        expect(component.addressPoolsGrouped[1][0]).toBe(0)
        expect(component.addressPoolsGrouped[1][1].length).toBe(3)
        expect(component.addressPoolsGrouped[1][1][0].pool).toBe('10.0.0.1-10.0.0.10')
        expect(component.addressPoolsGrouped[1][1][1].pool).toBe('10.0.0.11-10.0.0.20')
        expect(component.addressPoolsGrouped[1][1][2].pool).toBe('10.0.0.21-10.0.0.30')
        // The next group has a custom pool ID and multiple pools.
        expect(component.addressPoolsGrouped[2][0]).toBe(1)
        expect(component.addressPoolsGrouped[2][1].length).toBe(3)
        expect(component.addressPoolsGrouped[2][1][0].pool).toBe('9.1.0.1-9.1.0.10')
        expect(component.addressPoolsGrouped[2][1][1].pool).toBe('9.1.0.11-9.1.0.20')
        expect(component.addressPoolsGrouped[2][1][2].pool).toBe('9.1.0.21-9.1.0.30')

        // The IPv6 pools are sorted same way.
        expect(component.addressPoolsGrouped[3][0]).toBe(2)
        expect(component.addressPoolsGrouped[3][1].length).toBe(1)
        expect(component.addressPoolsGrouped[3][1][0].pool).toBe('2001:db8:2::1-2001:db8:2::10')
        expect(component.addressPoolsGrouped[4][0]).toBe(0)
        expect(component.addressPoolsGrouped[4][1].length).toBe(3)
        expect(component.addressPoolsGrouped[4][1][0].pool).toBe('2001:db8:0::1-2001:db8:0::10')
        expect(component.addressPoolsGrouped[4][1][1].pool).toBe('2001:db8:0::11-2001:db8:0::20')
        expect(component.addressPoolsGrouped[4][1][2].pool).toBe('2001:db8:0::21-2001:db8:0::30')
        expect(component.addressPoolsGrouped[5][0]).toBe(1)
        expect(component.addressPoolsGrouped[5][1].length).toBe(3)
    })

    it('should group delegated prefix pools by their IDs and sort pools in groups', () => {
        // Arrange
        const pdPools: DelegatedPrefixPool[] = [
            // The pools with a default pool ID.
            { prefix: '2001:db8:0:2::/64', delegatedLength: 80 },
            { prefix: '2001:db8:0:1::/64', delegatedLength: 80 },
            { prefix: '2001:db8:0:0::/64', delegatedLength: 80 },
            // A pool with a unique pool ID.
            { prefix: '2001:db8:3:1::/64', delegatedLength: 80, keaConfigPoolParameters: { poolID: 2 } },
            { prefix: '2001:db8:2:0::/64', delegatedLength: 80, keaConfigPoolParameters: { poolID: 3 } },
            // A pools with unique pool IDs and overlapping prefixes.
            { prefix: '2001:db8:4:0::/80', delegatedLength: 96, keaConfigPoolParameters: { poolID: 4 } },
            { prefix: '2001:db8:4:0::/64', delegatedLength: 80, keaConfigPoolParameters: { poolID: 5 } },
            // The pools with a custom pool ID.
            { prefix: '2001:db8:6:0::/64', delegatedLength: 80, keaConfigPoolParameters: { poolID: 6 } },
            { prefix: '2001:db8:6:3::/64', delegatedLength: 80, keaConfigPoolParameters: { poolID: 6 } },
            { prefix: '2001:db8:6:2::/64', delegatedLength: 80, keaConfigPoolParameters: { poolID: 6 } },
            { prefix: '2001:db8:6:1::/64', delegatedLength: 80, keaConfigPoolParameters: { poolID: 6 } },
            // The overlapping prefixes with the same delegated length are sorted by their excluded prefixes.
            // I'm not sure if this is a case allowed by the Kea DHCPv6 server but we are prepared for it.
            { prefix: '2001:db8:6:3::/64', delegatedLength: 80, keaConfigPoolParameters: { poolID: 6 } },
            {
                prefix: '2001:db8:6:3::/64',
                delegatedLength: 80,
                excludedPrefix: '2001:db8:6:3::/96',
                keaConfigPoolParameters: { poolID: 6 },
            },
            {
                prefix: '2001:db8:6:3::/64',
                delegatedLength: 80,
                excludedPrefix: '2001:db8:6:3::/80',
                keaConfigPoolParameters: { poolID: 6 },
            },
            // Non-canonical prefixes should be handled as well. I'm unsure if this is a case allowed by the Kea DHCPv6
            // server but we are prepared for it.
            {
                prefix: '2001:db8:7:2:2:2:2:2::/64',
                delegatedLength: 80,
                keaConfigPoolParameters: { poolID: 7 },
            },
            {
                prefix: '2001:db8:7:1:1:1:1:1::/64',
                delegatedLength: 80,
                keaConfigPoolParameters: { poolID: 7 },
            },
            {
                prefix: '2001:db8:7:3::/64',
                delegatedLength: 80,
                keaConfigPoolParameters: { poolID: 7 },
            },
        ]

        // Act
        component.source = { prefixDelegationPools: pdPools }
        component.ngOnInit()

        // Assert
        expect(component.pdPoolsGrouped.length).toBe(7)
        // The groups with a single pool takes precedence over the group with multiple pools.
        // They are sorted by their prefixes.
        expect(component.pdPoolsGrouped[0][0]).toBe(3)
        expect(component.pdPoolsGrouped[0][1].length).toBe(1)
        expect(component.pdPoolsGrouped[0][1][0].prefix).toBe('2001:db8:2:0::/64')
        expect(component.pdPoolsGrouped[1][0]).toBe(2)
        expect(component.pdPoolsGrouped[1][1].length).toBe(1)
        expect(component.pdPoolsGrouped[1][1][0].prefix).toBe('2001:db8:3:1::/64')
        // The groups with a single pool and overlapping prefixes are sorted by their delegated length.
        expect(component.pdPoolsGrouped[2][0]).toBe(5)
        expect(component.pdPoolsGrouped[2][1].length).toBe(1)
        expect(component.pdPoolsGrouped[2][1][0].prefix).toBe('2001:db8:4:0::/64')
        expect(component.pdPoolsGrouped[3][0]).toBe(4)
        expect(component.pdPoolsGrouped[3][1].length).toBe(1)
        expect(component.pdPoolsGrouped[3][1][0].prefix).toBe('2001:db8:4:0::/80')
        // The groups with multiple pools are sorted by their pool ID.
        // The prefixes in the group are sorted by their prefixes.
        expect(component.pdPoolsGrouped[4][0]).toBe(0)
        expect(component.pdPoolsGrouped[4][1].length).toBe(3)
        expect(component.pdPoolsGrouped[4][1][0].prefix).toBe('2001:db8:0:0::/64')
        expect(component.pdPoolsGrouped[4][1][1].prefix).toBe('2001:db8:0:1::/64')
        expect(component.pdPoolsGrouped[4][1][2].prefix).toBe('2001:db8:0:2::/64')
        expect(component.pdPoolsGrouped[5][0]).toBe(6)
        expect(component.pdPoolsGrouped[5][1].length).toBe(7)
        expect(component.pdPoolsGrouped[5][1][0].prefix).toBe('2001:db8:6:0::/64')
        expect(component.pdPoolsGrouped[5][1][1].prefix).toBe('2001:db8:6:1::/64')
        expect(component.pdPoolsGrouped[5][1][2].prefix).toBe('2001:db8:6:2::/64')
        expect(component.pdPoolsGrouped[5][1][3].prefix).toBe('2001:db8:6:3::/64')
        expect(component.pdPoolsGrouped[5][1][3].excludedPrefix).toBeUndefined()
        expect(component.pdPoolsGrouped[5][1][4].prefix).toBe('2001:db8:6:3::/64')
        expect(component.pdPoolsGrouped[5][1][4].excludedPrefix).toBeUndefined()
        expect(component.pdPoolsGrouped[5][1][5].prefix).toBe('2001:db8:6:3::/64')
        expect(component.pdPoolsGrouped[5][1][5].excludedPrefix).toBe('2001:db8:6:3::/80')
        expect(component.pdPoolsGrouped[5][1][6].prefix).toBe('2001:db8:6:3::/64')
        expect(component.pdPoolsGrouped[5][1][6].excludedPrefix).toBe('2001:db8:6:3::/96')
        expect(component.pdPoolsGrouped[6][0]).toBe(7)
        expect(component.pdPoolsGrouped[6][1].length).toBe(3)
        expect(component.pdPoolsGrouped[6][1][0].prefix).toBe('2001:db8:7:3::/64')
        expect(component.pdPoolsGrouped[6][1][1].prefix).toBe('2001:db8:7:1:1:1:1:1::/64')
        expect(component.pdPoolsGrouped[6][1][2].prefix).toBe('2001:db8:7:2:2:2:2:2::/64')
    })

    it('should not display the out of pool bar if there are no out of pool statistics', () => {
        component.source = {}
        fixture.detectChanges()
        expect((fixture.debugElement.nativeElement as HTMLElement).textContent).not.toContain('Out of pool')
    })

    it('should not display the out of pool bar if the out of pool statistics are zero', () => {
        const subnet: SubnetWithUniquePools = {
            pools: [{ pool: '10.0.0.1-10.0.10' }],
            outOfPoolAddrUtilization: 0,
            stats: { 'total-out-of-pool-addresses': 0, 'assigned-out-of-pool-addresses': 0 },
            statsCollectedAt: '2023-10-01T00:00:00Z',
        }
        component.source = subnet
        fixture.detectChanges()
        expect((fixture.debugElement.nativeElement as HTMLElement).textContent).not.toContain('Out of pool')
    })

    it('should display the out of pool bar if the out of pool statistics are not zero', () => {
        const subnet: SubnetWithUniquePools = {
            pools: [{ pool: '10.0.0.1-10.0.10' }],
            outOfPoolAddrUtilization: 0.1,
            stats: { 'total-out-of-pool-addresses': 10, 'assigned-out-of-pool-addresses': 1 },
            statsCollectedAt: '2023-10-01T00:00:00Z',
        }
        component.source = subnet
        fixture.detectChanges()
        expect((fixture.debugElement.nativeElement as HTMLElement).textContent).toContain('Out of pool')
    })
})
