import { ComponentFixture, TestBed, waitForAsync } from '@angular/core/testing'
import { KeaAppTabComponent } from './kea-app-tab.component'
import { ActivatedRoute } from '@angular/router'
import { RouterTestingModule } from '@angular/router/testing'
import { HaStatusComponent } from '../ha-status/ha-status.component'
import { TableModule } from 'primeng/table'
import { TabViewModule } from 'primeng/tabview'
import { LocaltimePipe } from '../pipes/localtime.pipe'
import { HttpClientTestingModule } from '@angular/common/http/testing'
import { PanelModule } from 'primeng/panel'
import { TooltipModule } from 'primeng/tooltip'
import { MessageModule } from 'primeng/message'
import { MessageService } from 'primeng/api'
import { MockLocationStrategy } from '@angular/common/testing'
import { By } from '@angular/platform-browser'
import { BehaviorSubject, of, throwError } from 'rxjs'

import { AppsVersions, DHCPService, ServicesService, UsersService } from '../backend'
import { ServerDataService } from '../server-data.service'
import { RenameAppDialogComponent } from '../rename-app-dialog/rename-app-dialog.component'
import { InputSwitchModule } from 'primeng/inputswitch'
import { FieldsetModule } from 'primeng/fieldset'
import { EventsPanelComponent } from '../events-panel/events-panel.component'
import { DialogModule } from 'primeng/dialog'
import { NoopAnimationsModule } from '@angular/platform-browser/animations'
import { PaginatorModule } from 'primeng/paginator'
import { FormsModule } from '@angular/forms'
import { OverlayPanelModule } from 'primeng/overlaypanel'
import { ConfigReviewPanelComponent } from '../config-review-panel/config-review-panel.component'
import { HelpTipComponent } from '../help-tip/help-tip.component'
import { AppOverviewComponent } from '../app-overview/app-overview.component'
import { ButtonModule } from 'primeng/button'
import { DataViewModule } from 'primeng/dataview'
import { ToggleButtonModule } from 'primeng/togglebutton'
import { ProgressSpinnerModule } from 'primeng/progressspinner'
import { PlaceholderPipe } from '../pipes/placeholder.pipe'
import { ServerSentEventsService, ServerSentEventsTestingService } from '../server-sent-events.service'
import { DividerModule } from 'primeng/divider'
import { TagModule } from 'primeng/tag'
import { EventTextComponent } from '../event-text/event-text.component'
import { VersionStatusComponent } from '../version-status/version-status.component'
import { Severity, VersionService } from '../version.service'

class Details {
    daemons: any = [
        {
            id: 1,
            pid: 1234,
            name: 'dhcp4',
            active: false,
            monitored: true,
            version: '1.9.4',
            extendedVersion: '1.9.4-extended',
            uptime: 100,
            reloadedAt: 10000,
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
        },
        {
            id: 2,
            pid: 2345,
            name: 'dhcp6',
            active: false,
            monitored: true,
            version: '1.9.5',
            extendedVersion: '1.9.5-extended',
            uptime: 100,
            reloadedAt: 10000,
            hooks: [],
            files: [],
            backends: [],
        },
    ]
}

class Machine {
    id = 1
}

class App {
    id = 1
    name = ''
    machine = new Machine()
    details = new Details()
}

class AppTab {
    app = new App()
}

describe('KeaAppTabComponent', () => {
    let component: KeaAppTabComponent
    let fixture: ComponentFixture<KeaAppTabComponent>
    let servicesApi: ServicesService
    let serverData: ServerDataService
    let route: ActivatedRoute
    let versionServiceStub: Partial<VersionService>

    beforeEach(waitForAsync(() => {
        versionServiceStub = {
            sanitizeSemver: () => '1.9.4',
            getCurrentData: () => of({} as AppsVersions),
            getSoftwareVersionFeedback: () => ({ severity: Severity.success, messages: ['test feedback'] }),
        }

        TestBed.configureTestingModule({
            providers: [
                UsersService,
                DHCPService,
                ServicesService,
                MessageService,
                MockLocationStrategy,
                { provide: ServerSentEventsService, useClass: ServerSentEventsTestingService },
                { provide: VersionService, useValue: versionServiceStub },
            ],
            imports: [
                RouterTestingModule,
                TableModule,
                TabViewModule,
                PanelModule,
                TooltipModule,
                MessageModule,
                HttpClientTestingModule,
                FormsModule,
                InputSwitchModule,
                FieldsetModule,
                DialogModule,
                NoopAnimationsModule,
                PaginatorModule,
                OverlayPanelModule,
                ButtonModule,
                FormsModule,
                ToggleButtonModule,
                DataViewModule,
                ProgressSpinnerModule,
                DividerModule,
                TagModule,
            ],
            declarations: [
                KeaAppTabComponent,
                HaStatusComponent,
                LocaltimePipe,
                PlaceholderPipe,
                RenameAppDialogComponent,
                EventsPanelComponent,
                ConfigReviewPanelComponent,
                HelpTipComponent,
                AppOverviewComponent,
                EventTextComponent,
                VersionStatusComponent,
            ],
        }).compileComponents()
    }))

    beforeEach(() => {
        fixture = TestBed.createComponent(KeaAppTabComponent)
        component = fixture.componentInstance
        servicesApi = fixture.debugElement.injector.get(ServicesService)
        serverData = fixture.debugElement.injector.get(ServerDataService)
        route = fixture.debugElement.injector.get(ActivatedRoute)
        fixture.debugElement.injector.get(VersionService)
        const appTab = new AppTab()
        component.refreshedAppTab = new BehaviorSubject(appTab)
        component.appTab = appTab
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should display rename dialog', () => {
        const fakeAppsNames = new Map()
        spyOn(serverData, 'getAppsNames').and.returnValue(of(fakeAppsNames))
        const fakeMachinesAddresses = new Set<string>()
        spyOn(serverData, 'getMachinesAddresses').and.returnValue(of(fakeMachinesAddresses))
        expect(component.appRenameDialogVisible).toBeFalse()
        component.showRenameAppDialog()
        expect(serverData.getAppsNames).toHaveBeenCalled()
        expect(serverData.getMachinesAddresses).toHaveBeenCalled()
        // The dialog should be visible after fetching apps names and machines
        // addresses successfully.
        expect(component.appRenameDialogVisible).toBeTrue()
    })

    it('should not display rename dialog when fetching machines fails', () => {
        const fakeAppsNames = new Map()
        spyOn(serverData, 'getAppsNames').and.returnValue(of(fakeAppsNames))
        // Simulate an error while getting machines addresses.
        spyOn(serverData, 'getMachinesAddresses').and.returnValue(throwError({ status: 404 }))
        expect(component.appRenameDialogVisible).toBeFalse()
        component.showRenameAppDialog()
        expect(serverData.getAppsNames).toHaveBeenCalled()
        expect(serverData.getMachinesAddresses).toHaveBeenCalled()
        // The dialog should not be visible because there was an error.
        expect(component.appRenameDialogVisible).toBeFalse()
    })

    it('should not display rename dialog when fetching apps fails', () => {
        // Simulate an error while getting apps names.
        spyOn(serverData, 'getAppsNames').and.returnValue(throwError({ status: 404 }))
        const fakeMachinesAddresses = new Set<string>()
        spyOn(serverData, 'getMachinesAddresses').and.returnValue(of(fakeMachinesAddresses))
        expect(component.appRenameDialogVisible).toBeFalse()
        component.showRenameAppDialog()
        expect(serverData.getAppsNames).toHaveBeenCalled()
        expect(serverData.getMachinesAddresses).toHaveBeenCalled()
        // The dialog should not be visible because there was an error.
        expect(component.appRenameDialogVisible).toBeFalse()
    })

    it('should send app rename request', () => {
        // Prepare fake success response to renameApp call.
        const fakeResponse: any = { data: {} }
        spyOn(servicesApi, 'renameApp').and.returnValue(of(fakeResponse))
        // Simulate submitting the app rename request.
        component.handleRenameDialogSubmitted('keax@machine3')
        // Make sure that the request to rename the app was submitted.
        expect(servicesApi.renameApp).toHaveBeenCalled()
        // As a result, the app name in the tab should have been updated.
        expect(component.appTab.app.name).toBe('keax@machine3')
    })

    it('should hide app rename dialog', () => {
        // Show the dialog box.
        component.appRenameDialogVisible = true
        fixture.detectChanges()
        spyOn(servicesApi, 'renameApp')
        // Cancel the dialog box.
        component.handleRenameDialogHidden()
        // Ensure that the dialog box is no longer visible.
        expect(component.appRenameDialogVisible).toBeFalse()
        // A request to rename the app should not be sent.
        expect(servicesApi.renameApp).not.toHaveBeenCalled()
    })

    it('should return filename from file', () => {
        let file: any = { filename: '/tmp/kea-leases4.csv', filetype: 'Lease file' }
        expect(component.filenameFromFile(file)).toBe('/tmp/kea-leases4.csv')

        file = { filetype: 'Lease file' }
        expect(component.filenameFromFile(file)).toBe('none (lease persistence disabled)')

        file = { filename: '', filetype: 'Lease file' }
        expect(component.filenameFromFile(file)).toBe('none (lease persistence disabled)')

        file = { filename: '', filetype: 'Forensic log' }
        expect(component.filenameFromFile(file)).toBe('none')
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
        component.appTab.app.details.daemons[0].files = []
        component.appTab.app.details.daemons[0].backends = []
        fixture.detectChanges()
        const dataStorage = fixture.debugElement.query(By.css('#data-storage-div'))
        expect(dataStorage).toBeNull()
    })

    it('should select tab according to the daemon parameter', () => {
        spyOn(route.snapshot.queryParamMap, 'get').and.returnValue('dhcp6')
        component.ngOnInit()
        expect(component.activeTabIndex).toBe(1)
        fixture.detectChanges()
        expect(fixture.debugElement.nativeElement.innerText).toContain('1.9.5')
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

    it('should display an empty placeholder when no app is loaded', () => {
        // Check legend.
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
        component.appTab.app.details.daemons[0].hooks = [
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
        expect(component.isNeverFetchedDaemon({})).toBeTrue()
        expect(component.isNeverFetchedDaemon({ reloadedAt: undefined })).toBeTrue()
        expect(component.isNeverFetchedDaemon({ reloadedAt: null })).toBeTrue()
        // Any value.
        for (const timestamp of [
            '1970-01-01T12:00:00Z',
            '0001-01-01 00:00:00.000 UTC',
            '0001-01-01T00:00:00.000Z',
            'foobar',
        ]) {
            expect(component.isNeverFetchedDaemon({ reloadedAt: timestamp })).toBeFalse()
        }
    })

    it('should properly recognize the DHCP daemons', () => {
        expect(component.isDhcpDaemon({ name: 'dhcp4' })).toBeTrue()
        expect(component.isDhcpDaemon({ name: 'dhcp6' })).toBeTrue()
        expect(component.isDhcpDaemon({ name: 'd2' })).toBeFalse()
        expect(component.isDhcpDaemon({ name: 'netconf' })).toBeFalse()
        expect(component.isDhcpDaemon({ name: 'foobar' })).toBeFalse()
    })

    it('should display version status component', () => {
        // One VersionStatus for Kea.
        let versionStatus = fixture.debugElement.queryAll(By.directive(VersionStatusComponent))
        expect(versionStatus).toBeTruthy()
        expect(versionStatus.length).toEqual(1)
        // Stubbed success icon for kea 1.9.4 is expected.
        expect(versionStatus[0].properties.outerHTML).toContain('1.9.4')
        expect(versionStatus[0].properties.outerHTML).toContain('kea')
        expect(versionStatus[0].properties.outerHTML).toContain('text-green-500')
        expect(versionStatus[0].properties.outerHTML).toContain('test feedback')
    })
})
