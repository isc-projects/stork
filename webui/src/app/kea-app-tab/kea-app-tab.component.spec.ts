import { fakeAsync, tick, ComponentFixture, TestBed, waitForAsync } from '@angular/core/testing'

import { KeaAppTabComponent } from './kea-app-tab.component'
import { RouterModule, Router, ActivatedRoute } from '@angular/router'
import { RouterTestingModule } from '@angular/router/testing'
import { HaStatusComponent } from '../ha-status/ha-status.component'
import { TableModule } from 'primeng/table'
import { TabViewModule } from 'primeng/tabview'
import { LocaltimePipe } from '../localtime.pipe'
import { HttpClientTestingModule } from '@angular/common/http/testing'
import { PanelModule } from 'primeng/panel'
import { TooltipModule } from 'primeng/tooltip'
import { MessageModule } from 'primeng/message'
import { MessageService } from 'primeng/api'
import { MockLocationStrategy } from '@angular/common/testing'
import { By } from '@angular/platform-browser'
import { BehaviorSubject, of, throwError } from 'rxjs'

import { DHCPService, ServicesService, UsersService } from '../backend'
import { ServerDataService } from '../server-data.service'
import { RenameAppDialogComponent } from '../rename-app-dialog/rename-app-dialog.component'
import { InputSwitchModule } from 'primeng/inputswitch'
import { FieldsetModule } from 'primeng/fieldset'
import { EventsPanelComponent } from '../events-panel/events-panel.component'
import { DialogModule } from 'primeng/dialog'
import { NoopAnimationsModule } from '@angular/platform-browser/animations'
import { PaginatorModule } from 'primeng/paginator'
import { FormsModule } from '@angular/forms'

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

    beforeEach(
        waitForAsync(() => {
            TestBed.configureTestingModule({
                providers: [UsersService, DHCPService, ServicesService, MessageService, MockLocationStrategy],
                imports: [
                    RouterModule,
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
                ],
                declarations: [
                    KeaAppTabComponent,
                    HaStatusComponent,
                    LocaltimePipe,
                    RenameAppDialogComponent,
                    EventsPanelComponent,
                ],
            }).compileComponents()
        })
    )

    beforeEach(() => {
        fixture = TestBed.createComponent(KeaAppTabComponent)
        component = fixture.componentInstance
        servicesApi = fixture.debugElement.injector.get(ServicesService)
        serverData = fixture.debugElement.injector.get(ServerDataService)
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
        const fakeMachinesAddresses = new Set()
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
        const fakeMachinesAddresses = new Set()
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
        expect(component.databaseNameFromType('other')).toBe('Unknown')
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
})
