import { TestBed } from '@angular/core/testing'
import { SharedNetwork, Subnet } from './backend'

import {
    getTotalAddresses,
    getAssignedAddresses,
    parseSubnetsStatisticValues,
    extractUniqueSubnetPools,
} from './subnets'

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
        const sharedNetworks: SharedNetwork[] = [
            {
                stats: { foo: '42' },
                subnets: [
                    {
                        stats: { bar: '24' },
                        localSubnets: [
                            {
                                stats: { baz: '4224' },
                            },
                        ],
                    },
                ],
            },
        ]

        // Act
        parseSubnetsStatisticValues(sharedNetworks)

        // Assert
        expect(sharedNetworks[0].stats['foo']).toBe(BigInt(42))
        expect(sharedNetworks[0].subnets[0].stats['bar']).toBe(BigInt(24))
        expect(sharedNetworks[0].subnets[0].localSubnets[0].stats['baz']).toBe(BigInt(4224))
    })

    it('extracts unique pools', () => {
        const subnets6 = [
            {
                subnet: '3000::/120',
                localSubnets: [
                    {
                        pools: ['3000::1-3000::5', '3000::10-3000::15', '3000::20-3000::35', '3000::40-3000::65'],
                        prefixDelegationPools: [
                            {
                                prefix: '3001::/64',
                                delegatedLength: 80,
                                excludedPrefix: '3001::/96',
                            },
                            {
                                prefix: '3002::/64',
                                delegatedLength: 80,
                                excludedPrefix: '3002::/96',
                            },
                            {
                                prefix: '3003::/64',
                                delegatedLength: 80,
                            },
                        ],
                    },
                    {
                        pools: ['3000::1-3000::5', '3000::10-3000::15', '3000::20-3000::35', '3000::70-3000::85'],
                        prefixDelegationPools: [
                            {
                                prefix: '3001::/64',
                                delegatedLength: 88,
                                excludedPrefix: '3001::/96',
                            },
                            {
                                prefix: '3002::/64',
                                delegatedLength: 80,
                                excludedPrefix: '3002::/112',
                            },
                            {
                                prefix: '3003::/64',
                                delegatedLength: 80,
                            },
                        ],
                    },
                ],
            },
        ]

        const convertedSubnets = extractUniqueSubnetPools(subnets6)
        expect(convertedSubnets.length).toBe(1)
        expect(convertedSubnets[0].pools?.length).toBe(5)
        expect(convertedSubnets[0].prefixDelegationPools?.length).toBe(5)

        expect(convertedSubnets[0].pools[0]).toBe('3000::1-3000::5')
        expect(convertedSubnets[0].pools[1]).toBe('3000::10-3000::15')
        expect(convertedSubnets[0].pools[2]).toBe('3000::20-3000::35')
        expect(convertedSubnets[0].pools[3]).toBe('3000::40-3000::65')
        expect(convertedSubnets[0].pools[4]).toBe('3000::70-3000::85')

        expect(convertedSubnets[0].prefixDelegationPools[0].prefix).toBe('3001::/64')
        expect(convertedSubnets[0].prefixDelegationPools[1].prefix).toBe('3001::/64')
        expect(convertedSubnets[0].prefixDelegationPools[2].prefix).toBe('3002::/64')
        expect(convertedSubnets[0].prefixDelegationPools[3].prefix).toBe('3002::/64')
        expect(convertedSubnets[0].prefixDelegationPools[4].prefix).toBe('3003::/64')
    })

    it('extracts unique pools for several subnets', () => {
        const subnets4 = [
            {
                subnet: '192.0.2.0/24',
                localSubnets: [
                    {
                        pools: ['192.0.2.10-192.0.2.20'],
                    },
                ],
            },
            {
                subnet: '192.0.3.0/24',
                localSubnets: [
                    {
                        pools: ['192.0.3.10-192.0.3.20'],
                    },
                ],
            },
            {
                subnet: '192.0.4.0/24',
                localSubnets: [
                    {
                        pools: ['192.0.4.10-192.0.4.20'],
                    },
                ],
            },
        ]
        const convertedSubnets = extractUniqueSubnetPools(subnets4)
        expect(convertedSubnets.length).toBe(3)
        expect(convertedSubnets[0].pools?.length).toBe(1)
        expect(convertedSubnets[1].pools?.length).toBe(1)
        expect(convertedSubnets[2].pools?.length).toBe(1)
    })

    it('does not extract unique pools when they do not exist', () => {
        const subnets4 = [
            {
                subnet: '2001:db8:1::/64',
            },
            {
                subnet: '2001:db8:2::/64',
                localSubnets: [{}],
            },
        ]
        const convertedSubnets = extractUniqueSubnetPools(subnets4)
        expect(convertedSubnets.length).toBe(2)
        expect(convertedSubnets[0].pools).toBeFalsy()
        expect(convertedSubnets[0].prefixDelegationPools).toBeFalsy()
        expect(convertedSubnets[1].pools).toBeFalsy()
        expect(convertedSubnets[1].prefixDelegationPools).toBeFalsy()
    })
})
