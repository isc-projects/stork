import { ComponentFixture, TestBed, fakeAsync, tick, waitForAsync } from '@angular/core/testing'

import { AppTabComponent } from './app-tab.component'
import { MessageService } from 'primeng/api'
import { MockLocationStrategy } from '@angular/common/testing'
import { provideHttpClientTesting } from '@angular/common/http/testing'
import { of, throwError } from 'rxjs'

import { AppsVersions, ServicesService } from '../backend'
import { ServerDataService } from '../server-data.service'
import { provideNoopAnimations } from '@angular/platform-browser/animations'
import { ServerSentEventsService, ServerSentEventsTestingService } from '../server-sent-events.service'
import { EventsPanelComponent } from '../events-panel/events-panel.component'
import { By } from '@angular/platform-browser'
import { Severity, VersionService } from '../version.service'
import { provideHttpClient, withInterceptorsFromDi } from '@angular/common/http'
import { provideRouter } from '@angular/router'
import { Bind9DaemonControlsComponent } from '../bind9-daemon-controls/bind9-daemon-controls.component'

class Daemon {
    name = 'named'
    version = '9.18.30'
    id = 7
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

describe('AppTabComponent', () => {
    let component: AppTabComponent
    let fixture: ComponentFixture<AppTabComponent>
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
                MessageService,
                MockLocationStrategy,
                { provide: ServerSentEventsService, useClass: ServerSentEventsTestingService },
                { provide: VersionService, useValue: versionServiceStub },
                provideNoopAnimations(),
                provideHttpClient(withInterceptorsFromDi()),
                provideHttpClientTesting(),
                provideRouter([]),
            ],
        }).compileComponents()
    }))

    beforeEach(() => {
        fixture = TestBed.createComponent(AppTabComponent)
        component = fixture.componentInstance
        servicesApi = fixture.debugElement.injector.get(ServicesService)
        serverData = fixture.debugElement.injector.get(ServerDataService)
        fixture.debugElement.injector.get(VersionService)
        const appTab = new AppTab()
        component.appTab = appTab
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should display rename dialog', fakeAsync(() => {
        const fakeAppsNames = new Map()
        spyOn(serverData, 'getAppsNames').and.returnValue(of(fakeAppsNames))
        const fakeMachinesAddresses = new Set<string>()
        spyOn(serverData, 'getMachinesAddresses').and.returnValue(of(fakeMachinesAddresses))
        expect(component.appRenameDialogVisible).toBeFalse()
        component.showRenameAppDialog()
        tick()
        expect(serverData.getAppsNames).toHaveBeenCalled()
        expect(serverData.getMachinesAddresses).toHaveBeenCalled()
        // The dialog should be visible after fetching apps names and machines
        // addresses successfully.
        expect(component.appRenameDialogVisible).toBeTrue()
    }))

    it('should not display rename dialog when fetching machines fails', fakeAsync(() => {
        const fakeAppsNames = new Map()
        spyOn(serverData, 'getAppsNames').and.returnValue(of(fakeAppsNames))
        // Simulate an error while getting machines addresses.
        spyOn(serverData, 'getMachinesAddresses').and.returnValue(throwError({ status: 404 }))
        expect(component.appRenameDialogVisible).toBeFalse()
        component.showRenameAppDialog()
        tick()
        expect(serverData.getAppsNames).toHaveBeenCalled()
        expect(serverData.getMachinesAddresses).toHaveBeenCalled()
        // The dialog should not be visible because there was an error.
        expect(component.appRenameDialogVisible).toBeFalse()
    }))

    it('should not display rename dialog when fetching apps fails', fakeAsync(() => {
        // Simulate an error while getting apps names.
        spyOn(serverData, 'getAppsNames').and.returnValue(throwError({ status: 404 }))
        const fakeMachinesAddresses = new Set<string>()
        spyOn(serverData, 'getMachinesAddresses').and.returnValue(of(fakeMachinesAddresses))
        expect(component.appRenameDialogVisible).toBeFalse()
        component.showRenameAppDialog()
        tick()
        expect(serverData.getAppsNames).toHaveBeenCalled()
        expect(serverData.getMachinesAddresses).toHaveBeenCalled()
        // The dialog should not be visible because there was an error.
        expect(component.appRenameDialogVisible).toBeFalse()
    }))

    it('should send app rename request', fakeAsync(() => {
        // Prepare fake success response to renameApp call.
        const fakeResponse: any = { data: {} }
        spyOn(servicesApi, 'renameApp').and.returnValue(of(fakeResponse))
        // Simulate submitting the app rename request.
        component.handleRenameDialogSubmitted('bindx@machine3')
        tick()
        // Make sure that the request to rename the app was submitted.
        expect(servicesApi.renameApp).toHaveBeenCalled()
        // As a result, the app name in the tab should have been updated.
        expect(component.appTab.app.name).toBe('bindx@machine3')
    }))

    it('should hide app rename dialog', fakeAsync(() => {
        // Show the dialog box.
        component.appRenameDialogVisible = true
        fixture.detectChanges()
        spyOn(servicesApi, 'renameApp')
        // Cancel the dialog box.
        component.handleRenameDialogHidden()
        tick()
        // Ensure that the dialog box is no longer visible.
        expect(component.appRenameDialogVisible).toBeFalse()
        // A request to rename the app should not be sent.
        expect(servicesApi.renameApp).not.toHaveBeenCalled()
    }))

    it('should include events', () => {
        const eventsPanel = fixture.debugElement.query(By.directive(EventsPanelComponent))
        expect(eventsPanel).toBeTruthy()
    })

    it('should include bind9 daemon controls', () => {
        const bind9DaemonControls = fixture.debugElement.query(By.directive(Bind9DaemonControlsComponent))
        expect(bind9DaemonControls).toBeTruthy()
    })
})
