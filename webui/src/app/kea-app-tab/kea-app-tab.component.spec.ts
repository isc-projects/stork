import { async, ComponentFixture, TestBed } from '@angular/core/testing'

import { KeaAppTabComponent } from './kea-app-tab.component'
import { RouterModule, Router, ActivatedRoute } from '@angular/router'
import { RouterTestingModule } from '@angular/router/testing'
import { HaStatusComponent } from '../ha-status/ha-status.component'
import { TableModule } from 'primeng/table'
import { TabViewModule } from 'primeng/tabview'
import { LocaltimePipe } from '../localtime.pipe'
import { DHCPService, ServicesService } from '../backend'
import { HttpClientTestingModule } from '@angular/common/http/testing'
import { PanelModule } from 'primeng/panel'
import { TooltipModule } from 'primeng/tooltip'
import { MessageModule } from 'primeng/message'
import { MessageService } from 'primeng/api'
import { MockLocationStrategy } from '@angular/common/testing'
import { of } from 'rxjs'

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

    beforeEach(async(() => {
        TestBed.configureTestingModule({
            providers: [DHCPService, ServicesService, MessageService, MockLocationStrategy],
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
        const appTab = new AppTab()
        component.refreshedAppTab = of(appTab)
        component.appTab = appTab
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should send app rename request', () => {
        // Prepare fake success response to renameApp call.
        const fakeResponse: any = { data: {} }
        spyOn(component.servicesApi, 'renameApp').and.returnValue(of(fakeResponse))
        // Simulate submitting the app rename request.
        component.handleRenameDialogSubmitted('keax@machine3')
        // Make sure that the request to rename the app was submitted.
        expect(component.servicesApi.renameApp).toHaveBeenCalled()
        // As a result, the app name in the tab should have been updated.
        expect(component.appTab.app.name).toBe('keax@machine3')
    })
})
