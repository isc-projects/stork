import { ZonesPageComponent } from './zones-page.component'
import { applicationConfig, Meta, moduleMetadata, StoryObj } from '@storybook/angular'
import { ConfirmationService } from 'primeng/api'
import { BreadcrumbsComponent } from '../breadcrumbs/breadcrumbs.component'
import { provideRouter, RouterModule, withHashLocation } from '@angular/router'
import { DialogModule } from 'primeng/dialog'
import { ButtonModule } from 'primeng/button'
import { TableModule } from 'primeng/table'
import { ConfirmDialogModule } from 'primeng/confirmdialog'
import { TabViewComponent } from '../tab-view/tab-view.component'
import { BreadcrumbModule } from 'primeng/breadcrumb'
import { HelpTipComponent } from '../help-tip/help-tip.component'
import { PluralizePipe } from '../pipes/pluralize.pipe'
import { MessageModule } from 'primeng/message'
import { PopoverModule } from 'primeng/popover'
import { PanelModule } from 'primeng/panel'
import { TooltipModule } from 'primeng/tooltip'
import { FloatLabelModule } from 'primeng/floatlabel'
import { MultiSelectModule } from 'primeng/multiselect'
import { InputNumberModule } from 'primeng/inputnumber'
import { SelectModule } from 'primeng/select'
import { InputIconModule } from 'primeng/inputicon'
import { IconFieldModule } from 'primeng/iconfield'
import { mockedFilterByText, toastDecorator } from '../utils-stories'
import { ToastModule } from 'primeng/toast'
import { FormsModule } from '@angular/forms'
import { InputTextModule } from 'primeng/inputtext'
import { TagModule } from 'primeng/tag'
import { LocaltimePipe } from '../pipes/localtime.pipe'
import { PlaceholderPipe } from '../pipes/placeholder.pipe'
import { UnrootPipe } from '../pipes/unroot.pipe'
import { LocalZone, Zone } from '../backend'
import { expect, userEvent, waitFor, within } from '@storybook/test'

const meta: Meta<ZonesPageComponent> = {
    title: 'App/ZonesPage',
    component: ZonesPageComponent,
    decorators: [
        applicationConfig({
            providers: [
                provideRouter(
                    [
                        {
                            path: 'dns/zones/:id',
                            component: ZonesPageComponent,
                        },
                    ],
                    withHashLocation()
                ),
            ],
        }),
        moduleMetadata({
            declarations: [
                BreadcrumbsComponent,
                HelpTipComponent,
                PluralizePipe,
                LocaltimePipe,
                PlaceholderPipe,
                UnrootPipe,
            ],
            imports: [
                RouterModule,
                DialogModule,
                ButtonModule,
                TableModule,
                ConfirmDialogModule,
                TabViewComponent,
                BreadcrumbModule,
                MessageModule,
                PopoverModule,
                PanelModule,
                TooltipModule,
                FloatLabelModule,
                MultiSelectModule,
                InputNumberModule,
                SelectModule,
                InputIconModule,
                IconFieldModule,
                ToastModule,
                FormsModule,
                InputTextModule,
                TagModule,
            ],
            providers: [ConfirmationService],
        }),
        toastDecorator,
    ],
}

export default meta
type Story = StoryObj<ZonesPageComponent>

export const EmptyList: Story = {
    parameters: {
        mockData: [
            {
                url: 'http://localhost/api/dns-management/zones-fetch',
                method: 'GET',
                status: 200,
                response: () => ({ total: 0, items: [] }),
            },
            {
                url: 'http://localhost/api/dns-management/zones-fetch',
                method: 'PUT',
                status: 202,
                response: () => {},
            },
            {
                url: 'http://localhost/api/zones?start=:start&limit=:limit&zoneType=consumer&zoneType=delegation-only&zoneType=forward&zoneType=hint&zoneType=mirror&zoneType=native&zoneType=primary&zoneType=producer&zoneType=redirect&zoneType=secondary&zoneType=static-stub&zoneType=stub',
                method: 'GET',
                status: 200,
                response: () => ({ total: 0, items: [] }),
            },
            {
                url: 'http://localhost/api/zones?start=:start&limit=:limit&zoneType=builtin&zoneType=consumer&zoneType=delegation-only&zoneType=forward&zoneType=hint&zoneType=mirror&zoneType=native&zoneType=primary&zoneType=producer&zoneType=redirect&zoneType=secondary&zoneType=static-stub&zoneType=stub',
                method: 'GET',
                status: 200,
                response: () => ({ total: 0, items: [] }),
            },
        ],
    },
}

const builtinZones = [
    {
        id: 20,
        localZones: [
            {
                appId: 2,
                appName: 'bind9@agent-bind9-2',
                class: 'CH',
                daemonId: 2,
                loadedAt: '2025-12-22T18:31:41.000Z',
                serial: 0,
                view: '_bind',
                zoneType: 'builtin',
            },
            {
                appId: 4,
                appName: 'bind9@agent-bind9',
                class: 'CH',
                daemonId: 4,
                loadedAt: '2025-12-22T18:31:41.000Z',
                serial: 0,
                view: 'guest',
                zoneType: 'builtin',
            },
            {
                appId: 4,
                appName: 'bind9@agent-bind9',
                class: 'CH',
                daemonId: 4,
                loadedAt: '2025-12-22T18:31:41.000Z',
                serial: 0,
                view: 'trusted',
                zoneType: 'builtin',
            },
        ],
        name: 'hostname.bind',
        rname: 'bind.hostname',
    },
    {
        id: 21,
        localZones: [
            {
                appId: 2,
                appName: 'bind9@agent-bind9-2',
                class: 'IN',
                daemonId: 2,
                loadedAt: '2025-12-22T18:31:41.000Z',
                serial: 0,
                view: '_default',
                zoneType: 'builtin',
            },
            {
                appId: 4,
                appName: 'bind9@agent-bind9',
                class: 'IN',
                daemonId: 4,
                loadedAt: '2025-12-22T18:31:41.000Z',
                serial: 0,
                view: 'guest',
                zoneType: 'builtin',
            },
            {
                appId: 4,
                appName: 'bind9@agent-bind9',
                class: 'IN',
                daemonId: 4,
                loadedAt: '2025-12-22T18:31:41.000Z',
                serial: 0,
                view: 'trusted',
                zoneType: 'builtin',
            },
        ],
        name: 'HOME.ARPA',
        rname: 'ARPA.HOME',
    },
    {
        id: 22,
        localZones: [
            {
                appId: 2,
                appName: 'bind9@agent-bind9-2',
                class: 'IN',
                daemonId: 2,
                loadedAt: '2025-12-22T18:31:41.000Z',
                serial: 0,
                view: '_default',
                zoneType: 'builtin',
            },
            {
                appId: 4,
                appName: 'bind9@agent-bind9',
                class: 'IN',
                daemonId: 4,
                loadedAt: '2025-12-22T18:31:41.000Z',
                serial: 0,
                view: 'guest',
                zoneType: 'builtin',
            },
            {
                appId: 4,
                appName: 'bind9@agent-bind9',
                class: 'IN',
                daemonId: 4,
                loadedAt: '2025-12-22T18:31:41.000Z',
                serial: 0,
                view: 'trusted',
                zoneType: 'builtin',
            },
        ],
        name: '0.IN-ADDR.ARPA',
        rname: 'ARPA.IN-ADDR.0',
    },
]

const rootZone = {
    id: 15,
    localZones: [
        {
            appId: 2,
            appName: 'bind9@agent-bind9-2',
            class: 'IN',
            daemonId: 2,
            loadedAt: '1970-01-01T00:00:00.000Z',
            serial: 0,
            view: '_default',
            zoneType: 'mirror',
        },
    ],
    name: '.',
    rname: '.',
}

const primaryZones = [
    {
        id: 1,
        localZones: [
            {
                appId: 1,
                appName: 'pdns@agent-pdns',
                class: 'IN',
                daemonId: 1,
                loadedAt: '2025-12-22T17:59:38.000Z',
                serial: 2024031501,
                view: 'localhost',
                zoneType: 'master',
            },
        ],
        name: '0.0.10.in-addr.arpa',
        rname: 'arpa.in-addr.10.0.0',
    },
    {
        id: 2,
        localZones: [
            {
                appId: 1,
                appName: 'pdns@agent-pdns',
                class: 'IN',
                daemonId: 1,
                loadedAt: '2025-12-22T17:59:38.000Z',
                serial: 2024031501,
                view: 'localhost',
                zoneType: 'master',
            },
        ],
        name: '1.0.10.in-addr.arpa',
        rname: 'arpa.in-addr.10.0.1',
    },
    {
        id: 3,
        localZones: [
            {
                appId: 1,
                appName: 'pdns@agent-pdns',
                class: 'IN',
                daemonId: 1,
                loadedAt: '2025-12-22T17:59:38.000Z',
                serial: 2024031501,
                view: 'localhost',
                zoneType: 'master',
            },
        ],
        name: '2.0.10.in-addr.arpa',
        rname: 'arpa.in-addr.10.0.2',
    },
    {
        id: 4,
        localZones: [
            {
                appId: 1,
                appName: 'pdns@agent-pdns',
                class: 'IN',
                daemonId: 1,
                loadedAt: '2025-12-22T17:59:38.000Z',
                serial: 2024031501,
                view: 'localhost',
                zoneType: 'master',
            },
        ],
        name: '3.0.10.in-addr.arpa',
        rname: 'arpa.in-addr.10.0.3',
    },
    {
        id: 5,
        localZones: [
            {
                appId: 2,
                appName: 'bind9@agent-bind9-2',
                class: 'IN',
                daemonId: 2,
                loadedAt: '2025-12-23T08:59:19.000Z',
                rpz: true,
                serial: 201702121,
                view: '_default',
                zoneType: 'secondary',
            },
            {
                appId: 4,
                appName: 'bind9@agent-bind9',
                class: 'IN',
                daemonId: 4,
                loadedAt: '2025-08-14T06:05:51.000Z',
                rpz: true,
                serial: 201702122,
                view: 'trusted',
                zoneType: 'primary',
            },
        ],
        name: 'drop.rpz.example.com',
        rname: 'com.example.rpz.drop',
    },
    {
        id: 6,
        localZones: [
            {
                appId: 2,
                appName: 'bind9@agent-bind9-2',
                class: 'IN',
                daemonId: 2,
                loadedAt: '2025-08-14T06:05:51.000Z',
                rpz: true,
                serial: 201702121,
                view: '_default',
                zoneType: 'primary',
            },
        ],
        name: 'rpz.local',
        rname: 'local.rpz',
    },
]

type TestLocalZone = LocalZone & { class: string }

const allZones: Zone[] = [rootZone, ...builtinZones, ...primaryZones]

const filterByZoneType = (url: string) => {
    const search = new URL(url, 'http://localhost').search
    const searchParams = new URLSearchParams(search)
    const zoneTypes = searchParams.getAll('zoneType')
    if (zoneTypes.includes('primary')) {
        zoneTypes.push('master')
    }

    return allZones.filter((z) => z.localZones.some((lz) => zoneTypes.includes(lz.zoneType)))
}

export const ListZones: Story = {
    parameters: {
        mockData: [
            {
                url: 'api/dns-management/zones-fetch',
                method: 'GET',
                status: 200,
                response: () => ({
                    total: 3,
                    items: [
                        {
                            appId: 1,
                            appName: 'pdns@agent-pdns',
                            builtinZonesCount: 0,
                            createdAt: '2025-12-22T17:22:46.009Z',
                            daemonId: 1,
                            distinctZonesCount: 10,
                            status: 'ok',
                            zoneConfigsCount: 10,
                        },
                        {
                            appId: 2,
                            appName: 'bind9@agent-bind9-2',
                            builtinZonesCount: 104,
                            createdAt: '2025-12-22T17:22:46.022Z',
                            daemonId: 2,
                            distinctZonesCount: 109,
                            status: 'ok',
                            zoneConfigsCount: 109,
                        },
                        {
                            appId: 4,
                            appName: 'bind9@agent-bind9',
                            builtinZonesCount: 104,
                            createdAt: '2025-12-22T17:22:46.040Z',
                            daemonId: 5,
                            distinctZonesCount: 107,
                            status: 'ok',
                            zoneConfigsCount: 207,
                        },
                    ],
                }),
            },
            {
                url: 'api/dns-management/zones-fetch',
                method: 'PUT',
                status: 202,
                response: () => {},
            },
            {
                url: 'api/zones?start=s&limit=l&zoneType=1',
                method: 'GET',
                status: 200,
                response: ({ url }) => {
                    const filteredZones = filterByZoneType(url)
                    return {
                        items: filteredZones,
                        total: filteredZones.length,
                    }
                },
            },
            {
                url: 'api/zones?start=s&limit=l&zoneType=1&zoneType=2',
                method: 'GET',
                status: 200,
                response: ({ url }) => {
                    const filteredZones = filterByZoneType(url)
                    return {
                        items: filteredZones,
                        total: filteredZones.length,
                    }
                },
            },
            {
                url: 'api/zones?start=s&limit=l&zoneType=1&zoneType=2&zoneType=3',
                method: 'GET',
                status: 200,
                response: ({ url }) => {
                    const filteredZones = filterByZoneType(url)
                    return {
                        items: filteredZones,
                        total: filteredZones.length,
                    }
                },
            },
            {
                url: 'api/zones?start=s&limit=l&zoneType=1&zoneType=2&zoneType=3&zoneType=4',
                method: 'GET',
                status: 200,
                response: ({ url }) => {
                    const filteredZones = filterByZoneType(url)
                    return {
                        items: filteredZones,
                        total: filteredZones.length,
                    }
                },
            },
            {
                url: 'api/zones?start=s&limit=l&zoneType=1&zoneType=2&zoneType=3&zoneType=4&zoneType=5',
                method: 'GET',
                status: 200,
                response: ({ url }) => {
                    const filteredZones = filterByZoneType(url)
                    return {
                        items: filteredZones,
                        total: filteredZones.length,
                    }
                },
            },
            {
                url: 'api/zones?start=s&limit=l&zoneType=1&zoneType=2&zoneType=3&zoneType=4&zoneType=5&zoneType=6',
                method: 'GET',
                status: 200,
                response: ({ url }) => {
                    const filteredZones = filterByZoneType(url)
                    return {
                        items: filteredZones,
                        total: filteredZones.length,
                    }
                },
            },
            {
                url: 'api/zones?start=s&limit=l&zoneType=1&zoneType=2&zoneType=3&zoneType=4&zoneType=5&zoneType=6&zoneType=7',
                method: 'GET',
                status: 200,
                response: ({ url }) => {
                    const filteredZones = filterByZoneType(url)
                    return {
                        items: filteredZones,
                        total: filteredZones.length,
                    }
                },
            },
            {
                url: 'api/zones?start=s&limit=l&zoneType=1&zoneType=2&zoneType=3&zoneType=4&zoneType=5&zoneType=6&zoneType=7&zoneType=8',
                method: 'GET',
                status: 200,
                response: ({ url }) => {
                    const filteredZones = filterByZoneType(url)
                    return {
                        items: filteredZones,
                        total: filteredZones.length,
                    }
                },
            },
            {
                url: 'api/zones?start=s&limit=l&zoneType=1&zoneType=2&zoneType=3&zoneType=4&zoneType=5&zoneType=6&zoneType=7&zoneType=8&zoneType=9',
                method: 'GET',
                status: 200,
                response: ({ url }) => {
                    const filteredZones = filterByZoneType(url)
                    return {
                        items: filteredZones,
                        total: filteredZones.length,
                    }
                },
            },
            {
                url: 'api/zones?start=s&limit=l&zoneType=1&zoneType=2&zoneType=3&zoneType=4&zoneType=5&zoneType=6&zoneType=7&zoneType=8&zoneType=9&zoneType=10',
                method: 'GET',
                status: 200,
                response: ({ url }) => {
                    const filteredZones = filterByZoneType(url)
                    return {
                        items: filteredZones,
                        total: filteredZones.length,
                    }
                },
            },
            {
                url: 'api/zones?start=s&limit=l&zoneType=1&zoneType=2&zoneType=3&zoneType=4&zoneType=5&zoneType=6&zoneType=7&zoneType=8&zoneType=9&zoneType=10&zoneType=11',
                method: 'GET',
                status: 200,
                response: ({ url }) => {
                    const filteredZones = filterByZoneType(url)
                    return {
                        items: filteredZones,
                        total: filteredZones.length,
                    }
                },
            },
            {
                url: 'api/zones?start=s&limit=l&zoneType=1&zoneType=2&zoneType=3&zoneType=4&zoneType=5&zoneType=6&zoneType=7&zoneType=8&zoneType=9&zoneType=10&zoneType=11&zoneType=12',
                method: 'GET',
                status: 200,
                response: ({ url }) => {
                    const filteredZones = filterByZoneType(url)
                    return {
                        items: filteredZones,
                        total: filteredZones.length,
                    }
                },
            },
            {
                url: 'api/zones?start=s&limit=l&zoneType=1&zoneType=2&zoneType=3&zoneType=4&zoneType=5&zoneType=6&zoneType=7&zoneType=8&zoneType=9&zoneType=10&zoneType=11&zoneType=12&zoneType=13',
                method: 'GET',
                status: 200,
                response: ({ url }) => {
                    const filteredZones = filterByZoneType(url)
                    return {
                        items: filteredZones,
                        total: filteredZones.length,
                    }
                },
            },
            {
                url: 'api/zones?start=s&limit=l&zoneType=1&zoneType=2&zoneType=3&zoneType=4&zoneType=5&zoneType=6&zoneType=7&zoneType=8&zoneType=9&zoneType=10&zoneType=11&zoneType=12&appId=a',
                method: 'GET',
                status: 200,
                response: (req) => {
                    let filteredZones = filterByZoneType(req.url)
                    filteredZones = filteredZones.filter((z) => {
                        return z.localZones.some((lz) => lz.appId == req.searchParams?.appId)
                    })
                    return { items: filteredZones, total: filteredZones.length }
                },
            },
            {
                url: 'api/zones?start=s&limit=l&zoneType=1&zoneType=2&zoneType=3&zoneType=4&zoneType=5&zoneType=6&zoneType=7&zoneType=8&zoneType=9&zoneType=10&zoneType=11&zoneType=12&zoneType=13&appId=a',
                method: 'GET',
                status: 200,
                response: (req) => {
                    let filteredZones = filterByZoneType(req.url)
                    filteredZones = filteredZones.filter((z) => {
                        return z.localZones.some((lz) => lz.appId == req.searchParams?.appId)
                    })
                    return { items: filteredZones, total: filteredZones.length }
                },
            },
            {
                url: 'api/zones?start=s&limit=l&zoneType=1&zoneType=2&zoneType=3&zoneType=4&zoneType=5&zoneType=6&zoneType=7&zoneType=8&zoneType=9&zoneType=10&zoneType=11&zoneType=12&rpz=r',
                method: 'GET',
                status: 200,
                response: (req) => {
                    let filteredZones = filterByZoneType(req.url)
                    filteredZones = filteredZones.filter((z) => {
                        return z.localZones.some((lz) => (!!lz.rpz).toString() == req.searchParams?.rpz)
                    })
                    return { items: filteredZones, total: filteredZones.length }
                },
            },
            {
                url: 'api/zones?start=s&limit=l&zoneType=1&zoneType=2&zoneType=3&zoneType=4&zoneType=5&zoneType=6&zoneType=7&zoneType=8&zoneType=9&zoneType=10&zoneType=11&zoneType=12&zoneType=13&rpz=r',
                method: 'GET',
                status: 200,
                response: (req) => {
                    let filteredZones = filterByZoneType(req.url)
                    filteredZones = filteredZones.filter((z) => {
                        return z.localZones.some((lz) => (!!lz.rpz).toString() == req.searchParams?.rpz)
                    })
                    return { items: filteredZones, total: filteredZones.length }
                },
            },
            {
                url: 'api/zones?start=s&limit=l&zoneType=1&zoneType=2&zoneType=3&zoneType=4&zoneType=5&zoneType=6&zoneType=7&zoneType=8&zoneType=9&zoneType=10&zoneType=11&zoneType=12&serial=s',
                method: 'GET',
                status: 200,
                response: (req) => {
                    let filteredZones = filterByZoneType(req.url)
                    filteredZones = filteredZones.filter((z) => {
                        return z.localZones.some((lz) => lz.serial.toString().includes(req.searchParams?.serial))
                    })
                    return { items: filteredZones, total: filteredZones.length }
                },
            },
            {
                url: 'api/zones?start=s&limit=l&zoneType=1&zoneType=2&zoneType=3&zoneType=4&zoneType=5&zoneType=6&zoneType=7&zoneType=8&zoneType=9&zoneType=10&zoneType=11&zoneType=12&zoneType=13&serial=s',
                method: 'GET',
                status: 200,
                response: (req) => {
                    let filteredZones = filterByZoneType(req.url)
                    filteredZones = filteredZones.filter((z) => {
                        return z.localZones.some((lz) => lz.serial.toString().includes(req.searchParams?.serial))
                    })
                    return { items: filteredZones, total: filteredZones.length }
                },
            },
            {
                url: 'api/zones?start=s&limit=l&zoneType=1&zoneType=2&zoneType=3&zoneType=4&zoneType=5&zoneType=6&zoneType=7&zoneType=8&zoneType=9&zoneType=10&zoneType=11&zoneType=12&class=c',
                method: 'GET',
                status: 200,
                response: (req) => {
                    let filteredZones = filterByZoneType(req.url)
                    filteredZones = filteredZones.filter((z) => {
                        return z.localZones.some((lz: TestLocalZone) => lz.class == req.searchParams?.class)
                    })
                    return { items: filteredZones, total: filteredZones.length }
                },
            },
            {
                url: 'api/zones?start=s&limit=l&zoneType=1&zoneType=2&zoneType=3&zoneType=4&zoneType=5&zoneType=6&zoneType=7&zoneType=8&zoneType=9&zoneType=10&zoneType=11&zoneType=12&zoneType=13&class=c',
                method: 'GET',
                status: 200,
                response: (req) => {
                    let filteredZones = filterByZoneType(req.url)
                    filteredZones = filteredZones.filter((z) => {
                        return z.localZones.some((lz: TestLocalZone) => lz.class == req.searchParams?.class)
                    })
                    return { items: filteredZones, total: filteredZones.length }
                },
            },
            {
                url: 'api/zones?start=s&limit=l&appType=a&zoneType=1&zoneType=2&zoneType=3&zoneType=4&zoneType=5&zoneType=6&zoneType=7&zoneType=8&zoneType=9&zoneType=10&zoneType=11&zoneType=12',
                method: 'GET',
                status: 200,
                response: (req) => {
                    let filteredZones = filterByZoneType(req.url)
                    filteredZones = filteredZones.filter((z) =>
                        z.localZones.some((lz) => lz.appName.indexOf(req.searchParams?.appType) == 0)
                    )
                    return { items: filteredZones, total: filteredZones.length }
                },
            },
            {
                url: 'api/zones?start=s&limit=l&appType=a&zoneType=1&zoneType=2&zoneType=3&zoneType=4&zoneType=5&zoneType=6&zoneType=7&zoneType=8&zoneType=9&zoneType=10&zoneType=11&zoneType=12&zoneType=13',
                method: 'GET',
                status: 200,
                response: (req) => {
                    let filteredZones = filterByZoneType(req.url)
                    filteredZones = filteredZones.filter((z) =>
                        z.localZones.some((lz) => lz.appName.indexOf(req.searchParams?.appType) == 0)
                    )
                    return { items: filteredZones, total: filteredZones.length }
                },
            },
            {
                url: 'api/zones?start=s&limit=l&zoneType=1&zoneType=2&zoneType=3&zoneType=4&zoneType=5&zoneType=6&zoneType=7&zoneType=8&zoneType=9&zoneType=10&zoneType=11&zoneType=12&text=t',
                method: 'GET',
                status: 200,
                response: (req) => {
                    const filteredZones = filterByZoneType(req.url)
                    const resp = { items: filteredZones, total: filteredZones.length }
                    return mockedFilterByText(resp, req, 'name')
                },
            },
            {
                url: 'api/zones?start=s&limit=l&zoneType=1&zoneType=2&zoneType=3&zoneType=4&zoneType=5&zoneType=6&zoneType=7&zoneType=8&zoneType=9&zoneType=10&zoneType=11&zoneType=12&zoneType=13&text=t',
                method: 'GET',
                status: 200,
                response: (req) => {
                    const filteredZones = filterByZoneType(req.url)
                    const resp = { items: filteredZones, total: filteredZones.length }
                    return mockedFilterByText(resp, req, 'name')
                },
            },
        ],
    },
}

export const TestAllZonesShown: Story = {
    globals: {
        role: 'super-admin',
    },
    parameters: ListZones.parameters,
    play: async ({ canvasElement }) => {
        // Arrange
        const canvas = within(canvasElement)
        const clearFiltersBtn = await canvas.findByRole('button', { name: 'Clear' })
        const table = await canvas.findByRole('table')

        // Act
        await userEvent.click(clearFiltersBtn)

        // Assert
        // At first, builtin zones should be hidden.
        await expect(await canvas.findAllByRole('row')).toHaveLength(allZones.length + 1 - builtinZones.length) // All rows in tbody + one row in the thead.
        await expect(canvas.getByText('(root)')).toBeInTheDocument()
        await expect(canvas.getByText(primaryZones[0].name)).toBeInTheDocument()
        await expect(canvas.getByText(primaryZones[1].name)).toBeInTheDocument()
        await expect(canvas.getByText(primaryZones[4].name)).toBeInTheDocument()
        await expect(canvas.getByText(primaryZones[5].name)).toBeInTheDocument()
        canvas.getAllByText(primaryZones[0].localZones[0].serial).forEach((el) => expect(el).toBeInTheDocument())
        await expect(within(table).getAllByText('RPZ')).toHaveLength(2)

        // Check expanding the row.
        const allCells = await canvas.findAllByRole('cell')
        const expandRootZoneRow = await within(allCells[0]).findByRole('button')
        await userEvent.click(expandRootZoneRow)
        await expect(within(table).getByText(rootZone.localZones[0].appName)).toBeInTheDocument()
        await expect(within(table).getByText(rootZone.localZones[0].view)).toBeInTheDocument()
        await userEvent.click(expandRootZoneRow)

        // Toggle builtin zones.
        const toggleBuiltinZones = await canvas.findByRole('button', { name: 'Toggle builtin zones' })
        await userEvent.click(toggleBuiltinZones)
        await expect(await canvas.findAllByRole('row')).toHaveLength(allZones.length + 1) // All rows in tbody + one row in the thead.
        await expect(canvas.getByText(builtinZones[0].name)).toBeInTheDocument()
        await expect(canvas.getByText(builtinZones[1].name)).toBeInTheDocument()
        await expect(canvas.getByText(builtinZones[2].name)).toBeInTheDocument()
        await canvas.findAllByText('builtin')

        await userEvent.click(clearFiltersBtn)
    },
}

export const TestZonesFiltering: Story = {
    globals: {
        role: 'super-admin',
    },
    parameters: ListZones.parameters,
    play: async ({ canvasElement }) => {
        // Arrange
        const canvas = within(canvasElement)
        const clearFiltersBtn = await canvas.findByRole('button', { name: 'Clear' })
        const table = await canvas.findByRole('table')
        let spanID = canvas.getByText('App Type').getAttribute('for')
        const comboboxes = canvas.getAllByRole('combobox') // PrimeNG p-select component has combobox role.
        let selectSpan = comboboxes.find((el) => el.getAttribute('id') == spanID)
        await expect(selectSpan).toBeTruthy()

        // Act
        // Filter only BIND9 zones.
        await userEvent.click(clearFiltersBtn)
        await userEvent.click(selectSpan)
        const bindOption = await canvas.findByRole('option', { name: 'BIND9' })
        await userEvent.click(bindOption)

        // Assert
        // 3 BIND9 zones are expected.
        await waitFor(() => expect(within(table).getAllByRole('row')).toHaveLength(4)) // All rows in tbody + one row in the thead.
        await expect(within(table).getByText('(root)')).toBeInTheDocument()
        await expect(within(table).getByText(primaryZones[4].name)).toBeInTheDocument()
        await expect(within(table).getByText(primaryZones[5].name)).toBeInTheDocument()

        // Toggle builtin zones.
        const toggleBuiltinZones = await canvas.findByRole('button', { name: 'Toggle builtin zones' })
        await userEvent.click(toggleBuiltinZones)
        // 3 additional (6 in total) BIND9 zones are expected.
        await waitFor(() => expect(within(table).getAllByRole('row')).toHaveLength(7)) // All rows in tbody + one row in the thead.
        await expect(within(table).getByText(builtinZones[0].name)).toBeInTheDocument()
        await expect(within(table).getByText(builtinZones[1].name)).toBeInTheDocument()
        await expect(within(table).getByText(builtinZones[2].name)).toBeInTheDocument()

        // Filter only PowerDNS zones.
        await userEvent.click(clearFiltersBtn)
        await userEvent.click(selectSpan)
        const pdnsOption = await canvas.findByRole('option', { name: 'PDNS' })
        await userEvent.click(pdnsOption)

        // 4 PowerDNS zones are expected.
        await waitFor(() => expect(within(table).getAllByRole('row')).toHaveLength(5)) // All rows in tbody + one row in the thead.
        await expect(within(table).getByText(primaryZones[0].name)).toBeInTheDocument()
        await expect(within(table).getByText(primaryZones[1].name)).toBeInTheDocument()
        await expect(within(table).getByText(primaryZones[2].name)).toBeInTheDocument()
        await expect(within(table).getByText(primaryZones[3].name)).toBeInTheDocument()

        // Check filtering by class.
        await userEvent.click(clearFiltersBtn)
        await userEvent.click(toggleBuiltinZones)
        spanID = canvas.getByText('Zone Class').getAttribute('for')
        selectSpan = comboboxes.find((el) => el.getAttribute('id') == spanID)
        await expect(selectSpan).toBeTruthy()

        // Filter by IN class.
        await userEvent.click(selectSpan)
        const inOption = await canvas.findByRole('option', { name: 'IN' })
        await userEvent.click(inOption)

        // 9 zones are expected.
        await waitFor(() => expect(within(table).getAllByRole('row')).toHaveLength(10)) // All rows in tbody + one row in the thead.

        // Filter by CH class.
        await userEvent.click(selectSpan)
        const chOption = await canvas.findByRole('option', { name: 'CH' })
        await userEvent.click(chOption)

        // 1 zone is expected.
        await waitFor(() => expect(within(table).getAllByRole('row')).toHaveLength(2)) // All rows in tbody + one row in the thead.

        // Check filtering by RPZ.
        await userEvent.click(clearFiltersBtn)
        await userEvent.click(toggleBuiltinZones)
        spanID = canvas.getAllByText('RPZ')[0].getAttribute('for')
        selectSpan = comboboxes.find((el) => el.getAttribute('id') == spanID)
        await expect(selectSpan).toBeTruthy()

        // Filter by RPZ - include.
        await userEvent.click(selectSpan)
        const includeOption = await canvas.findByRole('option', { name: 'include' })
        await userEvent.click(includeOption)

        // All 10 zones are expected.
        await waitFor(() => expect(within(table).getAllByRole('row')).toHaveLength(11)) // All rows in tbody + one row in the thead.
        await expect(within(table).queryAllByText('RPZ')).toHaveLength(2)

        // Filter by RPZ - exclude.
        await userEvent.click(selectSpan)
        const excludeOption = await canvas.findByRole('option', { name: 'exclude' })
        await userEvent.click(excludeOption)

        // 8 zones are expected.
        await waitFor(() => expect(within(table).getAllByRole('row')).toHaveLength(9)) // All rows in tbody + one row in the thead.
        await expect(within(table).queryAllByText('RPZ')).toHaveLength(0)

        // Filter by RPZ - only.
        await userEvent.click(selectSpan)
        const onlyOption = await canvas.findByRole('option', { name: 'only' })
        await userEvent.click(onlyOption)

        // 2 zones are expected.
        await waitFor(() => expect(within(table).getAllByRole('row')).toHaveLength(3)) // All rows in tbody + one row in the thead.
        await expect(within(table).queryAllByText('RPZ')).toHaveLength(2)

        await userEvent.click(clearFiltersBtn)
    },
}
