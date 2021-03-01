import { async, ComponentFixture, TestBed } from '@angular/core/testing'

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
import { of, throwError } from 'rxjs'

import { DHCPService, ServicesService, UsersService } from '../backend'
import { ServerDataService } from '../server-data.service'

class Details {
    daemons: any = []
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

    beforeEach(async(() => {
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
            ],
            declarations: [KeaAppTabComponent, HaStatusComponent, LocaltimePipe],
        }).compileComponents()
    }))

    beforeEach(() => {
        fixture = TestBed.createComponent(KeaAppTabComponent)
        component = fixture.componentInstance
        servicesApi = fixture.debugElement.injector.get(ServicesService)
        serverData = fixture.debugElement.injector.get(ServerDataService)
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
})
