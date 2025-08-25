import { moduleMetadata, Meta, StoryObj, applicationConfig } from '@storybook/angular'
import { provideNoopAnimations } from '@angular/platform-browser/animations'
import { toastDecorator } from '../utils-stories'
import { ZoneViewerComponent } from './zone-viewer.component'
import { ToastModule } from 'primeng/toast'
import { MessageService } from 'primeng/api'
import { TableModule } from 'primeng/table'
import { SidebarModule } from 'primeng/sidebar'
import { LocaltimePipe } from '../pipes/localtime.pipe'
import { ProgressSpinnerModule } from 'primeng/progressspinner'
import { PlaceholderPipe } from '../pipes/placeholder.pipe'
import { ButtonModule } from 'primeng/button'
import { DividerModule } from 'primeng/divider'
import { TooltipModule } from 'primeng/tooltip'
import { HelpTipComponent } from '../help-tip/help-tip.component'
import { PopoverModule } from 'primeng/popover'

export default {
    title: 'App/ZoneViewer',
    component: ZoneViewerComponent,
    decorators: [
        applicationConfig({
            providers: [provideNoopAnimations(), MessageService],
        }),
        moduleMetadata({
            imports: [
                ButtonModule,
                DividerModule,
                PopoverModule,
                ProgressSpinnerModule,
                SidebarModule,
                TableModule,
                ToastModule,
                TooltipModule,
            ],
            declarations: [HelpTipComponent, LocaltimePipe, PlaceholderPipe],
        }),
        toastDecorator,
    ],
} as Meta

type Story = StoryObj<ZoneViewerComponent>

export const Zone: Story = {
    args: {
        zoneTransferAt: '2024-03-15T01:00:00Z',
        data: {
            items: [
                {
                    rrClass: 'IN',
                    data: 'ns1.bind9.example.com. admin.bind9.example.com. 2024031501 43200 900 1814400 7200',
                    name: 'bind9.example.com.',
                    rrType: 'SOA',
                    ttl: 172800,
                },
                {
                    rrClass: 'IN',
                    data: 'ns1.bind9.example.com.',
                    name: 'bind9.example.com.',
                    rrType: 'NS',
                    ttl: 172800,
                },
                {
                    rrClass: 'IN',
                    data: 'ns2.bind9.example.com.',
                    name: 'bind9.example.com.',
                    rrType: 'NS',
                    ttl: 172800,
                },
                {
                    rrClass: 'IN',
                    data: '10 mail.bind9.example.com.',
                    name: 'bind9.example.com.',
                    rrType: 'MX',
                    ttl: 172800,
                },
                {
                    rrClass: 'IN',
                    data: '11.0.0.4',
                    name: 'mail.bind9.example.com.',
                    rrType: 'A',
                    ttl: 172800,
                },
                {
                    rrClass: 'IN',
                    data: '11.0.0.2',
                    name: 'ns1.bind9.example.com.',
                    rrType: 'A',
                    ttl: 172800,
                },
                {
                    rrClass: 'IN',
                    data: '11.0.0.3',
                    name: 'ns2.bind9.example.com.',
                    rrType: 'A',
                    ttl: 172800,
                },
                {
                    rrClass: 'IN',
                    data: '11.0.0.5',
                    name: 'www.bind9.example.com.',
                    rrType: 'A',
                    ttl: 172800,
                },
                {
                    rrClass: 'IN',
                    data: 'ns1.bind9.example.com. admin.bind9.example.com. 2024031501 43200 900 1814400 7200',
                    name: 'bind9.example.com.',
                    rrType: 'SOA',
                    ttl: 172800,
                },
            ],
        },
    },
}

export const NoZoneData: Story = {
    args: {
        zoneTransferAt: '2024-03-15T01:00:00Z',
        data: { items: [] },
    },
}
