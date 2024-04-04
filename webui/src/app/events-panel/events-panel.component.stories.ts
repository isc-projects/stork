import { HttpClientModule } from '@angular/common/http'
import { RouterModule } from '@angular/router'
import { applicationConfig, Meta, moduleMetadata, StoryObj } from '@storybook/angular'
import { MessageService } from 'primeng/api'
import { PaginatorModule } from 'primeng/paginator'
import { Events, EventsService, ServicesService, UsersService } from '../backend'
import { toastDecorator } from '../utils-stories'
import { EventsPanelComponent } from './events-panel.component'
import { TableModule } from 'primeng/table'
import { ToastModule } from 'primeng/toast'
import { action } from '@storybook/addon-actions'
import { LocaltimePipe } from '../pipes/localtime.pipe'
import { EventTextComponent } from '../event-text/event-text.component'
import { ButtonModule } from 'primeng/button'
import { importProvidersFrom } from '@angular/core'

export default {
    title: 'App/EventsPanel',
    component: EventsPanelComponent,
    decorators: [
        applicationConfig({
            providers: [
                EventsService,
                UsersService,
                ServicesService,
                MessageService,
                importProvidersFrom(HttpClientModule),
            ],
        }),
        moduleMetadata({
            imports: [HttpClientModule, PaginatorModule, RouterModule, TableModule, ToastModule, ButtonModule],
            declarations: [EventsPanelComponent, LocaltimePipe, EventTextComponent],
        }),
        toastDecorator,
    ],
    argTypes: {
        ui: {
            defaultValue: 'bare',
            control: 'radio',
            options: ['bare', 'table'],
        },
    },
} as Meta

type Story = StoryObj<EventsPanelComponent>

export const Primary: Story = {
    parameters: {
        mockData: [
            {
                url: 'http://localhost/api/events?start=0&limit=10&level=0',
                method: 'GET',
                status: 200,
                delay: 2000,
                response: (request) => {
                    const { searchParams } = request
                    const limit = parseInt(searchParams.limit, 10)
                    const start = parseInt(searchParams.start, 10)
                    action('onFetchEvents')()
                    return {
                        total: 100,
                        items: Array(limit)
                            .fill(null)
                            .map((_, idx) => ({
                                id: start + idx,
                                createdAt: new Date().toLocaleString(),
                                details:
                                    idx % 5 !== 1
                                        ? null
                                        : Array(start + idx)
                                              .fill('Lorem ipsum.')
                                              .join(' '),
                                level: idx % 4 == 3 ? undefined : idx % 4,
                                text: Array(10)
                                    .fill(0)
                                    .map(
                                        () =>
                                            ['Lorem', 'ipsum', 'dolor', 'sit', 'ament.'][Math.round(Math.random() * 4)]
                                    )
                                    .join(' '),
                            })),
                    } as Events
                },
            },
        ],
    },
}

export const Empty: Story = {
    parameters: {
        mockData: [
            {
                url: 'http://localhost/api/events?start=0&limit=10&level=0',
                method: 'GET',
                status: 200,
                delay: 2000,
                response: {
                    items: [],
                    total: 0,
                } as Events,
            },
        ],
    },
}
