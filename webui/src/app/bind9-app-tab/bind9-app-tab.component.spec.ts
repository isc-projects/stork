import { async, ComponentFixture, TestBed } from '@angular/core/testing'

import { Bind9AppTabComponent } from './bind9-app-tab.component'
import { RouterLink, Router, RouterModule, ActivatedRoute } from '@angular/router'
import { RouterTestingModule } from '@angular/router/testing'
import { TooltipModule } from 'primeng/tooltip'
import { TabViewModule } from 'primeng/tabview'
import { MessageService } from 'primeng/api'
import { LocaltimePipe } from '../localtime.pipe'
import { MockLocationStrategy } from '@angular/common/testing'
import { HttpClientTestingModule } from '@angular/common/http/testing'
import { of } from 'rxjs'

import { ServicesService } from '../backend'

class Daemon {
    name = 'bind9'
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
    machine = new Machine()
    details = new Details()
}

class AppTab {
    app: App = new App()
}

describe('Bind9AppTabComponent', () => {
    let component: Bind9AppTabComponent
    let fixture: ComponentFixture<Bind9AppTabComponent>

    beforeEach(async(() => {
        TestBed.configureTestingModule({
            providers: [ServicesService, MessageService, MockLocationStrategy],
            imports: [HttpClientTestingModule, RouterModule, RouterTestingModule, TooltipModule, TabViewModule],
            declarations: [Bind9AppTabComponent, LocaltimePipe],
        }).compileComponents()
    }))

    beforeEach(() => {
        fixture = TestBed.createComponent(Bind9AppTabComponent)
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
        component.handleRenameDialogSubmitted('bindx@machine3')
        // Make sure that the request to rename the app was submitted.
        expect(component.servicesApi.renameApp).toHaveBeenCalled()
        // As a result, the app name in the tab should have been updated.
        expect(component.appTab.app.name).toBe('bindx@machine3')
    })
})
