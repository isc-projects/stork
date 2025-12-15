import { moduleMetadata, Meta, StoryObj, applicationConfig } from '@storybook/angular'
import { provideNoopAnimations } from '@angular/platform-browser/animations'
import { toastDecorator } from '../utils-stories'
import { ToastModule } from 'primeng/toast'
import { MessageService } from 'primeng/api'
import { TableModule } from 'primeng/table'
import { provideHttpClient, withInterceptorsFromDi } from '@angular/common/http'
import { ZoneViewerComponent } from './zone-viewer.component'
import { ZoneRRs } from '../backend/model/zoneRRs'
import { LocaltimePipe } from '../pipes/localtime.pipe'
import { ProgressSpinnerModule } from 'primeng/progressspinner'
import { PlaceholderPipe } from '../pipes/placeholder.pipe'
import { TooltipModule } from 'primeng/tooltip'
import { DividerModule } from 'primeng/divider'
import { ButtonModule } from 'primeng/button'
import { PopoverModule } from 'primeng/popover'
import { HelpTipComponent } from '../help-tip/help-tip.component'

let mockGetZoneRRs: ZoneRRs = {
    zoneTransferAt: '2024-03-15T01:00:00Z',
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
    total: 10,
}

export default {
    title: 'App/ZoneViewer',
    component: ZoneViewerComponent,
    decorators: [
        applicationConfig({
            providers: [MessageService, provideHttpClient(withInterceptorsFromDi()), provideNoopAnimations()],
        }),
        moduleMetadata({
            imports: [
                ButtonModule,
                DividerModule,
                PopoverModule,
                ProgressSpinnerModule,
                TableModule,
                ToastModule,
                TooltipModule,
            ],
            declarations: [HelpTipComponent, LocaltimePipe, PlaceholderPipe, ZoneViewerComponent],
        }),
        toastDecorator,
    ],
    parameters: {
        mockData: [
            {
                url: 'http://localhost/api/daemons/:daemonId/:viewName/zones/:zoneId/rrs?start=0&limit=10',
                method: 'GET',
                status: 200,
                delay: 1000,
                response: mockGetZoneRRs,
            },
            {
                url: 'http://localhost/api/daemons/:daemonId/:viewName/zones/:zoneId/rrs/cache?start=0&limit=10',
                method: 'PUT',
                status: 200,
                delay: 1000,
                response: mockGetZoneRRs,
            },
        ],
    },
} as Meta

type Story = StoryObj<ZoneViewerComponent>

export const Zone: Story = {
    args: {
        daemonId: 1,
        viewName: 'default',
        zoneId: 1,
    },
}
