import { TestBed } from '@angular/core/testing'
import { SharedNetwork, Subnet } from './backend'

import { getTotalAddresses, getAssignedAddresses, parseSubnetsStatisticValues } from './subnets'

describe('subnets', () => {
    beforeEach(() => TestBed.configureTestingModule({}))

    it('stats funcs should work for DHCPv4', () => {
        const subnet4 = {
            subnet: '192.168.0.0/24',
            stats: {
                'total-addresses': 4,
                'assigned-addresses': 2,
            },
        }
        const totalAddrs = getTotalAddresses(subnet4)
        expect(totalAddrs).toBe(4)

        const assignedAddrs = getAssignedAddresses(subnet4)
        expect(assignedAddrs).toBe(2)
    })

    it('stats funcs should work for DHCPv6', () => {
        const subnet6 = {
            subnet: '3000::0/24',
            stats: {
                'total-nas': 4,
                'assigned-nas': BigInt('18446744073709551615'),
            },
        }
        const totalAddrs = getTotalAddresses(subnet6)
        expect(totalAddrs).toBe(4)

        const assignedAddrs = getAssignedAddresses(subnet6)
        expect(assignedAddrs).toBe(BigInt('18446744073709551615'))
    })

    it('parse stats from string to big int', () => {
        // Arrange
        const subnets6 = [
            {
                subnet: '3000::0/24',
                stats: {
                    'total-nas': '4',
                    'assigned-nas': '18446744073709551615',
                    'total-pds': '',
                },
            },
        ]

        // Act
        parseSubnetsStatisticValues(subnets6)

        // Assert
        expect(subnets6[0].stats['total-nas']).toBe(BigInt('4') as any)
        expect(subnets6[0].stats['assigned-nas']).toBe(BigInt('18446744073709551615') as any)
        expect(subnets6[0].stats['total-pds']).toBe(BigInt(0) as any)
    })

    it('parse stats from non-string to big int', () => {
        // Arrange
        const obj = new Date()
        const subnets6 = [
            {
                subnet: '3000::0/24',
                stats: {
                    'total-nas': true,
                    'assigned-nas': 42,
                    'declined-nas': obj,
                    'assigned-pds': null,
                },
            },
        ]

        // Act
        parseSubnetsStatisticValues(subnets6)

        // Assert
        expect(subnets6[0].stats['total-nas']).toBe(true)
        expect(subnets6[0].stats['assigned-nas']).toBe(42)
        expect(subnets6[0].stats['declined-nas']).toBe(obj)
        expect(subnets6[0].stats['assigned-pds']).toBe(null)
    })

    it('parse stats from non-numeric string to big int', () => {
        // Arrange
        const subnets6 = [
            {
                subnet: '3000::0/24',
                stats: {
                    'total-nas': 'abc',
                    'assigned-nas': 'FF',
                },
            },
        ]

        // Act
        parseSubnetsStatisticValues(subnets6)

        // Assert
        expect(subnets6[0].stats['total-nas']).toBe('abc')
        expect(subnets6[0].stats['assigned-nas']).toBe('FF')
    })

    it('parse stats for missing local subnets', () => {
        // Arrange
        const subnets6 = [
            {
                subnet: '3000::0/24',
            },
        ]

        // Act
        parseSubnetsStatisticValues(subnets6)

        // Assert
        // No throw
        expect().nothing()
    })

    it('parse nested stats', () => {
        // Arrange
        const sharedNetworks: SharedNetwork[] = [{
            stats: { 'foo': '42' },
            subnets: [{
                stats: { 'bar': '24' },
                localSubnets: [{
                    stats: { 'baz': '4224' }
                }]
            }]
        }]

        // Act
        parseSubnetsStatisticValues(sharedNetworks)

        // Assert
        expect(sharedNetworks[0].stats['foo']).toBe(BigInt(42))
        expect(sharedNetworks[0].subnets[0].stats['bar']).toBe(BigInt(24))
        expect(sharedNetworks[0].subnets[0].localSubnets[0].stats['baz']).toBe(BigInt(4224))
    })
})
