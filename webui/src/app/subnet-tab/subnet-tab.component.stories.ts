import { Meta, StoryObj, applicationConfig } from '@storybook/angular'
import { SubnetTabComponent } from './subnet-tab.component'
import { IPType } from '../iptype'
import { ConfirmationService, MessageService } from 'primeng/api'
import { toastDecorator } from '../utils-stories'
import { provideHttpClient, withInterceptorsFromDi } from '@angular/common/http'
import { provideRouter, withHashLocation } from '@angular/router'
import { expect, within } from 'storybook/test'

export default {
    title: 'App/SubnetTab',
    component: SubnetTabComponent,
    decorators: [
        applicationConfig({
            providers: [
                ConfirmationService,
                MessageService,
                provideHttpClient(withInterceptorsFromDi()),
                provideRouter(
                    [
                        { path: 'dhcp/subnets/:id', component: SubnetTabComponent },
                        { path: '**', component: SubnetTabComponent },
                    ],
                    withHashLocation()
                ),
            ],
        }),
        toastDecorator,
    ],
} as Meta

type Story = StoryObj<SubnetTabComponent>

export const Subnet4: Story = {
    args: {
        subnet: {
            id: 123,
            subnet: '192.0.2.0/24',
            sharedNetwork: 'Fiber',
            addrUtilization: 30,
            stats: {
                'total-addresses': 240,
                'assigned-addresses': 70,
                'declined-addresses': 10,
            },
            statsCollectedAt: '2023-06-05',
            localSubnets: [
                {
                    id: 1,
                    daemonLabel: 'DHCPv4@localhost',
                    pools: [
                        {
                            pool: '192.0.2.1-192.0.2.100',
                        },
                    ],
                    keaConfigSubnetParameters: {
                        subnetLevelParameters: {
                            cacheThreshold: 0.25,
                            cacheMaxAge: 1000,
                            clientClass: 'baz',
                            requireClientClasses: ['foo', 'bar'],
                            ddnsGeneratedPrefix: 'myhost',
                            ddnsOverrideClientUpdate: true,
                            ddnsOverrideNoUpdate: true,
                            ddnsQualifyingSuffix: 'example.org',
                            ddnsReplaceClientName: 'never',
                            ddnsSendUpdates: true,
                            ddnsUpdateOnRenew: false,
                            ddnsUseConflictResolution: true,
                            fourOverSixInterface: 'bar',
                            fourOverSixInterfaceID: 'foo',
                            fourOverSixSubnet: '2001:db8:1::/64',
                            hostnameCharReplacement: 'Cde',
                            hostnameCharSet: '[^A-Za-z0-9.-]',
                            preferredLifetime: 1800,
                            minPreferredLifetime: 1600,
                            maxPreferredLifetime: 2000,
                            reservationMode: 'out-of-pool',
                            reservationsGlobal: false,
                            reservationsInSubnet: true,
                            reservationsOutOfPool: true,
                            renewTimer: 2000,
                            rebindTimer: 2600,
                            t1Percent: 0.25,
                            t2Percent: 0.75,
                            calculateTeeTimes: false,
                            validLifetime: 3600,
                            minValidLifetime: 3400,
                            maxValidLifetime: 3800,
                            allocator: 'random',
                            authoritative: false,
                            bootFileName: '/tmp/boot',
                            _interface: 'eth0',
                            interfaceID: 'foobar',
                            matchClientID: false,
                            nextServer: '192.1.2.3',
                            options: [
                                {
                                    code: 3,
                                    fields: [
                                        {
                                            fieldType: 'ipv4-address',
                                            values: ['192.0.2.1'],
                                        },
                                    ],
                                    universe: IPType.IPv4,
                                },
                            ],
                            optionsHash: '123',
                            pdAllocator: 'flq',
                            rapidCommit: true,
                            relay: {
                                ipAddresses: ['192.0.2.1', '192.0.2.2'],
                            },
                            serverHostname: 'foo.example.org',
                            storeExtendedInfo: true,
                        },
                        sharedNetworkLevelParameters: {
                            cacheThreshold: 0.3,
                            cacheMaxAge: 900,
                            clientClass: 'zab',
                            requireClientClasses: ['bar'],
                            ddnsGeneratedPrefix: 'herhost',
                            ddnsOverrideClientUpdate: false,
                            ddnsOverrideNoUpdate: true,
                            ddnsQualifyingSuffix: 'foo.example.org',
                            ddnsReplaceClientName: 'always',
                            ddnsSendUpdates: false,
                            ddnsUpdateOnRenew: true,
                            ddnsUseConflictResolution: false,
                            fourOverSixInterface: 'nn',
                            fourOverSixInterfaceID: 'ofo',
                            fourOverSixSubnet: '2001:db8:1::/64',
                            hostnameCharReplacement: 'X',
                            hostnameCharSet: '[^A-Za-z0-9.-]',
                            preferredLifetime: 1700,
                            minPreferredLifetime: 1500,
                            maxPreferredLifetime: 1900,
                            reservationMode: 'in-pool',
                            reservationsGlobal: false,
                            reservationsInSubnet: true,
                            reservationsOutOfPool: false,
                            renewTimer: 1900,
                            rebindTimer: 2500,
                            t1Percent: 0.26,
                            t2Percent: 0.74,
                            calculateTeeTimes: true,
                            validLifetime: 3700,
                            minValidLifetime: 3500,
                            maxValidLifetime: 4000,
                            allocator: 'flq',
                            authoritative: true,
                            bootFileName: '/tmp/boot.1',
                            _interface: 'eth1',
                            interfaceID: 'foo',
                            matchClientID: true,
                            nextServer: '192.1.2.4',
                            options: [
                                {
                                    code: 5,
                                    fields: [
                                        {
                                            fieldType: 'ipv4-address',
                                            values: ['8.8.8.8'],
                                        },
                                    ],
                                    universe: IPType.IPv4,
                                },
                            ],
                            optionsHash: '234',
                            pdAllocator: 'iterative',
                            rapidCommit: false,
                            relay: {
                                ipAddresses: ['192.0.2.2'],
                            },
                            serverHostname: 'off.example.org',
                            storeExtendedInfo: false,
                        },
                        globalParameters: {
                            cacheThreshold: 0.29,
                            cacheMaxAge: 800,
                            clientClass: 'abc',
                            requireClientClasses: [],
                            ddnsGeneratedPrefix: 'hishost',
                            ddnsOverrideClientUpdate: true,
                            ddnsOverrideNoUpdate: false,
                            ddnsQualifyingSuffix: 'uff.example.org',
                            ddnsReplaceClientName: 'never',
                            ddnsSendUpdates: true,
                            ddnsUpdateOnRenew: false,
                            ddnsUseConflictResolution: true,
                            fourOverSixInterface: 'enp0s8',
                            fourOverSixInterfaceID: 'idx',
                            fourOverSixSubnet: '2001:db8:1:1::/64',
                            hostnameCharReplacement: 'Y',
                            hostnameCharSet: '[^A-Za-z0-9.-]',
                            preferredLifetime: 1600,
                            minPreferredLifetime: 1400,
                            maxPreferredLifetime: 1800,
                            reservationMode: 'out-of-pool',
                            reservationsGlobal: true,
                            reservationsInSubnet: false,
                            reservationsOutOfPool: true,
                            renewTimer: 1800,
                            rebindTimer: 2400,
                            t1Percent: 0.24,
                            t2Percent: 0.7,
                            calculateTeeTimes: false,
                            validLifetime: 3600,
                            minValidLifetime: 3400,
                            maxValidLifetime: 3900,
                            allocator: 'iterative',
                            authoritative: false,
                            bootFileName: '/tmp/bootx',
                            _interface: 'eth0',
                            interfaceID: 'uffa',
                            matchClientID: false,
                            nextServer: '10.1.1.1',
                            options: [
                                {
                                    code: 23,
                                    fields: [
                                        {
                                            fieldType: 'uint8',
                                            values: ['10'],
                                        },
                                    ],
                                    universe: IPType.IPv4,
                                },
                            ],
                            optionsHash: '345',
                            pdAllocator: 'random',
                            rapidCommit: true,
                            serverHostname: 'abc.example.org',
                            storeExtendedInfo: false,
                        },
                    },
                },
                {
                    id: 1,
                    daemonLabel: 'DHCPv4@localhost',
                    pools: [
                        {
                            pool: '192.0.2.1-192.0.2.100',
                        },
                    ],
                    keaConfigSubnetParameters: {
                        subnetLevelParameters: {
                            cacheThreshold: 0.25,
                            cacheMaxAge: 1000,
                            clientClass: 'baz',
                            requireClientClasses: ['foo', 'bar'],
                            ddnsGeneratedPrefix: 'myhost',
                            ddnsOverrideClientUpdate: true,
                            ddnsOverrideNoUpdate: true,
                            ddnsQualifyingSuffix: 'example.org',
                            ddnsReplaceClientName: 'never',
                            ddnsSendUpdates: true,
                            ddnsUpdateOnRenew: false,
                            ddnsUseConflictResolution: true,
                            fourOverSixInterface: 'bar',
                            fourOverSixInterfaceID: 'foo',
                            fourOverSixSubnet: '2001:db8:1::/64',
                            hostnameCharReplacement: 'Cde',
                            hostnameCharSet: '[^A-Za-z0-9.-]',
                            preferredLifetime: 1800,
                            minPreferredLifetime: 1600,
                            maxPreferredLifetime: 2000,
                            reservationMode: 'out-of-pool',
                            reservationsGlobal: false,
                            reservationsInSubnet: true,
                            reservationsOutOfPool: true,
                            renewTimer: 2000,
                            rebindTimer: 2600,
                            t1Percent: 0.25,
                            t2Percent: 0.75,
                            calculateTeeTimes: false,
                            validLifetime: 3600,
                            minValidLifetime: 3400,
                            maxValidLifetime: 3800,
                            //                        allocator: 'random',
                            authoritative: false,
                            bootFileName: '/tmp/boot',
                            _interface: 'eth0',
                            interfaceID: 'foobar',
                            matchClientID: false,
                            nextServer: '192.1.2.3',
                            options: [
                                {
                                    code: 3,
                                    fields: [
                                        {
                                            fieldType: 'ipv4-address',
                                            values: ['192.0.2.1'],
                                        },
                                    ],
                                },
                            ],
                            optionsHash: '123',
                            pdAllocator: 'flq',
                            rapidCommit: true,
                            relay: {
                                ipAddresses: ['192.0.2.1', '192.0.2.2'],
                            },
                            serverHostname: 'foo.example.org',
                            storeExtendedInfo: true,
                        },
                        sharedNetworkLevelParameters: {
                            cacheThreshold: 0.3,
                            cacheMaxAge: 900,
                            clientClass: 'zab',
                            requireClientClasses: ['bar'],
                            ddnsGeneratedPrefix: 'herhost',
                            ddnsOverrideClientUpdate: false,
                            ddnsOverrideNoUpdate: true,
                            ddnsQualifyingSuffix: 'foo.example.org',
                            ddnsReplaceClientName: 'always',
                            ddnsSendUpdates: false,
                            ddnsUpdateOnRenew: true,
                            ddnsUseConflictResolution: false,
                            fourOverSixInterface: 'nn',
                            fourOverSixInterfaceID: 'ofo',
                            fourOverSixSubnet: '2001:db8:1::/64',
                            hostnameCharReplacement: 'X',
                            hostnameCharSet: '[^A-Za-z0-9.-]',
                            preferredLifetime: 1700,
                            minPreferredLifetime: 1500,
                            maxPreferredLifetime: 1900,
                            reservationMode: 'in-pool',
                            reservationsGlobal: false,
                            reservationsInSubnet: true,
                            reservationsOutOfPool: false,
                            renewTimer: 1900,
                            rebindTimer: 2500,
                            t1Percent: 0.26,
                            t2Percent: 0.74,
                            calculateTeeTimes: true,
                            validLifetime: 3700,
                            minValidLifetime: 3500,
                            maxValidLifetime: 4000,
                            allocator: 'flq',
                            authoritative: true,
                            bootFileName: '/tmp/boot.1',
                            _interface: 'eth1',
                            interfaceID: 'foo',
                            matchClientID: true,
                            nextServer: '192.1.2.4',
                            pdAllocator: 'iterative',
                            rapidCommit: false,
                            relay: {
                                ipAddresses: ['192.0.2.2'],
                            },
                            serverHostname: 'off.example.org',
                            storeExtendedInfo: false,
                        },
                        globalParameters: {
                            cacheThreshold: 0.29,
                            cacheMaxAge: 800,
                            clientClass: 'abc',
                            requireClientClasses: [],
                            ddnsGeneratedPrefix: 'hishost',
                            ddnsOverrideClientUpdate: true,
                            ddnsOverrideNoUpdate: false,
                            ddnsQualifyingSuffix: 'uff.example.org',
                            ddnsReplaceClientName: 'never',
                            ddnsSendUpdates: true,
                            ddnsUpdateOnRenew: false,
                            ddnsUseConflictResolution: true,
                            fourOverSixInterface: 'enp0s8',
                            fourOverSixInterfaceID: 'idx',
                            fourOverSixSubnet: '2001:db8:1:1::/64',
                            hostnameCharReplacement: 'Y',
                            hostnameCharSet: '[^A-Za-z0-9.-]',
                            preferredLifetime: 1600,
                            minPreferredLifetime: 1400,
                            maxPreferredLifetime: 1800,
                            reservationMode: 'out-of-pool',
                            reservationsGlobal: true,
                            reservationsInSubnet: false,
                            reservationsOutOfPool: true,
                            renewTimer: 1800,
                            rebindTimer: 2400,
                            t1Percent: 0.24,
                            t2Percent: 0.7,
                            calculateTeeTimes: false,
                            validLifetime: 3600,
                            minValidLifetime: 3400,
                            maxValidLifetime: 3900,
                            allocator: 'iterative',
                            authoritative: false,
                            bootFileName: '/tmp/bootx',
                            _interface: 'eth0',
                            interfaceID: 'uffa',
                            matchClientID: false,
                            nextServer: '10.1.1.1',
                            pdAllocator: 'random',
                            rapidCommit: true,
                            serverHostname: 'abc.example.org',
                            storeExtendedInfo: false,
                        },
                    },
                },
            ],
        },
    },
}

export const Subnet4NoPools: Story = {
    args: {
        subnet: {
            id: 123,
            subnet: '192.0.2.0/24',
            sharedNetwork: 'Fiber',
            addrUtilization: 50,
            stats: {
                'total-addresses': 30,
                'assigned-addresses': 15,
                'declined-addresses': 0,
            },
            statsCollectedAt: '2023-06-05',
            localSubnets: [
                {
                    id: 1,
                    daemonLabel: 'DHCPv4@localhost',
                    pools: null,
                },
            ],
        },
    },
}

export const Subnet4NoPoolsInOneServer: Story = {
    args: {
        subnet: {
            id: 123,
            subnet: '192.0.2.0/24',
            sharedNetwork: 'Fiber',
            addrUtilization: 30,
            stats: {
                'total-addresses': 240,
                'assigned-addresses': 70,
                'declined-addresses': 10,
            },
            statsCollectedAt: '2023-06-05',
            localSubnets: [
                {
                    id: 1,
                    daemonLabel: 'DHCPv4@localhost',
                    pools: null,
                },
                {
                    id: 2,
                    daemonLabel: 'DHCPv4@localhost',
                    pools: [
                        {
                            pool: '192.0.2.10-192.0.2.20',
                        },
                    ],
                },
            ],
        },
    },
}

export const Subnet6Address: Story = {
    args: {
        subnet: {
            id: 123,
            subnet: '2001:db8:1::/64',
            addrUtilization: 60,
            stats: {
                'total-nas': 1000,
                'assigned-nas': 30,
                'declined-nas': 10,
            },
            statsCollectedAt: '2023-06-05',
            localSubnets: [
                {
                    daemonLabel: 'DHCPv6@localhost',
                    pools: [
                        {
                            pool: '2001:db8:1::2-2001:db8:1::786',
                        },
                    ],
                },
            ],
        },
    },
}

export const Subnet6Prefix: Story = {
    args: {
        subnet: {
            id: 123,
            subnet: '2001:db8:1::/64',
            pdUtilization: 60,
            stats: {
                'total-pds': 500,
                'assigned-pds': 358,
            },
            statsCollectedAt: '2023-06-05',
            localSubnets: [
                {
                    id: 1,
                    daemonLabel: 'DHCPv6@localhost',
                    prefixDelegationPools: [
                        {
                            prefix: '3000::',
                            delegatedLength: 80,
                        },
                    ],
                },
            ],
        },
    },
}

export const Subnet6AddressPrefix: Story = {
    args: {
        subnet: {
            id: 123,
            subnet: '2001:db8:1::/64',
            addrUtilization: 88,
            pdUtilization: 60,
            stats: {
                'total-nas': 1024,
                'assigned-nas': 980,
                'declined-nas': 10,
                'total-pds': 500,
                'assigned-pds': 358,
            },
            statsCollectedAt: '2023-06-05',
            localSubnets: [
                {
                    id: 1,
                    daemonLabel: 'DHCPv6@localhost',
                    pools: [
                        {
                            pool: '2001:db8:1::2-2001:db8:1::768',
                        },
                    ],
                    prefixDelegationPools: [
                        {
                            prefix: '3000::',
                            delegatedLength: 80,
                        },
                    ],
                },
            ],
        },
    },
}

export const Subnet6DifferentPoolsOnDifferentServers: Story = {
    args: {
        subnet: {
            id: 123,
            subnet: '2001:db8:1::/64',
            addrUtilization: 88,
            pdUtilization: 60,
            stats: {
                'total-nas': 1024,
                'assigned-nas': 980,
                'declined-nas': 10,
                'total-pds': 500,
                'assigned-pds': 358,
            },
            statsCollectedAt: '2023-06-05',
            localSubnets: [
                {
                    id: 1,
                    daemonLabel: 'DHCPv6@localhost',
                    pools: [
                        {
                            pool: '2001:db8:1::2-2001:db8:1::768',
                        },
                    ],
                    prefixDelegationPools: [
                        {
                            prefix: '3000::',
                            delegatedLength: 80,
                        },
                    ],
                    stats: {
                        'total-nas': 1024,
                        'assigned-nas': 500,
                        'declined-nas': 4,
                        'total-pds': 300,
                        'assigned-pds': 200,
                    },
                },
                {
                    id: 2,
                    daemonLabel: 'DHCPv6@localhost',
                    pools: [
                        {
                            pool: '2001:db8:1::2-2001:db8:1::768',
                        },
                    ],
                    prefixDelegationPools: [
                        {
                            prefix: '3000::',
                            delegatedLength: 80,
                        },
                        {
                            prefix: '3000:1::',
                            delegatedLength: 96,
                        },
                    ],
                    stats: {
                        'total-nas': 1024,
                        'assigned-nas': 480,
                        'declined-nas': 6,
                        'total-pds': 200,
                        'assigned-pds': 158,
                    },
                },
            ],
        },
    },
}

export const Subnet6NoPools: Story = {
    args: {
        subnet: {
            id: 123,
            subnet: '2001:db8:1::/64',
            addrUtilization: 88,
            pdUtilization: 60,
            stats: {
                'total-nas': 1024,
                'assigned-nas': 980,
                'declined-nas': 10,
                'total-pds': 500,
                'assigned-pds': 358,
            },
            statsCollectedAt: '2023-06-05',
            localSubnets: [
                {
                    id: 1,
                    daemonLabel: 'DHCPv6@localhost',
                },
            ],
        },
    },
}

export const TestDisplaySubnet4NoPools: Story = {
    args: {
        subnet: {
            id: 1,
            subnet: '192.0.2.0/24',
            sharedNetwork: 'Fiber',
            addrUtilization: 30,
            stats: {
                'total-addresses': 240,
                'assigned-addresses': 70,
                'declined-addresses': 10,
            },
            statsCollectedAt: '2023-06-05',
            localSubnets: [
                {
                    id: 12223,
                    daemonId: 42,
                    daemonLabel: 'DHCPv4@localhost',
                    stats: {
                        'total-addresses': 240,
                        'assigned-addresses': 70,
                        'declined-addresses': 10,
                    },
                },
            ],
        },
    },
    play: async ({ canvasElement, userEvent }) => {
        // Test that it should display an IPv4 subnet without pools.
        const canvas = within(canvasElement)
        const title = canvasElement.querySelector('#tab-title-span')

        await expect(title).toHaveTextContent('Subnet 192.0.2.0/24 in shared network Fiber')

        await expect(canvas.getByRole('group', { name: 'DHCP Servers Using the Subnet' })).toBeVisible()

        await expect(canvas.getByText('12223')).toBeVisible()
        await expect(canvas.getByText(/\[42\] DHCPv4@localhost/)).toBeVisible()
        await expect(canvas.getByText('No pools configured.')).toBeVisible()

        await expect(canvas.getByRole('link', { name: 'Fiber' })).toBeVisible()
        await expect(canvas.getByText('No pools configured.')).toBeVisible()
        await expect(canvas.getByText('No user context configured.')).toBeVisible()

        await expect(canvas.getByRole('group', { name: /Pools\s\/\s+All Servers/ })).toBeVisible()

        await expect(canvas.getByRole('group', { name: 'Statistics' })).toBeVisible()
        // There should be 1 utilization pie chart. It appears as a <canvas> element with role "img".
        const statsFieldset = canvas.getByRole('group', { name: 'Statistics' })
        const charts = await within(statsFieldset).findAllByRole('img')
        await expect(charts).toHaveLength(1)

        const dhcpParamsBtn = await canvas.findByRole('button', { name: 'DHCP Parameters' })
        await userEvent.click(dhcpParamsBtn)

        await expect(canvas.getByText('No parameters configured.')).toBeVisible()

        const dhcpOptionsBtn = await canvas.findByRole('button', { name: /DHCP Options/ })
        await userEvent.click(dhcpOptionsBtn)
        await expect(canvas.getByText('No options configured.')).toBeVisible()
    },
}

export const TestDisplaySubnet6: Story = {
    args: {
        subnet: {
            id: 1,
            subnet: '2001:db8:1::/64',
            addrUtilization: 60,
            stats: {
                'total-nas': 1000,
                'assigned-nas': 30,
                'declined-nas': 10,
            },
            statsCollectedAt: '2023-06-05',
            localSubnets: [
                {
                    id: 12223,
                    daemonId: 42,
                    daemonLabel: 'DHCPv6@localhost',
                    pools: [{ pool: '2001:db8:1::2-2001:db8:1::786' }],
                    stats: {
                        'total-nas': 1000,
                        'assigned-nas': 30,
                        'declined-nas': 10,
                    },
                },
            ],
        },
    },
    play: async ({ canvasElement, userEvent }) => {
        // Test that it should display an IPv6 subnet.
        const canvas = within(canvasElement)

        await expect(canvas.getByText('Subnet 2001:db8:1::/64')).toBeVisible()

        await expect(canvas.getByRole('group', { name: 'DHCP Servers Using the Subnet' })).toBeVisible()
        await expect(canvas.getByText('12223')).toBeVisible()
        await expect(canvas.getByText(/\[42\] DHCPv6@localhost/)).toBeVisible()

        await expect(canvas.getByText('No user context configured.')).toBeVisible()

        await expect(canvas.getByRole('group', { name: /Pools\s\/\s+All Servers/ })).toBeVisible()

        await expect(canvas.getByText('2001:db8:1::2-2001:db8:1::786')).toBeVisible()

        await expect(canvas.getByRole('group', { name: 'Statistics' })).toBeVisible()
        // There should be 1 utilization pie chart. It appears as a <canvas> element with role "img".
        const statsFieldset = canvas.getByRole('group', { name: 'Statistics' })
        const charts = await within(statsFieldset).findAllByRole('img')
        await expect(charts).toHaveLength(1)

        await expect(canvas.getByText('No user context configured.')).toBeVisible()

        const dhcpParamsBtn = await canvas.findByRole('button', { name: 'DHCP Parameters' })
        await userEvent.click(dhcpParamsBtn)
        await expect(canvas.getByText('No parameters configured.')).toBeVisible()

        const dhcpOptionsBtn = await canvas.findByRole('button', { name: /DHCP Options/ })
        await userEvent.click(dhcpOptionsBtn)
        await expect(canvas.getByText('No options configured.')).toBeVisible()
    },
}

export const TestDisplaySubnet6AddressAndPrefix: Story = {
    args: {
        subnet: {
            id: 1,
            subnet: '2001:db8:1::/64',
            addrUtilization: 88,
            pdUtilization: 60,
            stats: {
                'total-nas': 1024,
                'assigned-nas': 980,
                'declined-nas': 10,
                'total-pds': 500,
                'assigned-pds': 358,
            },
            statsCollectedAt: '2023-06-05',
            localSubnets: [
                {
                    id: 12223,
                    daemonId: 42,
                    daemonLabel: 'DHCPv6@localhost',
                    pools: [{ pool: '2001:db8:1::2-2001:db8:1::768' }],
                    prefixDelegationPools: [{ prefix: '3000::', delegatedLength: 80 }],
                    stats: {
                        'total-nas': 1024,
                        'assigned-nas': 980,
                        'declined-nas': 10,
                        'total-pds': 500,
                        'assigned-pds': 358,
                    },
                },
            ],
        },
    },
    play: async ({ canvasElement, userEvent }) => {
        // Test that it should display an IPv6 subnet with address and prefix delegation pools.
        const canvas = within(canvasElement)

        await expect(canvas.getByText('Subnet 2001:db8:1::/64')).toBeVisible()
        await expect(canvas.getByText('2001:db8:1::2-2001:db8:1::768')).toBeVisible()
        await expect(canvas.getByText('3000::')).toBeVisible()
        await expect(canvas.getByText('No user context configured.')).toBeVisible()
        await expect(canvas.getByRole('group', { name: 'DHCP Servers Using the Subnet' })).toBeVisible()
        await expect(canvas.getByText('12223')).toBeVisible()
        await expect(canvas.getByText(/\[42\] DHCPv6@localhost/)).toBeVisible()
        await expect(canvas.getByRole('group', { name: /Pools\s\/\s+All Servers/ })).toBeVisible()

        // There should be 2 utilization pie charts. It appears as a <canvas> element with role "img".
        const statsFieldset = canvas.getByRole('group', { name: 'Statistics' })
        const charts = await within(statsFieldset).findAllByRole('img')
        await expect(charts).toHaveLength(2)

        const dhcpParamsBtn = await canvas.findByRole('button', { name: 'DHCP Parameters' })
        await userEvent.click(dhcpParamsBtn)
        await expect(canvas.getByText('No parameters configured.')).toBeVisible()

        const dhcpOptionsBtn = await canvas.findByRole('button', { name: /DHCP Options/ })
        await userEvent.click(dhcpOptionsBtn)
        await expect(canvas.getByText('No options configured.')).toBeVisible()
    },
}

export const TestDisplaySubnet6DifferentServers: Story = {
    args: {
        subnet: {
            id: 2,
            subnet: '2001:db8:1::/64',
            addrUtilization: 88,
            pdUtilization: 60,
            stats: {
                'total-nas': 1024,
                'assigned-nas': 980,
                'declined-nas': 10,
                'total-pds': 500,
                'assigned-pds': 358,
            },
            statsCollectedAt: '2023-06-05',
            localSubnets: [
                {
                    id: 123,
                    daemonId: 42,
                    daemonLabel: 'DHCPv6@host1',
                    pools: [{ pool: '2001:db8:1::2-2001:db8:1::768' }],
                    prefixDelegationPools: [{ prefix: '3000::', delegatedLength: 80 }],
                    stats: {
                        'total-nas': 1024,
                        'assigned-nas': 500,
                        'declined-nas': 5,
                        'total-pds': 500,
                        'assigned-pds': 200,
                    },
                    keaConfigSubnetParameters: {
                        subnetLevelParameters: {
                            cacheThreshold: 0.12, // value for daemon 1 (daemons 2 uses different value)
                            options: [
                                {
                                    code: 3,
                                    fields: [{ fieldType: 'ipv4-address', values: ['192.0.2.1'] }],
                                },
                            ],
                            optionsHash: '123',
                        },
                        sharedNetworkLevelParameters: { cacheThreshold: 0.3 },
                        globalParameters: { cacheThreshold: 0.111 },
                    },
                    userContext: { foo: 'user-context-is-here' },
                },
                {
                    id: 456,
                    daemonId: 43,
                    daemonLabel: 'DHCPv6@host2',
                    pools: [{ pool: '2001:db8:1::2-2001:db8:1::768' }],
                    prefixDelegationPools: [
                        { prefix: '3000::/64', delegatedLength: 80 },
                        { prefix: '3000:1::/64', delegatedLength: 96 },
                    ],
                    stats: {
                        'total-nas': 1024,
                        'assigned-nas': 480,
                        'declined-nas': 5,
                        'total-pds': 500,
                        'assigned-pds': 158,
                    },
                    keaConfigSubnetParameters: {
                        subnetLevelParameters: {
                            cacheThreshold: 0.34, // different value for daemon 2 (daemons 1 uses different value)
                            options: [
                                {
                                    code: 3,
                                    fields: [{ fieldType: 'ipv4-address', values: ['192.0.2.2'] }],
                                },
                            ],
                            optionsHash: '234',
                        },
                        sharedNetworkLevelParameters: { cacheThreshold: 0.3 },
                        globalParameters: { cacheThreshold: 0.222 },
                    },
                },
            ],
        },
    },
    play: async ({ canvasElement, userEvent }) => {
        // Test that it should display an IPv6 subnet with different
        // option values for different servers.
        const canvas = within(canvasElement)

        await expect(canvas.getByText('Subnet 2001:db8:1::/64')).toBeVisible()
        await expect(canvas.getByText('123')).toBeVisible()

        // This check is a bit more involved than in other test, because
        // it has two servers. Simple search for "[42] DHCPv6@localhost"
        // would return multiple matches.
        const dhcpServersUsingSubnet = canvas.getByRole('group', { name: 'DHCP Servers Using the Subnet' })
        await expect(dhcpServersUsingSubnet).toBeVisible()
        await expect(within(dhcpServersUsingSubnet).getByText('123')).toBeVisible()
        await expect(within(dhcpServersUsingSubnet).getByText(/\[42\] DHCPv6@host1/)).toBeVisible()
        await expect(within(dhcpServersUsingSubnet).getByText('456')).toBeVisible()
        await expect(within(dhcpServersUsingSubnet).getByText(/\[43\] DHCPv6@host2/)).toBeVisible()

        // Pools at server 1:
        const poolsAtServer1 = canvas.getByRole('group', { name: /Pools\s\/\s+\[42\]\s+DHCPv6@host1/ })
        await expect(poolsAtServer1).toBeVisible()
        await expect(within(poolsAtServer1).getByText('2001:db8:1::2-2001:db8:1::768')).toBeVisible()
        await expect(within(poolsAtServer1).getByText('3000::')).toBeVisible()

        // Pools at server 2:
        const poolsAtServer2 = canvas.getByRole('group', { name: /Pools\s\/\s+\[43\]\s+DHCPv6@host2/ })
        await expect(poolsAtServer2).toBeVisible()
        await expect(within(poolsAtServer2).getByText('2001:db8:1::2-2001:db8:1::768')).toBeVisible()
        await expect(within(poolsAtServer2).getByText('3000::/64')).toBeVisible()
        await expect(within(poolsAtServer2).getByText('3000:1::/64')).toBeVisible()

        // There should be 6 utilization pie charts. It appears as a <canvas> element with role "img".
        const statsFieldset = canvas.getByRole('group', { name: 'Statistics' })
        const charts = await within(statsFieldset).findAllByRole('img')
        await expect(charts).toHaveLength(6)

        await expect(canvas.getByText('3000::')).toBeVisible()
        await expect(canvas.getByText('3000:1::/64')).toBeVisible()
        const userCtx1 = await canvas.findByRole('group', { name: /User Context\s\/\s+\[42\]\s+DHCPv6@host1/ })
        const userCtx2 = await canvas.findByRole('group', { name: /User Context\s\/\s+\[43\]\s+DHCPv6@host2/ })
        await expect(within(userCtx1).getByText('foo')).toBeVisible()
        await expect(within(userCtx1).getByText('user-context-is-here')).toBeVisible()
        await expect(within(userCtx2).getByText('No user context configured.')).toBeVisible()

        const dhcpParamsBtn = await canvas.findByRole('button', { name: 'DHCP Parameters' })
        await userEvent.click(dhcpParamsBtn)
        await expect(canvas.getByText('Cache Threshold')).toBeVisible()
        await expect(canvas.getAllByText('0.12').length).toBeGreaterThan(0)
        await expect(canvas.getAllByText('0.34').length).toBeGreaterThan(0)

        const dhcpOptionsButtons = await canvas.findAllByRole('button', { name: /DHCP Options/ })
        await expect(dhcpOptionsButtons.length).toBe(2)
    },
}
