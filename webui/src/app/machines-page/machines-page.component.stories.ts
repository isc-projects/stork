import { MachinesPageComponent } from './machines-page.component'
import { applicationConfig, Meta, moduleMetadata, StoryObj } from '@storybook/angular'
import { provideHttpClient, withInterceptorsFromDi } from '@angular/common/http'
import { ConfirmationService, MessageService } from 'primeng/api'
import { BreadcrumbsComponent } from '../breadcrumbs/breadcrumbs.component'
import { DialogModule } from 'primeng/dialog'
import { ButtonModule } from 'primeng/button'
import { TabViewComponent } from '../tab-view/tab-view.component'
import { ConfirmDialogModule } from 'primeng/confirmdialog'
import { provideRouter, RouterModule } from '@angular/router'
import { MachinesTableComponent } from '../machines-table/machines-table.component'
import { BreadcrumbModule } from 'primeng/breadcrumb'
import { TableModule } from 'primeng/table'
import { PluralizePipe } from '../pipes/pluralize.pipe'
import { PanelModule } from 'primeng/panel'
import { IconFieldModule } from 'primeng/iconfield'
import { InputIconModule } from 'primeng/inputicon'
import { HelpTipComponent } from '../help-tip/help-tip.component'
import { TriStateCheckboxComponent } from '../tri-state-checkbox/tri-state-checkbox.component'
import { PopoverModule } from 'primeng/popover'
import { SelectButtonModule } from 'primeng/selectbutton'
import { MenuModule } from 'primeng/menu'
import { InputTextModule } from 'primeng/inputtext'
import { BadgeModule } from 'primeng/badge'
import { TagModule } from 'primeng/tag'
import { FormsModule } from '@angular/forms'
import { MessageServiceMock, mockedFilterByText, toastDecorator } from '../utils-stories'
import { ToastModule } from 'primeng/toast'
import { VersionStatusComponent } from '../version-status/version-status.component'
import { ProgressBarModule } from 'primeng/progressbar'
import { MessageModule } from 'primeng/message'
import { TooltipModule } from 'primeng/tooltip'
import { AppDaemonsStatusComponent } from '../app-daemons-status/app-daemons-status.component'
import { Severity, VersionService } from '../version.service'
import { of } from 'rxjs'
import { AppsVersions } from '../backend'
import { LocaltimePipe } from '../pipes/localtime.pipe'
import { PlaceholderPipe } from '../pipes/placeholder.pipe'
import { userEvent, within, expect, waitFor } from '@storybook/test'

const meta: Meta<MachinesPageComponent> = {
    title: 'App/MachinesPage',
    component: MachinesPageComponent,
    subcomponents: MachinesTableComponent,
    decorators: [
        applicationConfig({
            providers: [
                provideHttpClient(withInterceptorsFromDi()),
                { provide: MessageService, useClass: MessageServiceMock },
                ConfirmationService,
                provideRouter([
                    {
                        path: 'machines',
                        pathMatch: 'full',
                        redirectTo: 'machines/all',
                    },
                    {
                        path: 'machines/:id',
                        component: MachinesPageComponent,
                    },
                    {
                        path: 'iframe.html',
                        component: MachinesPageComponent,
                    },
                ]),
                {
                    provide: VersionService,
                    useValue: {
                        sanitizeSemver: () => '3.2.1',
                        getCurrentData: () => of({} as AppsVersions),
                        getSoftwareVersionFeedback: () => ({ severity: Severity.success, messages: ['test feedback'] }),
                    },
                },
            ],
        }),
        moduleMetadata({
            declarations: [
                BreadcrumbsComponent,
                MachinesTableComponent,
                PluralizePipe,
                HelpTipComponent,
                VersionStatusComponent,
                AppDaemonsStatusComponent,
                LocaltimePipe,
                PlaceholderPipe,
            ],
            imports: [
                DialogModule,
                ButtonModule,
                TabViewComponent,
                ConfirmDialogModule,
                BreadcrumbModule,
                TableModule,
                PanelModule,
                IconFieldModule,
                InputIconModule,
                TriStateCheckboxComponent,
                PopoverModule,
                SelectButtonModule,
                MenuModule,
                InputTextModule,
                BadgeModule,
                TagModule,
                FormsModule,
                ToastModule,
                ProgressBarModule,
                MessageModule,
                TooltipModule,
                RouterModule,
            ],
        }),
        toastDecorator,
    ],
    args: {
        registrationDisabled: false,
    },
}

export default meta
type Story = StoryObj<MachinesPageComponent>
const mockedUnauthorizedMachines = [
    {
        address: 'agent-kea-large',
        agentPort: 8884,
        agentToken: 'random-agent-kea-large',
        apps: [],
        id: 4,
    },
    {
        address: 'agent-pdns',
        agentPort: 8891,
        agentToken: 'random-agent-pdns-token',
        apps: [],
        id: 1,
    },
    {
        address: 'agent-bind9',
        agentPort: 8883,
        agentToken: 'random-agent-bind9',
        agentVersion: '2.3.0',
        apps: [],
        id: 2,
    },
]
const mockedAuthorizedMachines = [
    {
        address: 'agent-bind9-2',
        agentPort: 8882,
        agentToken: 'random-agent-bind9-2',
        agentVersion: '2.3.0',
        apps: [
            {
                accessPoints: [
                    { address: '127.0.0.1', port: 953, type: 'control' },
                    { address: '127.0.0.1', port: 8053, type: 'statistics' },
                ],
                details: {
                    daemons: [],
                    daemon: {
                        active: true,
                        autoZoneCount: 100,
                        id: 3,
                        monitored: true,
                        name: 'named',
                        reloadedAt: '2025-10-14T15:09:54.000Z',
                        uptime: 19715,
                        version: 'BIND 9.20.13 (Stable Release) <id:1f79fb9>',
                        views: [{ name: '_default', queryHits: 0, queryMisses: 0 }],
                        zoneCount: 6,
                    },
                },
                id: 3,
                machine: { id: 3 },
                name: 'bind9@agent-bind9-2',
                type: 'bind9',
                version: 'BIND 9.20.13 (Stable Release) <id:1f79fb9>',
            },
        ],
        authorized: true,
        cpus: 12,
        cpusLoad: '0.89 0.84 0.86',
        hostID: 'agent-bind9-2-host-id',
        hostname: 'agent-bind9-2',
        id: 3,
        kernelArch: 'x86_64',
        kernelVersion: '6.10.14-linuxkit',
        lastVisitedAt: '2025-10-14T20:38:29.104Z',
        memory: 7,
        os: 'linux',
        platform: 'alpine',
        platformFamily: 'alpine',
        platformVersion: '3.22.1',
        usedMemory: 69,
        virtualizationRole: 'guest',
        virtualizationSystem: 'docker',
    },
    {
        address: 'agent-kea6',
        agentPort: 8887,
        agentToken: 'random-agent-kea6',
        agentVersion: '2.3.0',
        apps: [
            {
                accessPoints: [{ address: '127.0.0.1', port: 8000, type: 'control' }],
                details: {
                    daemons: [
                        {
                            active: true,
                            backends: [],
                            extendedVersion:
                                '3.1.0 (3.1.0 (isc20250728104543 deb))\npremium: yes (isc20250728104543 deb)\nlinked with:\n- log4cplus 2.0.8\n- OpenSSL 3.0.17 1 Jul 2025',
                            files: [],
                            hooks: [],
                            id: 4,
                            logTargets: [],
                            monitored: true,
                            name: 'ca',
                            version: '3.1.0',
                        },
                        {
                            active: true,
                            backends: [
                                {
                                    backendType: 'postgresql',
                                    dataTypes: ['Leases'],
                                    database: 'agent_kea6',
                                    host: 'postgres',
                                },
                            ],
                            extendedVersion:
                                '3.1.0 (3.1.0 (isc20250728104543 deb))\npremium: yes (isc20250728104543 deb)\nlinked with:\n- log4cplus 2.0.8\n- OpenSSL 3.0.17 1 Jul 2025\nlease backends:\n- Memfile backend 5.0\n- PostgreSQL backend 30.0, library 150013\nhost backends:\n- PostgreSQL backend 30.0, library 150013\nforensic backends:\n- PostgreSQL backend 30.0, library 150013',
                            files: [{ filename: '/var/log/kea', filetype: 'Forensic Logging', persist: true }],
                            hooks: [
                                'libdhcp_lease_cmds.so',
                                'libdhcp_pgsql.so',
                                'libdhcp_legal_log.so',
                                'libdhcp_subnet_cmds.so',
                            ],
                            id: 5,
                            logTargets: [],
                            monitored: true,
                            name: 'dhcp6',
                            reloadedAt: '2025-10-14T15:10:00.066Z',
                            uptime: 19709,
                            version: '3.1.0',
                        },
                    ],
                },
                id: 4,
                machine: { id: 5 },
                name: 'kea@agent-kea6',
                type: 'kea',
                version: '3.1.0',
            },
        ],
        authorized: true,
        cpus: 12,
        cpusLoad: '0.89 0.84 0.86',
        hostID: 'agent-kea6-host-id',
        hostname: 'agent-kea6',
        id: 5,
        kernelArch: 'aarch64',
        kernelVersion: '6.10.14-linuxkit',
        lastVisitedAt: '2025-10-14T20:38:29.058Z',
        memory: 7,
        os: 'linux',
        platform: 'debian',
        platformFamily: 'debian',
        platformVersion: '12.11',
        usedMemory: 59,
        virtualizationRole: 'guest',
        virtualizationSystem: 'docker',
    },
    {
        address: 'agent-kea',
        agentPort: 8888,
        agentToken: 'random-agent-kea',
        agentVersion: '2.3.0',
        apps: [
            {
                accessPoints: [{ address: '127.0.0.1', port: 8000, type: 'control' }],
                details: {
                    daemons: [
                        { backends: [], files: [], hooks: [], id: 9, logTargets: [], name: 'dhcp6' },
                        { backends: [], files: [], hooks: [], id: 7, logTargets: [], name: 'd2' },
                        {
                            active: true,
                            backends: [
                                {
                                    backendType: 'mysql',
                                    dataTypes: ['Leases', 'Config Backend'],
                                    database: 'agent_kea',
                                    host: 'mariadb',
                                },
                            ],
                            extendedVersion:
                                '3.1.0 (3.1.0 (isc20250728104543 deb))\npremium: yes (isc20250728104543 deb)\nlinked with:\n- log4cplus 2.0.8\n- OpenSSL 3.0.17 1 Jul 2025\nlease backends:\n- Memfile backend 3.0\n- MySQL backend 31.0, library 3.3.14\nhost backends:\n- MySQL backend 31.0, library 3.3.14\nforensic backends:\n- MySQL backend 31.0, library 3.3.14',
                            files: [{ filename: '/var/log/kea', filetype: 'Forensic Logging', persist: true }],
                            hooks: [
                                'libdhcp_lease_cmds.so',
                                'libdhcp_stat_cmds.so',
                                'libdhcp_mysql.so',
                                'libdhcp_legal_log.so',
                                'libdhcp_subnet_cmds.so',
                            ],
                            id: 8,
                            logTargets: [],
                            monitored: true,
                            name: 'dhcp4',
                            reloadedAt: '2025-10-14T15:10:07.212Z',
                            uptime: 19703,
                            version: '3.1.0',
                        },
                        {
                            active: true,
                            backends: [],
                            extendedVersion:
                                '3.1.0 (3.1.0 (isc20250728104543 deb))\npremium: yes (isc20250728104543 deb)\nlinked with:\n- log4cplus 2.0.8\n- OpenSSL 3.0.17 1 Jul 2025',
                            files: [],
                            hooks: [],
                            id: 6,
                            logTargets: [],
                            monitored: true,
                            name: 'ca',
                            version: '3.1.0',
                        },
                    ],
                },
                id: 5,
                machine: { id: 6 },
                name: 'kea@agent-kea',
                type: 'kea',
                version: '3.1.0',
            },
        ],
        authorized: true,
        cpus: 12,
        cpusLoad: '0.89 0.84 0.86',
        hostID: 'agent-kea-host-id',
        hostname: 'agent-kea',
        id: 6,
        kernelArch: 'aarch64',
        kernelVersion: '6.10.14-linuxkit',
        lastVisitedAt: '2025-10-14T20:38:29.206Z',
        memory: 7,
        os: 'linux',
        platform: 'debian',
        platformFamily: 'debian',
        platformVersion: '12.11',
        usedMemory: 49,
        virtualizationRole: 'guest',
        virtualizationSystem: 'docker',
    },
    {
        address: 'agent-kea-ha1',
        agentPort: 8886,
        agentToken: 'random-agent-kea-ha1',
        agentVersion: '2.3.0',
        apps: [
            {
                accessPoints: [{ address: '127.0.0.1', port: 8001, type: 'control' }],
                details: {
                    daemons: [
                        { backends: [], files: [], hooks: [], id: 11, logTargets: [], name: 'd2' },
                        {
                            active: true,
                            backends: [
                                {
                                    backendType: 'mysql',
                                    dataTypes: ['Host Reservations'],
                                    database: 'agent_kea_ha1',
                                    host: 'mariadb',
                                },
                            ],
                            extendedVersion:
                                '3.1.0 (3.1.0 (isc20250728104543 deb))\npremium: yes (isc20250728104543 deb)\nlinked with:\n- log4cplus 2.0.8\n- OpenSSL 3.0.17 1 Jul 2025\nlease backends:\n- Memfile backend 3.0\n- MySQL backend 31.0, library 3.3.14\nhost backends:\n- MySQL backend 31.0, library 3.3.14\nforensic backends:\n- MySQL backend 31.0, library 3.3.14',
                            files: [{ filetype: 'Lease file', persist: true }],
                            hooks: [
                                'libdhcp_lease_cmds.so',
                                'libdhcp_host_cmds.so',
                                'libdhcp_subnet_cmds.so',
                                'libdhcp_mysql.so',
                                'libdhcp_ha.so',
                            ],
                            id: 12,
                            logTargets: [],
                            monitored: true,
                            name: 'dhcp4',
                            reloadedAt: '2025-10-14T15:10:07.186Z',
                            uptime: 19703,
                            version: '3.1.0',
                        },
                        { backends: [], files: [], hooks: [], id: 13, logTargets: [], name: 'dhcp6' },
                        {
                            active: true,
                            backends: [],
                            extendedVersion:
                                '3.1.0 (3.1.0 (isc20250728104543 deb))\npremium: yes (isc20250728104543 deb)\nlinked with:\n- log4cplus 2.0.8\n- OpenSSL 3.0.17 1 Jul 2025',
                            files: [],
                            hooks: [],
                            id: 10,
                            logTargets: [],
                            monitored: true,
                            name: 'ca',
                            version: '3.1.0',
                        },
                    ],
                },
                id: 6,
                machine: { id: 7 },
                name: 'kea@agent-kea-ha1',
                type: 'kea',
                version: '3.1.0',
            },
        ],
        authorized: true,
        cpus: 12,
        cpusLoad: '0.89 0.84 0.86',
        hostID: 'agent-kea-ha1-host-id',
        hostname: 'agent-kea-ha1',
        id: 7,
        kernelArch: 'aarch64',
        kernelVersion: '6.10.14-linuxkit',
        lastVisitedAt: '2025-10-14T20:38:29.180Z',
        memory: 7,
        os: 'linux',
        platform: 'debian',
        platformFamily: 'debian',
        platformVersion: '12.11',
        usedMemory: 39,
        virtualizationRole: 'guest',
        virtualizationSystem: 'docker',
    },
    {
        address: 'agent-kea-ha3',
        agentPort: 8890,
        agentToken: 'random-agent-kea-ha3',
        agentVersion: '2.3.0',
        apps: [
            {
                accessPoints: [{ address: '127.0.0.1', port: 8000, type: 'control' }],
                details: {
                    daemons: [
                        { backends: [], files: [], hooks: [], id: 16, logTargets: [], name: 'd2' },
                        {
                            active: true,
                            backends: [
                                {
                                    backendType: 'mysql',
                                    dataTypes: ['Host Reservations'],
                                    database: 'agent_kea_ha3',
                                    host: 'mariadb',
                                },
                            ],
                            extendedVersion:
                                '3.1.0 (3.1.0 (isc20250728104543 deb))\npremium: yes (isc20250728104543 deb)\nlinked with:\n- log4cplus 2.0.8\n- OpenSSL 3.0.17 1 Jul 2025\nlease backends:\n- Memfile backend 3.0\n- MySQL backend 31.0, library 3.3.14\nhost backends:\n- MySQL backend 31.0, library 3.3.14\nforensic backends:\n- MySQL backend 31.0, library 3.3.14',
                            files: [{ filetype: 'Lease file', persist: true }],
                            hooks: [
                                'libdhcp_lease_cmds.so',
                                'libdhcp_host_cmds.so',
                                'libdhcp_subnet_cmds.so',
                                'libdhcp_mysql.so',
                                'libdhcp_ha.so',
                            ],
                            id: 17,
                            logTargets: [],
                            monitored: true,
                            name: 'dhcp4',
                            reloadedAt: '2025-10-14T15:10:07.086Z',
                            uptime: 19703,
                            version: '3.1.0',
                        },
                        { backends: [], files: [], hooks: [], id: 14, logTargets: [], name: 'dhcp6' },
                        {
                            active: true,
                            backends: [],
                            extendedVersion:
                                '3.1.0 (3.1.0 (isc20250728104543 deb))\npremium: yes (isc20250728104543 deb)\nlinked with:\n- log4cplus 2.0.8\n- OpenSSL 3.0.17 1 Jul 2025',
                            files: [],
                            hooks: [],
                            id: 15,
                            logTargets: [],
                            monitored: true,
                            name: 'ca',
                            version: '3.1.0',
                        },
                    ],
                },
                id: 7,
                machine: { id: 8 },
                name: 'kea@agent-kea-ha3',
                type: 'kea',
                version: '3.1.0',
            },
        ],
        authorized: true,
        cpus: 12,
        cpusLoad: '0.89 0.84 0.86',
        hostID: 'agent-kea-ha3-host-id',
        hostname: 'agent-kea-ha3',
        id: 8,
        kernelArch: 'aarch64',
        kernelVersion: '6.10.14-linuxkit',
        lastVisitedAt: '2025-10-14T20:38:29.081Z',
        memory: 7,
        os: 'linux',
        platform: 'debian',
        platformFamily: 'debian',
        platformVersion: '12.11',
        usedMemory: 29,
        virtualizationRole: 'guest',
        virtualizationSystem: 'docker',
    },
    {
        address: 'agent-kea-ha2',
        agentPort: 8885,
        agentToken: 'random-agent-kea-ha2',
        agentVersion: '2.3.0',
        apps: [
            {
                accessPoints: [{ address: '127.0.0.1', port: 8002, type: 'control' }],
                details: {
                    daemons: [
                        { backends: [], files: [], hooks: [], id: 19, logTargets: [], name: 'd2' },
                        {
                            active: true,
                            backends: [
                                {
                                    backendType: 'mysql',
                                    dataTypes: ['Host Reservations'],
                                    database: 'agent_kea_ha2',
                                    host: 'mariadb',
                                },
                            ],
                            extendedVersion:
                                '3.1.0 (3.1.0 (isc20250728104543 deb))\npremium: yes (isc20250728104543 deb)\nlinked with:\n- log4cplus 2.0.8\n- OpenSSL 3.0.17 1 Jul 2025\nlease backends:\n- Memfile backend 3.0\n- MySQL backend 31.0, library 3.3.14\nhost backends:\n- MySQL backend 31.0, library 3.3.14\nforensic backends:\n- MySQL backend 31.0, library 3.3.14',
                            files: [{ filetype: 'Lease file', persist: true }],
                            hooks: [
                                'libdhcp_lease_cmds.so',
                                'libdhcp_host_cmds.so',
                                'libdhcp_subnet_cmds.so',
                                'libdhcp_mysql.so',
                                'libdhcp_ha.so',
                            ],
                            id: 20,
                            logTargets: [],
                            monitored: true,
                            name: 'dhcp4',
                            reloadedAt: '2025-10-14T15:10:08.232Z',
                            uptime: 19701,
                            version: '3.1.0',
                        },
                        { backends: [], files: [], hooks: [], id: 21, logTargets: [], name: 'dhcp6' },
                        {
                            active: true,
                            backends: [],
                            extendedVersion:
                                '3.1.0 (3.1.0 (isc20250728104543 deb))\npremium: yes (isc20250728104543 deb)\nlinked with:\n- log4cplus 2.0.8\n- OpenSSL 3.0.17 1 Jul 2025',
                            files: [],
                            hooks: [],
                            id: 18,
                            logTargets: [],
                            monitored: true,
                            name: 'ca',
                            version: '3.1.0',
                        },
                    ],
                },
                id: 8,
                machine: { id: 9 },
                name: 'kea@agent-kea-ha2',
                type: 'kea',
                version: '3.1.0',
            },
        ],
        authorized: true,
        cpus: 12,
        cpusLoad: '0.89 0.84 0.86',
        hostID: 'agent-kea-ha2-host-id',
        hostname: 'agent-kea-ha2',
        id: 9,
        kernelArch: 'aarch64',
        kernelVersion: '6.10.14-linuxkit',
        lastVisitedAt: '2025-10-14T20:38:29.228Z',
        memory: 7,
        os: 'linux',
        platform: 'debian',
        platformFamily: 'debian',
        platformVersion: '12.11',
        usedMemory: 89,
        virtualizationRole: 'guest',
        virtualizationSystem: 'docker',
    },
]
const mockedAllRespData = {
    items: [...mockedAuthorizedMachines, ...mockedUnauthorizedMachines],
    total: mockedAuthorizedMachines.length + mockedUnauthorizedMachines.length,
}
const mockedAuthorizedRespData = {
    items: mockedAuthorizedMachines,
    total: mockedAuthorizedMachines.length,
}
const mockedUnauthorizedRespData = {
    items: mockedUnauthorizedMachines,
    total: mockedUnauthorizedMachines.length,
}

export const EmptyList: Story = {
    parameters: {
        mockData: [
            {
                url: 'http://localhost/api/machines?start=:start&limit=:limit',
                method: 'GET',
                status: 200,
                response: () => ({ items: [] }),
            },
            {
                url: 'http://localhost/api/machines?start=:start&limit=:limit&text=:text',
                method: 'GET',
                status: 200,
                response: () => ({ items: [] }),
            },
            {
                url: 'http://localhost/api/machines?start=:start&limit=:limit&text=:text&authorized=:authorized',
                method: 'GET',
                status: 200,
                response: () => ({ items: [] }),
            },
            {
                url: 'http://localhost/api/machines?start=:start&limit=:limit&authorized=:authorized',
                method: 'GET',
                status: 200,
                response: () => ({ items: [] }),
            },
            {
                url: 'http://localhost/api/machines/unauthorized/count',
                method: 'GET',
                status: 200,
                response: () => 0,
            },
            {
                url: 'http://localhost/api/settings',
                method: 'GET',
                status: 200,
                response: () => ({ enableMachineRegistration: !meta.args.registrationDisabled }),
            },
            {
                url: 'http://localhost/api/machines-server-token',
                method: 'GET',
                status: 200,
                response: () => ({ token: 'randomMachineToken' }),
            },
            {
                url: 'http://localhost/api/machines-server-token',
                method: 'PUT',
                status: 200,
                response: () => ({ token: 'regeneratedRandomMachineToken' }),
            },
        ],
    },
}

export const ListMachines: Story = {
    parameters: {
        mockData: [
            {
                url: 'http://localhost/api/machines?start=:start&limit=:limit',
                method: 'GET',
                status: 200,
                response: () => mockedAllRespData,
            },
            {
                url: 'http://localhost/api/machines?start=:start&limit=:limit&text=:text',
                method: 'GET',
                status: 200,
                response: (req) => mockedFilterByText(mockedAllRespData, req, 'address'),
            },
            {
                url: 'http://localhost/api/machines?start=:start&limit=:limit&text=:text&authorized=:authorized',
                method: 'GET',
                status: 200,
                response: (req) => {
                    if (req.searchParams?.authorized == 'true') {
                        return mockedFilterByText(mockedAuthorizedRespData, req, 'address')
                    }
                    return mockedFilterByText(mockedUnauthorizedRespData, req, 'address')
                },
            },
            {
                url: 'http://localhost/api/machines?start=:start&limit=:limit&authorized=:authorized',
                method: 'GET',
                status: 200,
                response: (req) => {
                    if (req.searchParams?.authorized == 'true') {
                        return mockedAuthorizedRespData
                    }
                    return mockedUnauthorizedRespData
                },
            },
            {
                url: 'http://localhost/api/machines/unauthorized/count',
                method: 'GET',
                status: 200,
                response: () => mockedUnauthorizedMachines.length,
            },
            {
                url: 'http://localhost/api/settings',
                method: 'GET',
                status: 200,
                response: () => ({ enableMachineRegistration: !meta.args.registrationDisabled }),
            },
            {
                url: 'http://localhost/api/machines-server-token',
                method: 'GET',
                status: 200,
                response: () => ({ token: 'randomMachineToken' }),
            },
            {
                url: 'http://localhost/api/machines-server-token',
                method: 'PUT',
                status: 200,
                response: () => ({ token: 'regeneratedRandomMachineToken' }),
            },
        ],
    },
}

export const TestAllMachinesShown: Story = {
    globals: {
        role: 'super-admin',
    },
    parameters: ListMachines.parameters,
    play: async ({ canvasElement }) => {
        // Arrange
        const canvas = within(canvasElement)
        const clearFiltersBtn = await canvas.findByRole('button', { name: 'Clear' })

        // Act
        await userEvent.click(clearFiltersBtn)

        // Assert
        // Check table content
        const allMachinesCount = mockedAuthorizedMachines.length + mockedUnauthorizedMachines.length
        await expect(await canvas.findAllByRole('row')).toHaveLength(allMachinesCount + 1) // All rows in tbody + one row in the thead.
        await expect(await canvas.findAllByRole('cell', { hidden: true })).toHaveLength(15 * allMachinesCount) // One row in the tbody has specific number of cells (15).
        await expect(canvas.getByText(mockedUnauthorizedMachines[0].address)).toBeInTheDocument()
        await expect(canvas.getByText(mockedUnauthorizedMachines[1].address)).toBeInTheDocument()
        await expect(canvas.getByText(mockedAuthorizedMachines[0].address)).toBeInTheDocument()
        await expect(canvas.getByText(mockedAuthorizedMachines[1].address)).toBeInTheDocument()

        // Check filtering panel content
        await expect(canvas.getByLabelText('Authorized')).toHaveProperty('checked', false) // Checkbox in the filtering panel.
        await expect(clearFiltersBtn).toBeDisabled()

        // Check bulk authorize button state
        const bulkAuthorizeBtn = await canvas.findByRole('button', { name: 'Authorize selected' })
        await expect(bulkAuthorizeBtn).toBeInTheDocument()
        await expect(bulkAuthorizeBtn).toBeDisabled()

        // Check table checkboxes behavior
        const checkboxes = await within(canvas.getByRole('table')).findAllByRole('checkbox')
        await expect(checkboxes).toHaveLength(allMachinesCount + 1)
        const disabledCheckboxes = checkboxes.filter((ch) => ch.hasAttribute('disabled'))
        await expect(disabledCheckboxes.length).toBe(mockedAuthorizedMachines.length)
        await expect(canvas.queryAllByRole('checkbox', { checked: true })).toHaveLength(0)
        // Click on Select All checkbox
        await userEvent.click(checkboxes[0])
        await expect(canvas.queryAllByRole('checkbox', { checked: true })).toHaveLength(
            mockedUnauthorizedMachines.length + 1
        )
        await expect(bulkAuthorizeBtn).toBeEnabled()
        // Click on Select All checkbox - selection should be cleared
        await userEvent.click(checkboxes[0])
        await expect(bulkAuthorizeBtn).toBeDisabled()
        await expect(canvas.queryAllByRole('checkbox', { checked: true })).toHaveLength(0)
        // Click on the checkboxes in the table rows one by one
        const enabledCheckboxes = checkboxes.filter((ch, idx) => idx !== 0 && !ch.hasAttribute('disabled'))
        for (let i = 0; i < enabledCheckboxes.length; i++) {
            await userEvent.click(enabledCheckboxes[i])
            await expect(bulkAuthorizeBtn).toBeEnabled()
            if (i < enabledCheckboxes.length - 1) {
                await expect(checkboxes[0]).not.toBeChecked()
            }
        }

        await expect(checkboxes[0]).toBeChecked()
    },
}

export const TestUnauthorizedShown: Story = {
    globals: {
        role: 'super-admin',
    },
    parameters: ListMachines.parameters,
    play: async ({ canvas }) => {
        // Arrange
        const selectButtonGroup = await canvas.findByRole('group') // PrimeNG p-selectButton has role=group

        // Act
        await userEvent.click(await within(selectButtonGroup).findByText('Unauthorized'))

        // Assert
        // Check table content
        await expect(canvas.getAllByRole('row')).toHaveLength(mockedUnauthorizedMachines.length + 1) // All rows in tbody + one row in the thead.
        await expect(canvas.getAllByRole('cell')).toHaveLength(5 * mockedUnauthorizedMachines.length) // One row in the tbody has specific number of cells.
        await expect(canvas.getByText(mockedUnauthorizedMachines[0].address)).toBeInTheDocument()
        await expect(canvas.getByText(mockedUnauthorizedMachines[1].address)).toBeInTheDocument()
        await expect(canvas.queryByText(mockedAuthorizedMachines[0].address)).toBeNull()
        await expect(canvas.queryByText(mockedAuthorizedMachines[1].address)).toBeNull()

        // Check filtering panel content
        await expect(canvas.getByLabelText('Authorized')).toHaveProperty('checked', false)
        const clearFiltersBtn = await canvas.findByRole('button', { name: 'Clear' })
        await expect(clearFiltersBtn).toBeEnabled()

        // Check bulk authorize button state
        const bulkAuthorizeBtn = await canvas.findByRole('button', { name: 'Authorize selected' })
        await expect(bulkAuthorizeBtn).toBeInTheDocument()
        await expect(bulkAuthorizeBtn).toBeDisabled()
        const checkboxes = await canvas.findAllByRole('checkbox')
        await userEvent.click(checkboxes[checkboxes.length - 1])
        await expect(bulkAuthorizeBtn).toBeEnabled()

        // Check menu items
        const menuButtons = await canvas.findAllByLabelText('Show machine menu')
        await expect(menuButtons.length).toBe(mockedUnauthorizedMachines.length)
        await userEvent.click(menuButtons[0])
        await canvas.findByRole('menu')
        await expect(await canvas.findAllByRole('menuitem')).toHaveLength(2)
        // PrimeNG menuitem role is a <LI> element, so we determine its disabled/enabled state by aria-disabled attribute.
        await expect(await canvas.findByRole('menuitem', { name: 'Authorize' })).not.toHaveAttribute(
            'aria-disabled',
            'true'
        )
        await expect(await canvas.findByRole('menuitem', { name: 'Remove' })).not.toHaveAttribute(
            'aria-disabled',
            'true'
        )
    },
}

export const TestAuthorizedShown: Story = {
    globals: {
        role: 'super-admin',
    },
    parameters: ListMachines.parameters,
    play: async ({ canvasElement }) => {
        // Arrange
        const canvas = within(canvasElement)
        const selectButtonGroup = await canvas.findByRole('group') // PrimeNG p-selectButton has role=group
        const authorizedButton = await within(selectButtonGroup).findByText('Authorized')

        // Act
        await userEvent.click(authorizedButton)

        // Assert
        // Check table content
        await expect(canvas.getAllByRole('row')).toHaveLength(mockedAuthorizedMachines.length + 1) // All rows in tbody + one row in the thead.
        await expect(canvas.getAllByRole('cell', { hidden: true })).toHaveLength(13 * mockedAuthorizedMachines.length) // One row in the tbody has specific number of cells (13).
        await expect(canvas.queryByText(mockedUnauthorizedMachines[0].address)).toBeNull()
        await expect(canvas.queryByText(mockedUnauthorizedMachines[1].address)).toBeNull()
        await expect(canvas.getByText(mockedAuthorizedMachines[0].address)).toBeInTheDocument()
        await expect(canvas.getByText(mockedAuthorizedMachines[1].address)).toBeInTheDocument()

        // Check filtering panel content
        await expect(canvas.getByLabelText('Authorized')).toHaveProperty('checked', true) // Checkbox in the filtering panel.
        const clearFiltersBtn = await canvas.findByRole('button', { name: 'Clear' })
        await expect(clearFiltersBtn).toBeEnabled()

        // Check there is no bulk authorize button
        await expect(canvas.queryByRole('button', { name: 'Authorize selected' })).toBeNull()

        // Check menu items
        const menuButtons = await canvas.findAllByLabelText('Show machine menu')
        await expect(menuButtons.length).toBe(mockedAuthorizedMachines.length)
        await userEvent.click(menuButtons[0])
        await canvas.findByRole('menu')
        await expect(await canvas.findAllByRole('menuitem')).toHaveLength(3)
        // PrimeNG menuitem role is a <LI> element, so we determine its disabled/enabled state by aria-disabled attribute.
        await expect(
            await canvas.findByRole('menuitem', { name: 'Refresh machine state information' })
        ).not.toHaveAttribute('aria-disabled', 'true')
        await expect(await canvas.findByRole('menuitem', { name: 'Dump troubleshooting data' })).not.toHaveAttribute(
            'aria-disabled',
            'true'
        )
        await expect(await canvas.findByRole('menuitem', { name: 'Remove' })).not.toHaveAttribute(
            'aria-disabled',
            'true'
        )
    },
}

export const TestTableFiltering: Story = {
    globals: {
        role: 'super-admin',
    },
    parameters: ListMachines.parameters,
    play: async ({ canvasElement }) => {
        // Arrange
        const canvas = within(canvasElement)
        const clearFiltersBtn = await canvas.findByRole('button', { name: 'Clear' })
        const authorizedCheckbox = await canvas.findByRole('checkbox', { name: 'Authorized' })
        const searchBox = await canvas.findByRole('textbox')

        // Act
        await userEvent.click(clearFiltersBtn)
        await userEvent.click(authorizedCheckbox) // At first, check filtering authorized machines.

        // Assert
        await expect(canvas.getAllByRole('row')).toHaveLength(mockedAuthorizedMachines.length + 1) // All rows in tbody + one row in the thead.
        await expect(canvas.getAllByRole('cell', { hidden: true })).toHaveLength(13 * mockedAuthorizedMachines.length) // One row in the tbody has specific number of cells (13).
        await expect(canvas.queryByText(mockedUnauthorizedMachines[0].address)).toBeNull()
        await expect(canvas.queryByText(mockedUnauthorizedMachines[1].address)).toBeNull()
        await expect(canvas.getByText(mockedAuthorizedMachines[0].address)).toBeInTheDocument()
        await expect(canvas.getByText(mockedAuthorizedMachines[1].address)).toBeInTheDocument()

        // Apply text filter.
        await userEvent.type(searchBox, 'ha')
        // Three authorized kea-ha machines are expected.
        await waitFor(() => expect(canvas.getAllByRole('row')).toHaveLength(4)) // All rows in tbody (3) + one row in the thead.
        await expect(canvas.getAllByRole('cell', { hidden: true })).toHaveLength(13 * 3) // One row in the tbody has specific number of cells (13).

        // Clear text filter.
        await userEvent.clear(searchBox)
        await waitFor(() => expect(canvas.getAllByRole('row')).toHaveLength(mockedAuthorizedMachines.length + 1)) // All rows in tbody + one row in the thead.

        // Filter unauthorized machines.
        await userEvent.click(authorizedCheckbox)
        await expect(canvas.getAllByRole('row')).toHaveLength(mockedUnauthorizedMachines.length + 1) // All rows in tbody + one row in the thead.
        await expect(canvas.getAllByRole('cell', { hidden: true })).toHaveLength(5 * mockedUnauthorizedMachines.length) // One row in the tbody has specific number of cells (5).
        await expect(canvas.getByText(mockedUnauthorizedMachines[0].address)).toBeInTheDocument()
        await expect(canvas.getByText(mockedUnauthorizedMachines[1].address)).toBeInTheDocument()
        await expect(canvas.queryByText(mockedAuthorizedMachines[0].address)).toBeNull()
        await expect(canvas.queryByText(mockedAuthorizedMachines[1].address)).toBeNull()

        // Apply text filter.
        await userEvent.type(searchBox, 'kea')
        // One unauthorized kea machine is expected.
        await waitFor(() => expect(canvas.getAllByRole('row')).toHaveLength(2)) // All rows in tbody + one row in the thead.
        await expect(canvas.getAllByRole('cell', { hidden: true })).toHaveLength(5) // One row in the tbody has specific number of cells (5).

        // Show authorized + unauthorized machines.
        await userEvent.click(authorizedCheckbox)
        // Six kea machines (authorized + unauthorized) are expected.
        await expect(canvas.getAllByRole('row')).toHaveLength(7) // All rows in tbody + one row in the thead.
        await expect(canvas.getAllByRole('cell', { hidden: true })).toHaveLength(15 * 6) // One row in the tbody has specific number of cells (15).
    },
}
