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
            providers: [DHCPService, ServicesService, MockLocationStrategy],
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
})
