import { applicationConfig, Meta, moduleMetadata, StoryObj } from '@storybook/angular'
import { TooltipModule } from 'primeng/tooltip'
import { UtilizationBarComponent } from '../utilization-bar/utilization-bar.component'
import { PoolBarsComponent } from './pool-bars.component'
import { AddressPoolBarComponent } from '../address-pool-bar/address-pool-bar.component'
import { DelegatedPrefixBarComponent } from '../delegated-prefix-bar/delegated-prefix-bar.component'

export default {
    title: 'App/PoolBars',
    component: PoolBarsComponent,
    decorators: [
        applicationConfig({
            providers: [],
        }),
        moduleMetadata({
            imports: [TooltipModule],
            declarations: [UtilizationBarComponent, AddressPoolBarComponent, DelegatedPrefixBarComponent],
        }),
    ],
} as Meta

type Story = StoryObj<PoolBarsComponent>

export const Primary: Story = {
    args: {
        addressPools: [
            // IPv4
            { pool: '10.0.10.1-10.0.10.42', utilization: 40 },
            { pool: '10.0.20.1-10.0.20.42', utilization: 40 },
            { pool: '10.1.10.1-10.1.10.42', utilization: 10, keaConfigPoolParameters: { poolID: 1 } },
            { pool: '10.1.20.1-10.1.20.42', utilization: 10, keaConfigPoolParameters: { poolID: 1 } },
            { pool: '10.1.30.1-10.1.30.42', utilization: 10, keaConfigPoolParameters: { poolID: 1 } },
            { pool: '10.1.40.1-10.1.40.42', utilization: 10, keaConfigPoolParameters: { poolID: 1 } },
            { pool: '10.1.50.1-10.1.50.42', utilization: 10, keaConfigPoolParameters: { poolID: 1 } },
            { pool: '10.1.60.1-10.1.60.42', utilization: 10, keaConfigPoolParameters: { poolID: 1 } },
            { pool: '10.1.70.1-10.1.70.42', utilization: 10, keaConfigPoolParameters: { poolID: 1 } },
            { pool: '10.1.80.1-10.1.80.42', utilization: 10, keaConfigPoolParameters: { poolID: 1 } },
            { pool: '10.1.90.1-10.1.90.42', utilization: 10, keaConfigPoolParameters: { poolID: 1 } },
            { pool: '10.1.100.1-10.1.100.42', utilization: 10, keaConfigPoolParameters: { poolID: 1 } },
            { pool: '10.1.110.1-10.1.110.42', utilization: 10, keaConfigPoolParameters: { poolID: 1 } },
            { pool: '10.1.120.1-10.1.120.42', utilization: 10, keaConfigPoolParameters: { poolID: 1 } },
            { pool: '10.1.130.1-10.1.130.42', utilization: 10, keaConfigPoolParameters: { poolID: 1 } },
            { pool: '10.1.140.1-10.1.140.42', utilization: 10, keaConfigPoolParameters: { poolID: 1 } },
            { pool: '10.1.150.1-10.1.150.42', utilization: 10, keaConfigPoolParameters: { poolID: 1 } },
            { pool: '10.1.160.1-10.1.160.42', utilization: 10, keaConfigPoolParameters: { poolID: 1 } },
            { pool: '10.2.10.1-10.2.10.42', utilization: 75, keaConfigPoolParameters: { poolID: 2 } },
            { pool: '10.2.20.1-10.2.20.42', utilization: 75, keaConfigPoolParameters: { poolID: 2 } },
            { pool: '10.0.30.1-10.0.30.42', utilization: 40 },
            { pool: '10.3.10.1-10.3.10.42', utilization: 80, keaConfigPoolParameters: { poolID: 3 } },
            { pool: '10.4.10.1-10.4.10.42', utilization: 85, keaConfigPoolParameters: { poolID: 4 } },
            { pool: '10.5.10.1-10.5.10.42', utilization: 95, keaConfigPoolParameters: { poolID: 5 } },
            // IPv6
            { pool: '2001:db8:0:0::1-2001:db8:0:0::42', utilization: 40 },
            { pool: '2001:db8:0:1::1-2001:db8:0:1::42', utilization: 40 },
            { pool: '2001:db8:1:2::1-2001:db8:1:2::42', utilization: 10, keaConfigPoolParameters: { poolID: 1 } },
            { pool: '2001:db8:1:3::1-2001:db8:1:3::42', utilization: 10, keaConfigPoolParameters: { poolID: 1 } },
            { pool: '2001:db8:1:4::1-2001:db8:1:4::42', utilization: 10, keaConfigPoolParameters: { poolID: 1 } },
            { pool: '2001:db8:1:5::1-2001:db8:1:5::42', utilization: 10, keaConfigPoolParameters: { poolID: 1 } },
            { pool: '2001:db8:1:6::1-2001:db8:1:6::42', utilization: 10, keaConfigPoolParameters: { poolID: 1 } },
            { pool: '2001:db8:1:7::1-2001:db8:1:7::42', utilization: 10, keaConfigPoolParameters: { poolID: 1 } },
            { pool: '2001:db8:1:8::1-2001:db8:1:8::42', utilization: 10, keaConfigPoolParameters: { poolID: 1 } },
            { pool: '2001:db8:1:9::1-2001:db8:1:9::42', utilization: 10, keaConfigPoolParameters: { poolID: 1 } },
            { pool: '2001:db8:1:a::1-2001:db8:1:a::42', utilization: 10, keaConfigPoolParameters: { poolID: 1 } },
            { pool: '2001:db8:1:b::1-2001:db8:1:b::42', utilization: 10, keaConfigPoolParameters: { poolID: 1 } },
            { pool: '2001:db8:1:c::1-2001:db8:1:c::42', utilization: 10, keaConfigPoolParameters: { poolID: 1 } },
            { pool: '2001:db8:1:d::1-2001:db8:1:d::42', utilization: 10, keaConfigPoolParameters: { poolID: 1 } },
            { pool: '2001:db8:2:e::1-2001:db8:2:e::42', utilization: 70, keaConfigPoolParameters: { poolID: 2 } },
            { pool: '2001:db8:3:f::1-2001:db8:3:f::42', utilization: 95, keaConfigPoolParameters: { poolID: 3 } },
        ],
        pdPools: [
            { prefix: '2001:db8:0:0::/64', delegatedLength: 64, utilization: 40 },
            { prefix: '2001:db8:0:1::/64', delegatedLength: 64, utilization: 40 },
            { prefix: '2001:db8:0:2::/64', delegatedLength: 64, utilization: 40, excludedPrefix: '2001:db8:1:2::/128' },
            { prefix: '2001:db8:0:3::/64', delegatedLength: 64, utilization: 40, excludedPrefix: '2001:db8:1:3::/128' },
            { prefix: '2001:db8:0:a::/64', delegatedLength: 64, utilization: 40, excludedPrefix: '2001:db8:1:a::/128' },
            { prefix: '2001:db8:0:e::/64', delegatedLength: 70, utilization: 40 },
            { prefix: '2001:db8:0::/32', delegatedLength: 32, utilization: 45 },
            {
                prefix: '2001:db8:1:0::/64',
                delegatedLength: 64,
                utilization: 40,
                keaConfigPoolParameters: { poolID: 1 },
            },
            {
                prefix: '2001:db8:1:1::/64',
                delegatedLength: 64,
                utilization: 40,
                keaConfigPoolParameters: { poolID: 1 },
            },
            {
                prefix: '2001:db8:2:0::/64',
                delegatedLength: 64,
                utilization: 70,
                keaConfigPoolParameters: { poolID: 2 },
            },
            {
                prefix: '2001:db8:3:0::/64',
                delegatedLength: 64,
                utilization: 40,
                keaConfigPoolParameters: { poolID: 3 },
            },
        ],
    },
}
