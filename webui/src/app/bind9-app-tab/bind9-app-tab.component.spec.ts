import { async, ComponentFixture, TestBed } from '@angular/core/testing'

import { Bind9AppTabComponent } from './bind9-app-tab.component'
import { RouterLink, Router, RouterModule, ActivatedRoute } from '@angular/router'
import { RouterTestingModule } from '@angular/router/testing'
import { TooltipModule } from 'primeng/tooltip'
import { TabViewModule } from 'primeng/tabview'
import { LocaltimePipe } from '../localtime.pipe'
import { MockLocationStrategy } from '@angular/common/testing'
import { of } from 'rxjs'

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
            imports: [TooltipModule, TabViewModule, RouterModule, RouterTestingModule],
            declarations: [Bind9AppTabComponent, LocaltimePipe],
            providers: [MockLocationStrategy],
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
})
