import { ComponentFixture, TestBed, fakeAsync, tick, waitForAsync } from '@angular/core/testing'

import { Bind9AppTabComponent } from './bind9-app-tab.component'
import { RouterTestingModule } from '@angular/router/testing'
import { TooltipModule } from 'primeng/tooltip'
import { TabViewModule } from 'primeng/tabview'
import { MessageService } from 'primeng/api'
import { LocaltimePipe } from '../pipes/localtime.pipe'
import { MockLocationStrategy } from '@angular/common/testing'
import { HttpClientTestingModule } from '@angular/common/http/testing'
import { of, throwError } from 'rxjs'

import { AppsVersions, Bind9DaemonView, ServicesService, UsersService } from '../backend'
import { ServerDataService } from '../server-data.service'
import { RenameAppDialogComponent } from '../rename-app-dialog/rename-app-dialog.component'
import { DialogModule } from 'primeng/dialog'
import { FormsModule } from '@angular/forms'
import { NoopAnimationsModule } from '@angular/platform-browser/animations'
import { AppOverviewComponent } from '../app-overview/app-overview.component'
import { PanelModule } from 'primeng/panel'
import { PlaceholderPipe } from '../pipes/placeholder.pipe'
import { ServerSentEventsService, ServerSentEventsTestingService } from '../server-sent-events.service'
import { EventsPanelComponent } from '../events-panel/events-panel.component'
import { By } from '@angular/platform-browser'
import { OverlayPanelModule } from 'primeng/overlaypanel'
import { DataViewModule } from 'primeng/dataview'
import { EventTextComponent } from '../event-text/event-text.component'
import { TableModule } from 'primeng/table'
import { VersionStatusComponent } from '../version-status/version-status.component'
import { Severity, VersionService } from '../version.service'

class Daemon {
    name = 'named'
    version = '9.18.30'
}

class Details {
    daemon: Daemon = new Daemon()
}

class Machine {
    id = 1
}

class App {
    id = 1
    name = ''
    type = 'bind9'
    machine = new Machine()
    details = new Details()
}

class AppTab {
    app: App = new App()
}

describe('Bind9AppTabComponent', () => {
    let component: Bind9AppTabComponent
    let fixture: ComponentFixture<Bind9AppTabComponent>
    let servicesApi: ServicesService
    let serverData: ServerDataService
    let versionServiceStub: Partial<VersionService>

    beforeEach(waitForAsync(() => {
        versionServiceStub = {
            sanitizeSemver: () => '9.18.30',
            getCurrentData: () => of({} as AppsVersions),
            getSoftwareVersionFeedback: () => ({ severity: Severity.success, messages: ['test feedback'] }),
        }

        TestBed.configureTestingModule({
            providers: [
                UsersService,
                ServicesService,
                MessageService,
                MockLocationStrategy,
                { provide: ServerSentEventsService, useClass: ServerSentEventsTestingService },
                { provide: VersionService, useValue: versionServiceStub },
            ],
            imports: [
                HttpClientTestingModule,
                FormsModule,
                RouterTestingModule,
                TooltipModule,
                TabViewModule,
                DialogModule,
                NoopAnimationsModule,
                PanelModule,
                OverlayPanelModule,
                DataViewModule,
                TableModule,
            ],
            declarations: [
                Bind9AppTabComponent,
                LocaltimePipe,
                PlaceholderPipe,
                RenameAppDialogComponent,
                AppOverviewComponent,
                EventsPanelComponent,
                EventTextComponent,
                VersionStatusComponent,
            ],
        }).compileComponents()
    }))

    beforeEach(() => {
        fixture = TestBed.createComponent(Bind9AppTabComponent)
        component = fixture.componentInstance
        servicesApi = fixture.debugElement.injector.get(ServicesService)
        serverData = fixture.debugElement.injector.get(ServerDataService)
        fixture.debugElement.injector.get(VersionService)
        const appTab = new AppTab()
        component.refreshedAppTab = of(appTab)
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
        component.handleRenameDialogSubmitted('bindx@machine3')
        // Make sure that the request to rename the app was submitted.
        expect(servicesApi.renameApp).toHaveBeenCalled()
        // As a result, the app name in the tab should have been updated.
        expect(component.appTab.app.name).toBe('bindx@machine3')
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

    it('should include events', fakeAsync(() => {
        component.ngOnInit()
        tick()
        fixture.detectChanges()

        const eventsPanel = fixture.debugElement.query(By.css('app-events-panel'))
        expect(eventsPanel).toBeTruthy()
    }))

    it('should display version status component', () => {
        // One VersionStatus BIND9.
        let versionStatus = fixture.debugElement.queryAll(By.directive(VersionStatusComponent))
        expect(versionStatus).toBeTruthy()
        expect(versionStatus.length).toEqual(1)
        // Stubbed success icon for BIND 9.18.30 is expected.
        expect(versionStatus[0].properties.outerHTML).toContain('9.18.30')
        expect(versionStatus[0].properties.outerHTML).toContain('bind9')
        expect(versionStatus[0].properties.outerHTML).toContain('text-green-500')
        expect(versionStatus[0].properties.outerHTML).toContain('test feedback')
    })

    it('should return 0 when queryHitRatio is undefined', () => {
        const view = {} as Bind9DaemonView
        expect(component.getQueryUtilization(view)).toBe(0)
    })

    it('should calculate correct utilization percentage', () => {
        const view = { queryHitRatio: 0.756 } as Bind9DaemonView
        expect(component.getQueryUtilization(view)).toBe(75)
    })
})
