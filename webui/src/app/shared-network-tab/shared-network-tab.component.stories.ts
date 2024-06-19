import { moduleMetadata, Meta, StoryObj, applicationConfig } from '@storybook/angular'
import { SharedNetworkTabComponent } from './shared-network-tab.component'
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
import { SubnetBarComponent } from '../subnet-bar/subnet-bar.component'
import { importProvidersFrom } from '@angular/core'
import { HttpClientModule } from '@angular/common/http'
import { ConfirmationService, MessageService } from 'primeng/api'
import { ConfirmDialogModule } from 'primeng/confirmdialog'
import { toastDecorator } from '../utils-stories'
import { MessageModule } from 'primeng/message'
import { ToastModule } from 'primeng/toast'
import { RouterModule, provideRouter } from '@angular/router'

export default {
    title: 'App/SharedNetworkTab',
    component: SharedNetworkTabComponent,
    decorators: [
        applicationConfig({
            providers: [
                ConfirmationService,
                importProvidersFrom(HttpClientModule),
                MessageService,
                provideNoopAnimations(),
                provideRouter([
                    { path: 'dhcp/shared-networks/:id', component: SharedNetworkTabComponent },
                    { path: 'iframe.html', component: SharedNetworkTabComponent },
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
                MessageModule,
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
                SubnetBarComponent,
                UtilizationStatsChartComponent,
                UtilizationStatsChartsComponent,
            ],
        }),
        toastDecorator,
    ],
} as Meta

type Story = StoryObj<SharedNetworkTabComponent>

export const SharedNetwork4: Story = {
    args: {
        sharedNetwork: {
            id: 1,
            name: 'foo',
            addrUtilization: 30,
            pools: [
                { pool: '192.0.2.1-192.0.2.10' },
                { pool: '192.0.2.100-192.0.2.110' },
                { pool: '192.0.2.150-192.0.2.160' },
                { pool: '192.0.3.1-192.0.3.10' },
                { pool: '192.0.3.100-192.0.3.110' },
                { pool: '192.0.3.150-192.0.3.160' },
            ],
            subnets: [
                {
                    id: 1,
                    subnet: '192.0.2.0/24',
                },
                {
                    id: 2,
                    subnet: '192.0.3.0/24',
                },
            ],
            stats: {
                'total-addresses': 240,
                'assigned-addresses': 70,
                'declined-addresses': 10,
            },
            statsCollectedAt: '2023-06-05',
            localSharedNetworks: [
                {
                    appId: 1,
                    appName: 'foo@192.0.2.1',
                    keaConfigSharedNetworkParameters: {
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
                    appId: 2,
                    appName: 'foo@192.0.2.2',
                    keaConfigSharedNetworkParameters: {
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

export const SharedNetwork6: Story = {
    args: {
        sharedNetwork: {
            id: 2,
            name: 'foo',
            universe: IPType.IPv6,
            addrUtilization: 30,
            pdUtilization: 60,
            pools: [{ pool: '2001:db8:1::2-2001:db8:1::786' }, { pool: '2001:db8:2::2-2001:db8:2::786' }],
            subnets: [
                {
                    id: 1,
                    subnet: '2001:db8:1::/64',
                },
                {
                    id: 2,
                    subnet: '2001:db8:2::/64',
                },
            ],
            stats: {
                'total-nas': 1000,
                'assigned-nas': 30,
                'declined-nas': 10,
                'total-pds': 500,
                'assigned-pds': 358,
            },
            statsCollectedAt: '2023-06-05',
            localSharedNetworks: [
                {
                    appId: 1,
                    appName: 'foo@192.0.2.1',
                    keaConfigSharedNetworkParameters: {
                        sharedNetworkLevelParameters: {
                            hostnameCharReplacement: 'X',
                            hostnameCharSet: '[^A-Za-z0-9.-]',
                        },
                        globalParameters: {
                            cacheThreshold: 0.29,
                            cacheMaxAge: 800,
                        },
                    },
                },
            ],
        },
    },
}
