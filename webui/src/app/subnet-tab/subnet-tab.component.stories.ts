import { moduleMetadata, Meta, StoryObj, applicationConfig } from '@storybook/angular'
import { SubnetTabComponent } from './subnet-tab.component'
import { ChartModule } from 'primeng/chart'
import { OverlayPanelModule } from 'primeng/overlaypanel'
import { HelpTipComponent } from '../help-tip/help-tip.component'
import { HumanCountComponent } from '../human-count/human-count.component'
import { HumanCountPipe } from '../pipes/human-count.pipe'
import { TooltipModule } from 'primeng/tooltip'
import { LocalNumberPipe } from '../pipes/local-number.pipe'
import { FieldsetModule } from 'primeng/fieldset'
import { DividerModule } from 'primeng/divider'
import { TableModule } from 'primeng/table'
import { provideNoopAnimations } from '@angular/platform-browser/animations'
import { UtilizationStatsChartComponent } from '../utilization-stats-chart/utilization-stats-chart.component'
import { EntityLinkComponent } from '../entity-link/entity-link.component'
import { AddressPoolBarComponent } from '../address-pool-bar/address-pool-bar.component'
import { DelegatedPrefixBarComponent } from '../delegated-prefix-bar/delegated-prefix-bar.component'
import { UtilizationStatsChartsComponent } from '../utilization-stats-charts/utilization-stats-charts.component'
import { CascadedParametersBoardComponent } from '../cascaded-parameters-board/cascaded-parameters-board.component'
import { ButtonModule } from 'primeng/button'
import { DhcpOptionSetViewComponent } from '../dhcp-option-set-view/dhcp-option-set-view.component'
import { TreeModule } from 'primeng/tree'
import { TagModule } from 'primeng/tag'
import { CheckboxModule } from 'primeng/checkbox'
import { FormsModule } from '@angular/forms'
import { IPType } from '../iptype'
import { PlaceholderPipe } from '../pipes/placeholder.pipe'
import { ConfirmationService, MessageService } from 'primeng/api'
import { toastDecorator } from '../utils-stories'
import { importProvidersFrom } from '@angular/core'
import { HttpClientModule } from '@angular/common/http'
import { ToastModule } from 'primeng/toast'
import { ConfirmDialogModule } from 'primeng/confirmdialog'
import { RouterModule, provideRouter } from '@angular/router'
import { PositivePipe } from '../pipes/positive.pipe'

export default {
    title: 'App/SubnetTab',
    component: SubnetTabComponent,
    decorators: [
        applicationConfig({
            providers: [
                ConfirmationService,
                importProvidersFrom(HttpClientModule),
                MessageService,
                provideNoopAnimations(),
                provideRouter([
                    { path: 'dhcp/subnets/:id', component: SubnetTabComponent },
                    { path: 'iframe.html', component: SubnetTabComponent },
                ]),
            ],
        }),
        moduleMetadata({
            imports: [
                ButtonModule,
                ChartModule,
                CheckboxModule,
                ConfirmDialogModule,
                DividerModule,
                FieldsetModule,
                FormsModule,
                OverlayPanelModule,
                RouterModule,
                TableModule,
                TagModule,
                ToastModule,
                TooltipModule,
                TreeModule,
            ],
            declarations: [
                AddressPoolBarComponent,
                CascadedParametersBoardComponent,
                DhcpOptionSetViewComponent,
                DelegatedPrefixBarComponent,
                EntityLinkComponent,
                HelpTipComponent,
                HumanCountComponent,
                HumanCountPipe,
                LocalNumberPipe,
                PlaceholderPipe,
                PositivePipe,
                UtilizationStatsChartComponent,
                UtilizationStatsChartsComponent,
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
                    appName: 'foo@192.0.2.1',
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
                    appName: 'foo@192.0.2.2',
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
                    appName: 'foo@192.0.2.1',
                    pools: null,
                },
            ],
        },
    },
}

export const Subnet4NoPoolsInOneServer: Story = {
    args: {
        subnet: {
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
                    appName: 'foo@192.0.2.1',
                    pools: null,
                },
                {
                    id: 2,
                    appName: 'bar@192.0.2.2',
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
                    appName: 'foo@2001:db8:1::1',
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
                    appName: 'foo@2001:db8:1::1',
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
                    appName: 'foo@2001:db8:1::1',
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
                    appName: 'foo@2001:db8:1::1',
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
                    appName: 'bar@2001:db8:2::5',
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
                    appName: 'foo@2001:db8:1::1',
                },
            ],
        },
    },
}
