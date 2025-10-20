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
import { ManagedAccessDirective } from '../managed-access.directive'
import { PopoverModule } from 'primeng/popover'
import { SelectButtonModule } from 'primeng/selectbutton'
import { MenuModule } from 'primeng/menu'
import { InputTextModule } from 'primeng/inputtext'
import { BadgeModule } from 'primeng/badge'
import { TagModule } from 'primeng/tag'
import { FormsModule } from '@angular/forms'
import { AuthService } from '../auth.service'
import { toastDecorator } from '../utils-stories'
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
import { userEvent, within, expect } from '@storybook/test'

const meta: Meta<MachinesPageComponent> = {
    title: 'App/MachinesPage',
    component: MachinesPageComponent,
    subcomponents: MachinesTableComponent,
    decorators: [
        applicationConfig({
            providers: [
                provideHttpClient(withInterceptorsFromDi()),
                MessageService,
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
                { provide: AuthService, useValue: { hasPrivilege: () => true } },
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
                ManagedAccessDirective,
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
]
const mockedAuthorizedMachines = [
    {
        address: 'agent-pdns',
        agentPort: 8891,
        agentToken: 'random-agent-pdns-token',
        agentVersion: '2.3.0',
        apps: [
            {
                accessPoints: [{ address: '127.0.0.1', port: 8085, type: 'control' }],
                details: {
                    daemons: null,
                    pdnsDaemon: {
                        active: true,
                        autoprimariesUrl: '/api/v1/servers/localhost/autoprimaries{/autoprimary}',
                        configUrl: '/api/v1/servers/localhost/config{/config_setting}',
                        id: 1,
                        monitored: true,
                        name: 'pdns',
                        uptime: 19716,
                        url: '/api/v1/servers/localhost',
                        version: '4.7.3',
                        zonesUrl: '/api/v1/servers/localhost/zones{/zone}',
                    },
                },
                id: 1,
                machine: { id: 1 },
                name: 'pdns@agent-pdns',
                type: 'pdns',
                version: '4.7.3',
            },
        ],
        authorized: true,
        cpus: 12,
        cpusLoad: '0.89 0.84 0.86',
        hostID: 'agent-pdns-host-id',
        hostname: 'agent-pdns',
        id: 1,
        kernelArch: 'aarch64',
        kernelVersion: '6.10.14-linuxkit',
        lastVisitedAt: '2025-10-14T20:38:29.173Z',
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
        address: 'agent-bind9',
        agentPort: 8883,
        agentToken: 'random-agent-bind9',
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
                        autoZoneCount: 200,
                        id: 2,
                        monitored: true,
                        name: 'named',
                        reloadedAt: '2025-10-14T15:09:54.000Z',
                        uptime: 19715,
                        version: 'BIND 9.20.13 (Stable Release) <id:1f79fb9>',
                        views: [
                            { name: 'guest', queryHits: 0, queryMisses: 0 },
                            { name: 'trusted', queryHits: 0, queryMisses: 0 },
                        ],
                        zoneCount: 3,
                    },
                },
                id: 2,
                machine: { id: 2 },
                name: 'bind9@agent-bind9',
                type: 'bind9',
                version: 'BIND 9.20.13 (Stable Release) <id:1f79fb9>',
            },
        ],
        authorized: true,
        cpus: 12,
        cpusLoad: '0.89 0.84 0.86',
        error: 'Cannot get state of machine',
        hostID: 'agent-bind9-host-id',
        hostname: 'agent-bind9',
        id: 2,
        kernelArch: 'x86_64',
        kernelVersion: '6.10.14-linuxkit',
        lastVisitedAt: '2025-10-14T20:38:28.982Z',
        memory: 7,
        os: 'linux',
        platform: 'alpine',
        platformFamily: 'alpine',
        platformVersion: '3.22.1',
        usedMemory: 28,
        virtualizationRole: 'guest',
        virtualizationSystem: 'docker',
    },
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
    total: 9,
}
const mockedAuthorizedRespData = {
    items: mockedAuthorizedMachines,
    total: 8,
}
const mockedUnauthorizedRespData = {
    items: mockedUnauthorizedMachines,
    total: 1,
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
                response: () => ({ enableMachineRegistration: true }),
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

export const ListMixedAuthorizedAndNonAuthorized: Story = {
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
                response: () => mockedAllRespData,
            },
            {
                url: 'http://localhost/api/machines?start=:start&limit=:limit&text=:text&authorized=:authorized',
                method: 'GET',
                status: 200,
                response: (req) => {
                    if (req.searchParams?.authorized == 'true') {
                        return mockedAuthorizedRespData
                    } else {
                        return mockedUnauthorizedRespData
                    }
                },
            },
            {
                url: 'http://localhost/api/machines?start=:start&limit=:limit&authorized=:authorized',
                method: 'GET',
                status: 200,
                response: (req) => {
                    if (req.searchParams?.authorized == 'true') {
                        return mockedAuthorizedRespData
                    } else {
                        return mockedUnauthorizedRespData
                    }
                },
            },
            {
                url: 'http://localhost/api/machines/unauthorized/count',
                method: 'GET',
                status: 200,
                response: () => 1,
            },
            {
                url: 'http://localhost/api/settings',
                method: 'GET',
                status: 200,
                response: () => ({ enableMachineRegistration: true }),
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

export const UnauthorizedShown: Story = {
    parameters: ListMixedAuthorizedAndNonAuthorized.parameters,
    play: async ({ canvasElement }) => {
        // Arrange
        const canvas = within(canvasElement)
        const selectButtonGroup = await canvas.findByRole('group') // PrimeNG p-selectButton has role=group
        const clearFiltersBtn = await canvas.findByRole('button', { name: `Clear` })

        // Act
        await userEvent.click(await within(selectButtonGroup).findByText('Unauthorized'))

        // Assert
        await expect(canvas.getAllByRole('row')).toHaveLength(2) // One row in the thead, and only one row in the tbody.
        await expect(canvas.getAllByRole('cell')).toHaveLength(5) // One row in the tbody has specific number of cells.
        await expect(canvas.getByLabelText('Authorized')).toHaveProperty('checked', false)
        await expect(clearFiltersBtn).toBeEnabled()
    },
}

export const AuthorizedShown: Story = {
    parameters: ListMixedAuthorizedAndNonAuthorized.parameters,
    play: async ({ canvasElement }) => {
        // Arrange
        const canvas = within(canvasElement)
        const selectButtonGroup = await canvas.findByRole('group') // PrimeNG p-selectButton has role=group
        const authorizedButton = await within(selectButtonGroup).findByText('Authorized')
        const clearFiltersBtn = await canvas.findByRole('button', { name: `Clear` })

        // Act
        await userEvent.click(authorizedButton)

        // Assert
        await expect(canvas.getAllByRole('row')).toHaveLength(9) // One row in the thead, and eight rows in the tbody.
        await expect(canvas.getAllByRole('cell')).toHaveLength(13 * 8) // One row in the tbody has specific number of cells (13).
        await expect(canvas.getByLabelText('Authorized')).toHaveProperty('checked', true)
        await expect(clearFiltersBtn).toBeEnabled()
    },
}

export const AllMachinesShown: Story = {
    parameters: ListMixedAuthorizedAndNonAuthorized.parameters,
    play: async (context) => {
        // Arrange
        const canvas = within(context.canvasElement)
        const clearFiltersBtn = await canvas.findByRole('button', { name: `Clear` })
        await UnauthorizedShown.play(context)

        // Act
        await userEvent.click(clearFiltersBtn)

        // Assert
        await expect(await canvas.findAllByRole('row')).toHaveLength(10) // One row in the thead, and nine rows in the tbody.
        await expect(await canvas.findAllByRole('cell')).toHaveLength(15 * 9) // One row in the tbody has specific number of cells (15).
        await expect(canvas.getByLabelText('Authorized')).toHaveProperty('checked', false)
        await expect(clearFiltersBtn).toBeDisabled()
    },
}
