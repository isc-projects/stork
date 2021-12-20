import { TestBed } from '@angular/core/testing'

import { getTotalAddresses, getAssignedAddresses, parseSubnetsStatisticValues } from './subnets'

describe('subnets', () => {
    beforeEach(() => TestBed.configureTestingModule({}))

    it('stats funcs should work for DHCPv4', () => {
        const subnet4 = {
            subnet: '192.168.0.0/24',
            localSubnets: [
                {
                    stats: {
                        'total-addresses': 4,
                        'assigned-addresses': 2,
                    },
                },
            ],
        }
        const totalAddrs = getTotalAddresses(subnet4)
        expect(totalAddrs).toBe(4)

        const assignedAddrs = getAssignedAddresses(subnet4)
        expect(assignedAddrs).toBe(2)
    })

    it('stats funcs should work for DHCPv6', () => {
        const subnet6 = {
            subnet: '3000::0/24',
            localSubnets: [
                {
                    stats: {
                        'total-nas': 4,
                        'assigned-nas': BigInt('18446744073709551615'),
                    },
                },
            ],
        }
        const totalAddrs = getTotalAddresses(subnet6)
        expect(totalAddrs).toBe(4)

        const assignedAddrs = getAssignedAddresses(subnet6)
        expect(assignedAddrs).toBe(BigInt('18446744073709551615'))
    })

    it('parse stats from string to big int', () => {
        // Arrange
        const subnets6 = {
            items: [
                {
                    subnet: '3000::0/24',
                    localSubnets: [
                        {
                            stats: {
                                'total-nas': '4',
                                'assigned-nas': '18446744073709551615',
                                foo: 'bar',
                            },
                        },
                    ],
                },
            ],
        }

        // Act
        parseSubnetsStatisticValues(subnets6)

        // Assert
        expect(subnets6.items[0].localSubnets[0].stats['total-nas']).toBe(BigInt('4') as any)
        expect(subnets6.items[0].localSubnets[0].stats['assigned-nas']).toBe(BigInt('18446744073709551615') as any)
        expect(subnets6.items[0].localSubnets[0].stats['foo']).toBe('bar')
    })
})
