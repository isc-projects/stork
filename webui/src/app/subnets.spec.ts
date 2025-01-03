import { TestBed } from '@angular/core/testing'
import { DHCPOption, SharedNetwork, Subnet } from './backend'

import {
    getTotalAddresses,
    getAssignedAddresses,
    parseSubnetStatisticValues,
    parseSubnetsStatisticValues,
    extractUniqueSubnetPools,
    hasDifferentLocalSubnetPools,
    hasAddressPools,
    hasPrefixPools,
    hasDifferentLocalSubnetOptions,
    extractUniqueSharedNetworkPools,
    hasDifferentLocalSharedNetworkOptions,
    hasDifferentSubnetLevelOptions,
    getStatisticValue,
    PoolWithLocalPools,
    hasDifferentLocalPoolOptions,
    hasDifferentSharedNetworkLevelOptions,
    hasDifferentGlobalLevelOptions,
    hasDifferentSubnetUserContexts,
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

    it('parse stats from string to big int for single subnet', () => {
        // Arrange
        const subnet6 = {
            subnet: '3000::0/24',
            stats: {
                'total-nas': '4',
                'assigned-nas': '18446744073709551615',
                'total-pds': '',
            },
        }

        // Act
        parseSubnetStatisticValues(subnet6)

        // Assert
        expect(subnet6.stats['total-nas']).toBe(BigInt('4') as any)
        expect(subnet6.stats['assigned-nas']).toBe(BigInt('18446744073709551615') as any)
        expect(subnet6.stats['total-pds']).toBe(BigInt(0) as any)
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
                        daemonId: 1,
                        appName: 'foo',
                        pools: [
                            {
                                pool: '3000::1-3000::5',
                                keaConfigPoolParameters: {
                                    clientClass: 'foo',
                                },
                            },
                            {
                                pool: '3000::40-3000::65',
                            },
                            {
                                pool: '3000::20-3000::35',
                            },
                            {
                                pool: '3000::10-3000::15',
                            },
                        ],
                        prefixDelegationPools: [
                            {
                                prefix: '3003::/64',
                                delegatedLength: 80,
                                keaConfigPoolParameters: {
                                    clientClass: 'bar',
                                },
                            },
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
                        ],
                    },
                    {
                        daemonId: 2,
                        appName: 'bar',
                        pools: [
                            {
                                pool: '3000::1-3000::5',
                                keaConfigPoolParameters: {
                                    clientClass: 'bar',
                                },
                            },
                            {
                                pool: '3000::20-3000::35',
                            },
                            {
                                pool: '3000::10-3000::15',
                            },
                            {
                                pool: '3000::70-3000::85',
                            },
                        ],
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
                                keaConfigPoolParameters: {
                                    clientClass: 'baz',
                                },
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

        expect(convertedSubnets[0].pools[0].pool).toBe('3000::1-3000::5')
        expect(convertedSubnets[0].pools[0].localPools?.length).toBe(2)
        expect(convertedSubnets[0].pools[0].localPools[0].daemonId).toBe(1)
        expect(convertedSubnets[0].pools[0].localPools[1].daemonId).toBe(2)
        expect(convertedSubnets[0].pools[0].localPools[0].appName).toBe('foo')
        expect(convertedSubnets[0].pools[0].localPools[1].appName).toBe('bar')
        expect(convertedSubnets[0].pools[0].localPools[0].keaConfigPoolParameters?.clientClass).toBe('foo')
        expect(convertedSubnets[0].pools[0].localPools[1].keaConfigPoolParameters?.clientClass).toBe('bar')
        expect(convertedSubnets[0].pools[1].pool).toBe('3000::10-3000::15')
        expect(convertedSubnets[0].pools[1].localPools?.length).toBe(2)
        expect(convertedSubnets[0].pools[1].localPools[0].daemonId).toBe(1)
        expect(convertedSubnets[0].pools[1].localPools[1].daemonId).toBe(2)
        expect(convertedSubnets[0].pools[1].localPools[0].appName).toBe('foo')
        expect(convertedSubnets[0].pools[1].localPools[1].appName).toBe('bar')
        expect(convertedSubnets[0].pools[2].pool).toBe('3000::20-3000::35')
        expect(convertedSubnets[0].pools[2].localPools?.length).toBe(2)
        expect(convertedSubnets[0].pools[2].localPools[0].daemonId).toBe(1)
        expect(convertedSubnets[0].pools[2].localPools[1].daemonId).toBe(2)
        expect(convertedSubnets[0].pools[2].localPools[0].appName).toBe('foo')
        expect(convertedSubnets[0].pools[2].localPools[1].appName).toBe('bar')
        expect(convertedSubnets[0].pools[3].pool).toBe('3000::40-3000::65')
        expect(convertedSubnets[0].pools[3].localPools?.length).toBe(1)
        expect(convertedSubnets[0].pools[3].localPools[0].appName).toBe('foo')
        expect(convertedSubnets[0].pools[3].localPools[0].daemonId).toBe(1)
        expect(convertedSubnets[0].pools[4].pool).toBe('3000::70-3000::85')
        expect(convertedSubnets[0].pools[4].localPools?.length).toBe(1)
        expect(convertedSubnets[0].pools[4].localPools[0].daemonId).toBe(2)
        expect(convertedSubnets[0].pools[4].localPools[0].appName).toBe('bar')

        expect(convertedSubnets[0].prefixDelegationPools[0].prefix).toBe('3001::/64')
        expect(convertedSubnets[0].prefixDelegationPools[0].localPools?.length).toBe(1)
        expect(convertedSubnets[0].prefixDelegationPools[0].localPools[0].daemonId).toBe(1)
        expect(convertedSubnets[0].prefixDelegationPools[1].prefix).toBe('3001::/64')
        expect(convertedSubnets[0].prefixDelegationPools[1].localPools?.length).toBe(1)
        expect(convertedSubnets[0].prefixDelegationPools[1].localPools[0].daemonId).toBe(2)
        expect(convertedSubnets[0].prefixDelegationPools[2].prefix).toBe('3002::/64')
        expect(convertedSubnets[0].prefixDelegationPools[2].localPools?.length).toBe(1)
        expect(convertedSubnets[0].prefixDelegationPools[2].localPools[0].daemonId).toBe(1)
        expect(convertedSubnets[0].prefixDelegationPools[3].prefix).toBe('3002::/64')
        expect(convertedSubnets[0].prefixDelegationPools[3].localPools?.length).toBe(1)
        expect(convertedSubnets[0].prefixDelegationPools[3].localPools[0].daemonId).toBe(2)
        expect(convertedSubnets[0].prefixDelegationPools[4].prefix).toBe('3003::/64')
        expect(convertedSubnets[0].prefixDelegationPools[4].localPools?.length).toBe(2)
        expect(convertedSubnets[0].prefixDelegationPools[4].localPools[0].daemonId).toBe(1)
        expect(convertedSubnets[0].prefixDelegationPools[4].localPools[1].daemonId).toBe(2)
        expect(convertedSubnets[0].prefixDelegationPools[4].localPools[0].keaConfigPoolParameters?.clientClass).toBe(
            'bar'
        )
        expect(convertedSubnets[0].prefixDelegationPools[4].localPools[1].keaConfigPoolParameters?.clientClass).toBe(
            'baz'
        )
    })

    it('extracts unique pools for several subnets', () => {
        const subnets4 = [
            {
                subnet: '192.0.2.0/24',
                localSubnets: [
                    {
                        pools: [
                            {
                                pool: '192.0.2.10-192.0.2.20',
                            },
                        ],
                    },
                ],
            },
            {
                subnet: '192.0.3.0/24',
                localSubnets: [
                    {
                        pools: [
                            {
                                pool: '192.0.3.10-192.0.3.20',
                            },
                        ],
                    },
                ],
            },
            {
                subnet: '192.0.4.0/24',
                localSubnets: [
                    {
                        pools: [
                            {
                                pool: '192.0.4.10-192.0.4.20',
                            },
                        ],
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
        expect(convertedSubnets[0].pools?.length).toBe(0)
        expect(convertedSubnets[0].prefixDelegationPools?.length).toBe(0)
        expect(convertedSubnets[1].pools?.length).toBe(0)
        expect(convertedSubnets[1].prefixDelegationPools?.length).toBe(0)
    })

    it('detects different address pools for servers', () => {
        const subnet = {
            subnet: '192.0.2.0/24',
            localSubnets: [
                {
                    pools: [
                        {
                            pool: '192.0.2.10-192.0.2.20',
                        },
                        {
                            pool: '192.0.2.30-192.0.2.40',
                        },
                        {
                            pool: '192.0.2.30-192.0.2.60',
                        },
                    ],
                },
                {
                    pools: [
                        {
                            pool: '192.0.2.10-192.0.2.40',
                        },
                        {
                            pool: '192.0.2.10-192.0.2.20',
                        },
                    ],
                },
            ],
        }
        expect(hasAddressPools(subnet)).toBeTrue()
        expect(hasPrefixPools(subnet)).toBeFalse()
        expect(hasDifferentLocalSubnetPools(subnet)).toBeTrue()
    })

    it('detects different address pools for the null case', () => {
        const subnet = {
            subnet: '192.0.2.0/24',
            localSubnets: [
                {
                    pools: [
                        {
                            pool: '192.0.2.10-192.0.2.20',
                        },
                        {
                            pool: '192.0.2.30-192.0.2.40',
                        },
                    ],
                },
                {
                    pools: null,
                },
            ],
        }
        expect(hasAddressPools(subnet)).toBeTrue()
        expect(hasPrefixPools(subnet)).toBeFalse()
        expect(hasDifferentLocalSubnetPools(subnet)).toBeTrue()
    })

    it('detects different prefix pool lengths for servers', () => {
        const subnet = {
            subnet: '2001:db8:1::/64',
            localSubnets: [
                {
                    prefixDelegationPools: [
                        {
                            prefix: '3000::',
                            delegatedLength: 64,
                        },
                    ],
                },
                {
                    prefixDelegationPools: [
                        {
                            prefix: '3000::',
                            delegatedLength: 80,
                        },
                    ],
                },
            ],
        }
        expect(hasAddressPools(subnet)).toBeFalse()
        expect(hasPrefixPools(subnet)).toBeTrue()
        expect(hasDifferentLocalSubnetPools(subnet)).toBeTrue()
    })

    it('detects different prefix pool prefixes for servers', () => {
        const subnet = {
            subnet: '2001:db8:1::/64',
            localSubnets: [
                {
                    prefixDelegationPools: [
                        {
                            prefix: '3000::',
                            delegatedLength: 64,
                        },
                        {
                            prefix: '3001::',
                            delegatedLength: 64,
                        },
                    ],
                },
                {
                    prefixDelegationPools: [
                        {
                            prefix: '3001::',
                            delegatedLength: 64,
                        },
                        {
                            prefix: '3002::',
                            delegatedLength: 64,
                        },
                    ],
                },
            ],
        }
        expect(hasAddressPools(subnet)).toBeFalse()
        expect(hasPrefixPools(subnet)).toBeTrue()
        expect(hasDifferentLocalSubnetPools(subnet)).toBeTrue()
    })

    it('it detects different prefix pools for the null case', () => {
        const subnet = {
            subnet: '2001:db8:1::/64',
            localSubnets: [
                {
                    prefixDelegationPools: null,
                },
                {
                    prefixDelegationPools: [
                        {
                            prefix: '3001::',
                            delegatedLength: 64,
                        },
                    ],
                },
            ],
        }
        expect(hasAddressPools(subnet)).toBeFalse()
        expect(hasPrefixPools(subnet)).toBeTrue()
        expect(hasDifferentLocalSubnetPools(subnet)).toBeTrue()
    })

    it('detects different prefix pool excluded prefixes for servers', () => {
        const subnet = {
            subnet: '2001:db8:1::/64',
            localSubnets: [
                {
                    prefixDelegationPools: [
                        {
                            prefix: '3000::',
                            delegatedLength: 64,
                            excludedPrefix: '3000::/120',
                        },
                    ],
                },
                {
                    prefixDelegationPools: [
                        {
                            prefix: '3000::',
                            delegatedLength: 64,
                            excludedPrefix: '3000::/116',
                        },
                    ],
                },
            ],
        }
        expect(hasAddressPools(subnet)).toBeFalse()
        expect(hasPrefixPools(subnet)).toBeTrue()
        expect(hasDifferentLocalSubnetPools(subnet)).toBeTrue()
    })

    it('detects the same address and prefix pools', () => {
        const subnet = {
            subnet: '2001:db8:1::/64',
            localSubnets: [
                {
                    pools: [
                        {
                            pool: '2001:db8:1::20-2001:db8:1::40',
                        },
                        {
                            pool: '2001:db8:1::60-2001:db8:1::80',
                        },
                    ],
                    prefixDelegationPools: [
                        {
                            prefix: '3000::',
                            delegatedLength: 64,
                            excludedPrefix: '3000::/120',
                        },
                        {
                            prefix: '3000:1::',
                            delegatedLength: 64,
                        },
                    ],
                },
                {
                    pools: [
                        {
                            pool: '2001:db8:1::60-2001:db8:1::80',
                        },
                        {
                            pool: '2001:db8:1::20-2001:db8:1::40',
                        },
                    ],
                    prefixDelegationPools: [
                        {
                            prefix: '3000:1::',
                            delegatedLength: 64,
                        },
                        {
                            prefix: '3000::',
                            delegatedLength: 64,
                            excludedPrefix: '3000::/120',
                        },
                    ],
                },
            ],
        }
        expect(hasAddressPools(subnet)).toBeTrue()
        expect(hasPrefixPools(subnet)).toBeTrue()
        expect(hasDifferentLocalSubnetPools(subnet)).toBeFalse()
    })

    it('detects different options for servers', () => {
        const subnet = {
            subnet: '192.0.2.0/24',
            localSubnets: [
                {
                    keaConfigSubnetParameters: {
                        subnetLevelParameters: {
                            optionsHash: '123',
                        },
                        sharedNetworkLevelParameters: {
                            optionsHash: '234',
                        },
                        globalParameters: {
                            optionsHash: '345',
                        },
                    },
                },
                {
                    keaConfigSubnetParameters: {
                        subnetLevelParameters: {
                            optionsHash: '123',
                        },
                        sharedNetworkLevelParameters: {
                            optionsHash: '345',
                        },
                        globalParameters: {
                            optionsHash: '234',
                        },
                    },
                },
            ],
        }
        expect(hasDifferentLocalSubnetOptions(subnet)).toBeTrue()
        expect(hasDifferentSubnetLevelOptions(subnet)).toBeFalse()
    })

    it('detects different options for servers for the null hash', () => {
        const subnet = {
            subnet: '192.0.2.0/24',
            localSubnets: [
                {
                    keaConfigSubnetParameters: {
                        subnetLevelParameters: {
                            optionsHash: '123',
                        },
                        sharedNetworkLevelParameters: {
                            optionsHash: null,
                        },
                        globalParameters: {
                            optionsHash: '345',
                        },
                    },
                },
                {
                    keaConfigSubnetParameters: {
                        subnetLevelParameters: {
                            optionsHash: '123',
                        },
                        sharedNetworkLevelParameters: {
                            optionsHash: '234',
                        },
                        globalParameters: {
                            optionsHash: '345',
                        },
                    },
                },
            ],
        }
        expect(hasDifferentLocalSubnetOptions(subnet)).toBeTrue()
        expect(hasDifferentSubnetLevelOptions(subnet)).toBeFalse()
    })

    it('detects different options for servers for the null parameters', () => {
        const subnet = {
            subnet: '192.0.2.0/24',
            localSubnets: [
                {
                    keaConfigSubnetParameters: {
                        subnetLevelParameters: {
                            optionsHash: '123',
                        },
                        sharedNetworkLevelParameters: null,
                        globalParameters: {
                            optionsHash: '345',
                        },
                    },
                },
                {
                    keaConfigSubnetParameters: {
                        subnetLevelParameters: {
                            optionsHash: '123',
                        },
                        sharedNetworkLevelParameters: {
                            optionsHash: '234',
                        },
                        globalParameters: {
                            optionsHash: '345',
                        },
                    },
                },
            ],
        }
        expect(hasDifferentLocalSubnetOptions(subnet)).toBeTrue()
        expect(hasDifferentSubnetLevelOptions(subnet)).toBeFalse()
    })

    it('detects different options for servers for non-existing parameters', () => {
        const subnet = {
            subnet: '192.0.2.0/24',
            localSubnets: [
                {
                    keaConfigSubnetParameters: {
                        subnetLevelParameters: {
                            optionsHash: '123',
                        },
                        globalParameters: {
                            optionsHash: '345',
                        },
                    },
                },
                {
                    keaConfigSubnetParameters: {
                        subnetLevelParameters: {
                            optionsHash: '123',
                        },
                        sharedNetworkLevelParameters: {
                            optionsHash: '234',
                        },
                        globalParameters: {
                            optionsHash: '345',
                        },
                    },
                },
            ],
        }
        expect(hasDifferentLocalSubnetOptions(subnet)).toBeTrue()
        expect(hasDifferentSubnetLevelOptions(subnet)).toBeFalse()
    })

    it('detects the same options for servers', () => {
        const subnet = {
            subnet: '192.0.2.0/24',
            localSubnets: [
                {
                    keaConfigSubnetParameters: {
                        subnetLevelParameters: {
                            optionsHash: '123',
                        },
                        sharedNetworkLevelParameters: {
                            optionsHash: '234',
                        },
                        globalParameters: {
                            optionsHash: '345',
                        },
                    },
                },
                {
                    keaConfigSubnetParameters: {
                        subnetLevelParameters: {
                            optionsHash: '123',
                        },
                        sharedNetworkLevelParameters: {
                            optionsHash: '234',
                        },
                        globalParameters: {
                            optionsHash: '345',
                        },
                    },
                },
            ],
        }
        expect(hasDifferentLocalSubnetOptions(subnet)).toBeFalse()
        expect(hasDifferentSubnetLevelOptions(subnet)).toBeFalse()
    })

    it('detects the same options for servers for null hashes', () => {
        const subnet = {
            subnet: '192.0.2.0/24',
            localSubnets: [
                {
                    keaConfigSubnetParameters: {
                        subnetLevelParameters: {
                            optionsHash: null,
                        },
                        sharedNetworkLevelParameters: {
                            optionsHash: '234',
                        },
                        globalParameters: {
                            optionsHash: '345',
                        },
                    },
                },
                {
                    keaConfigSubnetParameters: {
                        subnetLevelParameters: {
                            optionsHash: null,
                        },
                        sharedNetworkLevelParameters: {
                            optionsHash: '234',
                        },
                        globalParameters: {
                            optionsHash: '345',
                        },
                    },
                },
            ],
        }
        expect(hasDifferentLocalSubnetOptions(subnet)).toBeFalse()
        expect(hasDifferentSubnetLevelOptions(subnet)).toBeFalse()
    })

    it('detects different subnet level options', () => {
        const subnet = {
            subnet: '192.0.2.0/24',
            localSubnets: [
                {
                    keaConfigSubnetParameters: {
                        subnetLevelParameters: {
                            optionsHash: '123',
                        },
                        sharedNetworkLevelParameters: {
                            optionsHash: '234',
                        },
                        globalParameters: {
                            optionsHash: '345',
                        },
                    },
                },
                {
                    keaConfigSubnetParameters: {
                        subnetLevelParameters: {
                            optionsHash: '234',
                        },
                        sharedNetworkLevelParameters: {
                            optionsHash: '234',
                        },
                        globalParameters: {
                            optionsHash: '345',
                        },
                    },
                },
            ],
        }
        expect(hasDifferentLocalSubnetOptions(subnet)).toBeTrue()
        expect(hasDifferentSubnetLevelOptions(subnet)).toBeTrue()
    })

    it('detect different user-contexts for subnet', () => {
        const subnet: Subnet = {
            localSubnets: [
                {
                    userContext: {
                        foo: 1,
                        bar: 2,
                    },
                },
                {
                    userContext: {
                        foo: 1,
                        bar: 2,
                    },
                },
                {
                    userContext: {
                        foo: 1,
                        bar: 3,
                    },
                },
                {
                    userContext: {},
                },
                {},
            ],
        }

        expect(hasDifferentSubnetUserContexts(subnet)).toBeTrue()
    })

    it('should detect same user-contexts for subnet', () => {
        const subnet: Subnet = {
            localSubnets: [
                {
                    userContext: {
                        foo: 1,
                        bar: 2,
                    },
                },
                {
                    userContext: {
                        foo: 1,
                        bar: 2,
                    },
                },
                {
                    userContext: {
                        foo: 1,
                        bar: 2,
                    },
                },
            ],
        }

        expect(hasDifferentSubnetUserContexts(subnet)).toBeFalse()
        expect(hasDifferentSubnetUserContexts({})).toBeFalse()
        expect(hasDifferentSubnetUserContexts({ localSubnets: [] })).toBeFalse()
        expect(hasDifferentSubnetUserContexts({ localSubnets: [{}] })).toBeFalse()
    })

    it('extracts unique pools from a shared network', () => {
        const sharedNetworks6 = [
            {
                name: 'foo',
                subnets: [
                    {
                        subnet: '3000::/120',
                        localSubnets: [
                            {
                                pools: [
                                    {
                                        pool: '3000::1-3000::5',
                                    },
                                    {
                                        pool: '3000::10-3000::15',
                                    },
                                    {
                                        pool: '3000::20-3000::35',
                                    },
                                    {
                                        pool: '3000::40-3000::65',
                                    },
                                ],
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
                                pools: [
                                    {
                                        pool: '3000::1-3000::5',
                                    },
                                    {
                                        pool: '3000::10-3000::15',
                                    },
                                    {
                                        pool: '3000::20-3000::35',
                                    },
                                    {
                                        pool: '3000::70-3000::85',
                                    },
                                ],
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
                ],
            },
        ]

        const convertedSharedNetworks = extractUniqueSharedNetworkPools(sharedNetworks6)
        expect(convertedSharedNetworks.length).toBe(1)
        expect(convertedSharedNetworks[0].pools?.length).toBe(5)
        expect(convertedSharedNetworks[0].prefixDelegationPools?.length).toBe(5)

        expect(convertedSharedNetworks[0].pools[0].pool).toBe('3000::1-3000::5')
        expect(convertedSharedNetworks[0].pools[1].pool).toBe('3000::10-3000::15')
        expect(convertedSharedNetworks[0].pools[2].pool).toBe('3000::20-3000::35')
        expect(convertedSharedNetworks[0].pools[3].pool).toBe('3000::40-3000::65')
        expect(convertedSharedNetworks[0].pools[4].pool).toBe('3000::70-3000::85')

        expect(convertedSharedNetworks[0].prefixDelegationPools[0].prefix).toBe('3001::/64')
        expect(convertedSharedNetworks[0].prefixDelegationPools[1].prefix).toBe('3001::/64')
        expect(convertedSharedNetworks[0].prefixDelegationPools[2].prefix).toBe('3002::/64')
        expect(convertedSharedNetworks[0].prefixDelegationPools[3].prefix).toBe('3002::/64')
        expect(convertedSharedNetworks[0].prefixDelegationPools[4].prefix).toBe('3003::/64')
    })

    it('extracts unique pools for several shared networks', () => {
        const sharedNetworks6 = [
            {
                name: 'foo',
                subnets: [
                    {
                        subnet: '3000::/120',
                        localSubnets: [
                            {
                                pools: [
                                    {
                                        pool: '3000::1-3000::5',
                                    },
                                    {
                                        pool: '3000::10-3000::15',
                                    },
                                    {
                                        pool: '3000::20-3000::35',
                                    },
                                    {
                                        pool: '3000::40-3000::65',
                                    },
                                ],
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
                        ],
                    },
                ],
            },
            {
                name: 'bar',
                subnets: [
                    {
                        subnet: '3000::/120',
                        localSubnets: [
                            {
                                pools: [
                                    {
                                        pool: '3000::1-3000::5',
                                    },
                                    {
                                        pool: '3000::10-3000::15',
                                    },
                                    {
                                        pool: '3000::20-3000::35',
                                    },
                                    {
                                        pool: '3000::70-3000::85',
                                    },
                                ],
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
                ],
            },
        ]

        const convertedSharedNetworks = extractUniqueSharedNetworkPools(sharedNetworks6)
        expect(convertedSharedNetworks.length).toBe(2)
        expect(convertedSharedNetworks[0].pools?.length).toBe(4)
        expect(convertedSharedNetworks[0].prefixDelegationPools?.length).toBe(3)
        expect(convertedSharedNetworks[1].pools?.length).toBe(4)
        expect(convertedSharedNetworks[1].prefixDelegationPools?.length).toBe(3)
    })

    it('detects different shared network options for servers', () => {
        const sharedNetwork = {
            name: 'foo',
            localSharedNetworks: [
                {
                    keaConfigSharedNetworkParameters: {
                        sharedNetworkLevelParameters: {
                            optionsHash: '234',
                        },
                        globalParameters: {
                            optionsHash: '345',
                        },
                    },
                },
                {
                    keaConfigSharedNetworkParameters: {
                        sharedNetworkLevelParameters: {
                            optionsHash: '345',
                        },
                        globalParameters: {
                            optionsHash: '234',
                        },
                    },
                },
            ],
        }
        expect(hasDifferentSharedNetworkLevelOptions(sharedNetwork)).toBeTrue()
        expect(hasDifferentLocalSharedNetworkOptions(sharedNetwork)).toBeTrue()
    })

    it('detects different shared network options for servers for the null hash', () => {
        const sharedNetwork = {
            name: 'foo',
            localSharedNetworks: [
                {
                    keaConfigSharedNetworkParameters: {
                        sharedNetworkLevelParameters: {
                            optionsHash: null,
                        },
                        globalParameters: {
                            optionsHash: '345',
                        },
                    },
                },
                {
                    keaConfigSharedNetworkParameters: {
                        sharedNetworkLevelParameters: {
                            optionsHash: '234',
                        },
                        globalParameters: {
                            optionsHash: '345',
                        },
                    },
                },
            ],
        }
        expect(hasDifferentSharedNetworkLevelOptions(sharedNetwork)).toBeTrue()
        expect(hasDifferentLocalSharedNetworkOptions(sharedNetwork)).toBeTrue()
    })

    it('detects different options for servers for the null parameters', () => {
        const sharedNetwork = {
            name: 'foo',
            localSharedNetworks: [
                {
                    keaConfigSharedNetworkParameters: {
                        sharedNetworkLevelParameters: null,
                        globalParameters: {
                            optionsHash: '345',
                        },
                    },
                },
                {
                    keaConfigSharedNetworkParameters: {
                        sharedNetworkLevelParameters: {
                            optionsHash: '234',
                        },
                        globalParameters: {
                            optionsHash: '345',
                        },
                    },
                },
            ],
        }
        expect(hasDifferentSharedNetworkLevelOptions(sharedNetwork)).toBeTrue()
        expect(hasDifferentLocalSharedNetworkOptions(sharedNetwork)).toBeTrue()
    })

    it('detects different shared network options for servers for non-existing parameters', () => {
        const sharedNetwork = {
            name: 'foo',
            localSharedNetworks: [
                {
                    keaConfigSharedNetworkParameters: {
                        globalParameters: {
                            optionsHash: '345',
                        },
                    },
                },
                {
                    keaConfigSharedNetworkParameters: {
                        sharedNetworkLevelParameters: {
                            optionsHash: '234',
                        },
                        globalParameters: {
                            optionsHash: '345',
                        },
                    },
                },
            ],
        }
        expect(hasDifferentSharedNetworkLevelOptions(sharedNetwork)).toBeTrue()
        expect(hasDifferentLocalSharedNetworkOptions(sharedNetwork)).toBeTrue()
    })

    it('detects the same shared network options for servers', () => {
        const sharedNetwork = {
            name: 'foo',
            localSharedNetworks: [
                {
                    keaConfigSharedNetworkParameters: {
                        sharedNetworkLevelParameters: {
                            optionsHash: '234',
                        },
                        globalParameters: {
                            optionsHash: '345',
                        },
                    },
                },
                {
                    keaConfigSharedNetworkParameters: {
                        sharedNetworkLevelParameters: {
                            optionsHash: '234',
                        },
                        globalParameters: {
                            optionsHash: '345',
                        },
                    },
                },
            ],
        }
        expect(hasDifferentSharedNetworkLevelOptions(sharedNetwork)).toBeFalse()
        expect(hasDifferentLocalSharedNetworkOptions(sharedNetwork)).toBeFalse()
    })

    it('detects the same shared network options for servers for null hashes', () => {
        const sharedNetwork = {
            name: 'foo',
            localSharedNetworks: [
                {
                    keaConfigSharedNetworkParameters: {
                        sharedNetworkLevelParameters: {
                            optionsHash: null,
                        },
                        globalParameters: {
                            optionsHash: '345',
                        },
                    },
                },
                {
                    keaConfigSharedNetworkParameters: {
                        sharedNetworkLevelParameters: {
                            optionsHash: null,
                        },
                        globalParameters: {
                            optionsHash: '345',
                        },
                    },
                },
            ],
        }
        expect(hasDifferentSharedNetworkLevelOptions(sharedNetwork)).toBeFalse()
        expect(hasDifferentLocalSharedNetworkOptions(sharedNetwork)).toBeFalse()
    })

    it('detects the same shared network level options but different local options', () => {
        const sharedNetwork = {
            name: 'foo',
            localSharedNetworks: [
                {
                    keaConfigSharedNetworkParameters: {
                        sharedNetworkLevelParameters: {
                            optionsHash: '234',
                        },
                        globalParameters: {
                            optionsHash: '345',
                        },
                    },
                },
                {
                    keaConfigSharedNetworkParameters: {
                        sharedNetworkLevelParameters: {
                            optionsHash: '234',
                        },
                        globalParameters: {
                            optionsHash: '111',
                        },
                    },
                },
            ],
        }
        expect(hasDifferentSharedNetworkLevelOptions(sharedNetwork)).toBeFalse()
        expect(hasDifferentLocalSharedNetworkOptions(sharedNetwork)).toBeTrue()
    })

    it('detects different shared network options for servers', () => {
        const sharedNetwork = {
            name: 'foo',
            localSharedNetworks: [
                {
                    keaConfigSharedNetworkParameters: {
                        sharedNetworkLevelParameters: {
                            optionsHash: '234',
                        },
                        globalParameters: {
                            optionsHash: '345',
                        },
                    },
                },
                {
                    keaConfigSharedNetworkParameters: {
                        sharedNetworkLevelParameters: {
                            optionsHash: '345',
                        },
                        globalParameters: {
                            optionsHash: '234',
                        },
                    },
                },
            ],
        }
        expect(hasDifferentSharedNetworkLevelOptions(sharedNetwork)).toBeTrue()
        expect(hasDifferentLocalSharedNetworkOptions(sharedNetwork)).toBeTrue()
    })

    it('detects different pool options for servers for the null hash', () => {
        const pool: PoolWithLocalPools = {
            pool: '192.0.2.1-192.0.2.10',
            localPools: [
                {
                    keaConfigPoolParameters: {
                        optionsHash: null,
                    },
                },
                {
                    keaConfigPoolParameters: {
                        optionsHash: '234',
                    },
                },
            ],
        }
        expect(hasDifferentLocalPoolOptions(pool)).toBeTrue()
    })

    it('detects different pool options for servers for the null parameters', () => {
        const pool: PoolWithLocalPools = {
            pool: '192.0.2.1-192.0.2.10',
            localPools: [
                {
                    keaConfigPoolParameters: null,
                },
                {
                    keaConfigPoolParameters: {
                        optionsHash: '234',
                    },
                },
            ],
        }
        expect(hasDifferentLocalPoolOptions(pool)).toBeTrue()
    })

    it('detects the same pool options for servers', () => {
        const pool: PoolWithLocalPools = {
            pool: '192.0.2.1-192.0.2.10',
            localPools: [
                {
                    keaConfigPoolParameters: {
                        optionsHash: '234',
                    },
                },
                {
                    keaConfigPoolParameters: {
                        optionsHash: '234',
                    },
                },
            ],
        }
        expect(hasDifferentLocalPoolOptions(pool)).toBeFalse()
    })

    it('detects the same pool options for servers for empty parameters hashes', () => {
        const pool: PoolWithLocalPools = {
            pool: '192.0.2.1-192.0.2.10',
            localPools: [
                {
                    keaConfigPoolParameters: {},
                },
                {
                    keaConfigPoolParameters: {},
                },
            ],
        }
        expect(hasDifferentLocalPoolOptions(pool)).toBeFalse()
    })

    it('does not detect different global options for no configs', () => {
        expect(hasDifferentGlobalLevelOptions([])).toBeFalse()
    })

    it('does not detect different global options for a single config', () => {
        const config = {
            options: {
                options: [],
                optionsHash: 'foo',
            },
        }
        expect(hasDifferentGlobalLevelOptions([config])).toBeFalse()
    })

    it('does not detect different global options for the same configs', () => {
        const configReference = {
            options: {
                options: [
                    {
                        code: 1,
                    },
                ] as DHCPOption[],
                optionsHash: 'foo',
            },
        }

        const configSameHashSameOptions = {
            options: {
                options: [
                    {
                        code: 1,
                    },
                ] as DHCPOption[],
                optionsHash: 'foo',
            },
        }

        const configSameHashOnly = {
            options: {
                options: [
                    {
                        code: 42,
                    },
                ] as DHCPOption[],
                optionsHash: 'foo',
            },
        }

        expect(hasDifferentGlobalLevelOptions([configReference, configReference])).toBeFalse()
        expect(hasDifferentGlobalLevelOptions([configReference, configSameHashSameOptions])).toBeFalse()
        expect(hasDifferentGlobalLevelOptions([configReference, configSameHashOnly])).toBeFalse()
    })

    it('detects different global options for different configs', () => {
        const configs = [
            {
                options: {
                    options: [
                        {
                            code: 1,
                        },
                    ] as DHCPOption[],
                    optionsHash: 'foo',
                },
            },
            {
                options: {
                    options: [
                        {
                            code: 2,
                        },
                    ] as DHCPOption[],
                    optionsHash: 'bar',
                },
            },
        ]

        expect(hasDifferentGlobalLevelOptions(configs)).toBeTrue()
    })

    it('handles the case when the options are missing', () => {
        expect(hasDifferentGlobalLevelOptions([{}, {}])).toBeFalse()

        const config = {
            options: {
                options: [],
                optionsHash: 'foo',
            },
        }

        expect(hasDifferentGlobalLevelOptions([{}, config])).toBeTrue()
    })

    it('should extract the total addresses', () => {
        // Missing stats.
        expect(getTotalAddresses({})).toBeNull()
        // Empty stats.
        expect(getTotalAddresses({ stats: {} })).toBeUndefined()
        // IPv4 stats.
        expect(getTotalAddresses({ stats: { 'total-addresses': 42 } })).toBe(42)
        // IPv6 stats.
        expect(getTotalAddresses({ stats: { 'total-nas': 42 } })).toBe(42)
    })

    it('should extract the assigned addresses', () => {
        // Missing stats.
        expect(getAssignedAddresses({})).toBeNull()
        // Empty stats.
        expect(getAssignedAddresses({ stats: {} })).toBeUndefined()
        // IPv4 stats.
        expect(getAssignedAddresses({ stats: { 'assigned-addresses': 42 } })).toBe(42)
        // IPv6 stats.
        expect(getAssignedAddresses({ stats: { 'assigned-nas': 42 } })).toBe(42)
    })

    it('should extract statistic by name', () => {
        // Missing stats.
        expect(getStatisticValue({}, 'foo')).toBeNull()
        // Empty stats.
        expect(getStatisticValue({ stats: {} }, 'foo')).toBeNull()
        // Big int value.
        expect(getStatisticValue({ stats: { foo: BigInt(42) } }, 'foo')).toBe(42n)
        // Numeric value.
        expect(getStatisticValue({ stats: { foo: 42 } }, 'foo')).toBe(42n)
        // String value.
        expect(getStatisticValue({ stats: { foo: '42' } }, 'foo')).toBeNull()
    })
})
