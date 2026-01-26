import { ComponentFixture, TestBed, waitForAsync } from '@angular/core/testing'
import { provideRouter } from '@angular/router'
import { provideHttpClientTesting } from '@angular/common/http/testing'
import { ConfirmationService, MessageService } from 'primeng/api'
import { MockLocationStrategy } from '@angular/common/testing'
import { By } from '@angular/platform-browser'
import { of } from 'rxjs'

import { AppsVersions, KeaDaemon } from '../backend'
import { provideNoopAnimations } from '@angular/platform-browser/animations'
import { ServerSentEventsService, ServerSentEventsTestingService } from '../server-sent-events.service'
import { VersionStatusComponent } from '../version-status/version-status.component'
import { Severity, VersionService } from '../version.service'
import { provideHttpClient, withInterceptorsFromDi } from '@angular/common/http'
import { KeaDaemonComponent } from './kea-daemon.component'

describe('KeaDaemonComponent', () => {
    let component: KeaDaemonComponent
    let fixture: ComponentFixture<KeaDaemonComponent>
    let versionServiceStub: Partial<VersionService>

    beforeEach(waitForAsync(() => {
        versionServiceStub = {
            sanitizeSemver: () => '1.9.4',
            getCurrentData: () => of({} as AppsVersions),
            getSoftwareVersionFeedback: () => ({ severity: Severity.success, messages: ['test feedback'] }),
        }

        TestBed.configureTestingModule({
            providers: [
                MessageService,
                MockLocationStrategy,
                ConfirmationService,
                { provide: ServerSentEventsService, useClass: ServerSentEventsTestingService },
                { provide: VersionService, useValue: versionServiceStub },
                provideHttpClient(withInterceptorsFromDi()),
                provideHttpClientTesting(),
                provideNoopAnimations(),
                provideRouter([]),
            ],
        }).compileComponents()
    }))

    beforeEach(() => {
        const dhcp4Daemon: KeaDaemon = {
            id: 1,
            pid: 1234,
            name: 'dhcp4',
            active: false,
            monitored: true,
            version: '1.9.4',
            extendedVersion: '1.9.4-extended',
            uptime: 100,
            reloadedAt: '2025-01-01T12:00:00Z',
            hooks: [],
            files: [
                {
                    filetype: 'Lease file',
                    filename: '/tmp/kea-leases4.csv',
                },
            ],
            backends: [
                {
                    backendType: 'mysql',
                    database: 'kea',
                    host: 'localhost',
                    dataTypes: ['Leases', 'Host Reservations'],
                },
            ],
            machineId: 1,
        }

        fixture = TestBed.createComponent(KeaDaemonComponent)
        component = fixture.componentInstance
        fixture.debugElement.injector.get(VersionService)
        fixture.componentRef.setInput('daemon', dhcp4Daemon)
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should return filename from file', () => {
        let file: any = { filename: '/tmp/kea-leases4.csv', filetype: 'Lease file' }
        expect(component.filenameFromFile(file)).toBe('/tmp/kea-leases4.csv')

        file = { filetype: 'Lease file' }
        expect(component.filenameFromFile(file)).toBe('none (persistence disabled)')

        file = { filename: '', filetype: 'Lease file' }
        expect(component.filenameFromFile(file)).toBe('none (persistence disabled)')

        file = { filetype: 'Lease file', persist: true }
        expect(component.filenameFromFile(file)).toBe('default (persistence enabled)')

        file = { filename: '', filetype: 'Forensic log' }
        expect(component.filenameFromFile(file)).toBe('none (persistence disabled)')

        file = { filename: '/tmp/kea-forensic.log', filetype: 'Forensic log' }
        expect(component.filenameFromFile(file)).toBe('/tmp/kea-forensic.log')

        file = { filetype: 'Forensic log', persist: true }
        expect(component.filenameFromFile(file)).toBe('default (persistence enabled)')
    })

    it('should return database name from type', () => {
        expect(component.databaseNameFromType('memfile')).toBe('Memfile')
        expect(component.databaseNameFromType('mysql')).toBe('MySQL')
        expect(component.databaseNameFromType('postgresql')).toBe('PostgreSQL')
        expect(component.databaseNameFromType('cql')).toBe('Cassandra')
        expect(component.databaseNameFromType('other' as any)).toBe('Unknown')
    })

    it('should display storage information', () => {
        const dataStorageFilesFieldset = fixture.debugElement.query(By.css('#data-storage-files-fieldset'))
        const dataStorageFilesElement = dataStorageFilesFieldset.nativeElement
        expect(dataStorageFilesElement.innerText).toContain('Lease file')
        expect(dataStorageFilesElement.innerText).toContain('/tmp/kea-leases4.csv')

        const dataStorageBackendsFieldset = fixture.debugElement.query(By.css('#data-storage-backends-fieldset'))
        const dataStorageBackendsElement = dataStorageBackendsFieldset.nativeElement
        expect(dataStorageBackendsElement.innerText).toContain('MySQL (kea@localhost) with')
        expect(dataStorageBackendsElement.innerText).toContain('Leases')
        expect(dataStorageBackendsElement.innerText).toContain('Host Reservations')
    })

    it('should not display data storage when files and backends are blank', () => {
        component.daemon().files = []
        component.daemon().backends = []
        fixture.detectChanges()
        const dataStorage = fixture.debugElement.query(By.css('#data-storage-div'))
        expect(dataStorage).toBeNull()
    })

    it('should know how to take the base name out of a path', () => {
        expect(component.basename('')).toBe('')
        expect(component.basename('base')).toBe('base')
        expect(component.basename('/base')).toBe('base')
        expect(component.basename('/path/to/base')).toBe('base')
    })

    it('should know how to convert hook libraries to Kea documentation anchors', () => {
        expect(component.docAnchorFromHookLibrary('libdhcp_user_chk.so', '')).toBeNull()
        expect(component.docAnchorFromHookLibrary('', '2.3.8')).toBeNull()
        expect(component.docAnchorFromHookLibrary('', '2.4.0')).toBeNull()
        expect(component.docAnchorFromHookLibrary('', '2.5.4-git')).toBeNull()
        expect(component.docAnchorFromHookLibrary('libdhcp_user_chk.so', '2.3.7')).toBe(
            'kea-2.3.7/arm/hooks.html#user-chk-user-check'
        )
        expect(component.docAnchorFromHookLibrary('libdhcp_user_chk.so', '2.4.0')).toBe(
            'kea-2.4.0/arm/hooks.html#std-ischooklib-libdhcp_user_chk.so'
        )
        expect(component.docAnchorFromHookLibrary('libdhcp_user_chk.so', '2.5.4-git')).toBe(
            'latest/arm/hooks.html#std-ischooklib-libdhcp_user_chk.so'
        )
        expect(component.docAnchorFromHookLibrary('libdhcp_fake.so', '2.3.8')).toBeNull()
        expect(component.docAnchorFromHookLibrary('libdns_fake.so', '2.4.0')).toBe(
            'kea-2.4.0/arm/hooks.html#std-ischooklib-libdns_fake.so'
        )
        expect(component.docAnchorFromHookLibrary('libca_fake.so', '2.5.4-git')).toBe(
            'latest/arm/hooks.html#std-ischooklib-libca_fake.so'
        )
        expect(component.docAnchorFromHookLibrary('kea-dhcp4', '2.3.7')).toBeNull()
        expect(component.docAnchorFromHookLibrary('libdhcp_mysql.so', '2.7.4')).toBe(
            'kea-2.7.4/arm/hooks.html#std-ischooklib-libdhcp_mysql.so'
        )
        expect(component.docAnchorFromHookLibrary('libdhcp_pgsql.so', '2.7.4')).toBe(
            'kea-2.7.4/arm/hooks.html#std-ischooklib-libdhcp_pgsql.so'
        )
        expect(component.docAnchorFromHookLibrary('libdhcp_rbac.so', '2.7.4')).toBe(
            'kea-2.7.4/arm/hooks.html#std-ischooklib-libdhcp_rbac.so'
        )
        expect(component.docAnchorFromHookLibrary('libdhcp_mysql.so', '2.3.8')).toBeNull()
    })

    it('should display an empty placeholder when no hooks are present', () => {
        const hooksFieldset = fixture.debugElement.query(By.css('#hooks-fieldset'))
        expect(hooksFieldset).toBeTruthy()
        expect(hooksFieldset.attributes['legend']).toEqual('Hooks')

        // Check content.
        const div = hooksFieldset.query(By.css('div'))
        expect(div).toBeTruthy()
        const divElement = div.nativeElement
        expect(divElement).toBeTruthy()
        expect(divElement.innerText).toEqual('No hooks')
    })

    it('should display hook libraries', () => {
        component.daemon().hooks = [
            '/libdhcp_cb_cmds.so',
            '/lib/libdhcp_custom.so',
            '/usr/lib/libdhcp_fake.so',
            '/usr/local/lib/libdhcp_lease_cmds.so',
        ]
        fixture.detectChanges()

        // Check legend.
        const hooksFieldset = fixture.debugElement.query(By.css('#hooks-fieldset'))
        expect(hooksFieldset).toBeTruthy()
        expect(hooksFieldset.attributes['legend']).toEqual('Hooks')

        // Check content.
        const div = hooksFieldset.query(By.css('div'))
        expect(div).toBeTruthy()
        const fieldsetContent = div.query(By.css('div.p-fieldset-content ol'))
        expect(fieldsetContent).toBeTruthy()
        const childNodes = fieldsetContent.nativeElement.childNodes
        expect(childNodes).toBeTruthy()
        expect(childNodes.length).toBeGreaterThanOrEqual(4)
        expect(childNodes[0]).toBeTruthy()
        expect((childNodes[0] as HTMLElement).tagName).toBe('LI')
        expect(childNodes[0].innerText.replace(/\n/g, '')).toBe('libdhcp_cb_cmds.so[doc]')
        expect(childNodes[1]).toBeTruthy()
        expect((childNodes[1] as HTMLElement).tagName).toBe('LI')
        expect(childNodes[1].innerText.replace(/\n/g, '')).toBe('libdhcp_custom.so')
        expect(childNodes[2]).toBeTruthy()
        expect((childNodes[2] as HTMLElement).tagName).toBe('LI')
        expect(childNodes[2].innerText.replace(/\n/g, '')).toBe('libdhcp_fake.so')
        expect(childNodes[3]).toBeTruthy()
        expect((childNodes[3] as HTMLElement).tagName).toBe('LI')
        expect(childNodes[3].innerText.replace(/\n/g, '')).toBe('libdhcp_lease_cmds.so[doc]')

        // There may be other children. Probably comments. Check that they are not divs which
        // ensures that no other hook libraries are displayed.
        for (let i = 4; i < childNodes.length; i++) {
            expect(childNodes[i]).toBeTruthy()
            expect((childNodes[i] as HTMLElement).tagName).not.toBe('DIV')
        }
    })

    it('should properly recognize that daemon was never monitored', () => {
        fixture.componentRef.setInput('daemon', {})
        expect(component.isNeverFetchedDaemon()).toBeTrue()
        fixture.componentRef.setInput('daemon', { reloadedAt: undefined })
        expect(component.isNeverFetchedDaemon()).toBeTrue()
        fixture.componentRef.setInput('daemon', { reloadedAt: null })
        expect(component.isNeverFetchedDaemon()).toBeTrue()
        // Any value.
        for (const timestamp of [
            '1970-01-01T12:00:00Z',
            '0001-01-01 00:00:00.000 UTC',
            '0001-01-01T00:00:00.000Z',
            'foobar',
        ]) {
            fixture.componentRef.setInput('daemon', { reloadedAt: timestamp })
            expect(component.isNeverFetchedDaemon()).toBeFalse()
        }
    })

    it('should properly recognize the DHCP daemons', () => {
        fixture.componentRef.setInput('daemon', { name: 'dhcp4' })
        expect(component.isDhcpDaemon()).toBeTrue()
        fixture.componentRef.setInput('daemon', { name: 'dhcp6' })
        expect(component.isDhcpDaemon()).toBeTrue()
        fixture.componentRef.setInput('daemon', { name: 'd2' })
        expect(component.isDhcpDaemon()).toBeFalse()
        fixture.componentRef.setInput('daemon', { name: 'netconf' })
        expect(component.isDhcpDaemon()).toBeFalse()
        fixture.componentRef.setInput('daemon', { name: 'foobar' })
        expect(component.isDhcpDaemon()).toBeFalse()
    })

    it('should display version status component', () => {
        // One VersionStatus for Kea.
        let versionStatus = fixture.debugElement.queryAll(By.directive(VersionStatusComponent))
        expect(versionStatus).toBeTruthy()
        expect(versionStatus.length).toEqual(1)
        // Stubbed success icon for kea 1.9.4 is expected.
        expect((versionStatus[0].nativeElement as HTMLElement).innerHTML).toContain('1.9.4')
        expect((versionStatus[0].nativeElement as HTMLElement).innerHTML).not.toContain('DHCPv4')
        expect((versionStatus[0].nativeElement as HTMLElement).innerHTML).toContain('text-green-500')
        expect((versionStatus[0].nativeElement as HTMLElement).innerHTML).toContain('test feedback')
    })
})
