import { async, ComponentFixture, TestBed } from '@angular/core/testing'

import { AppsPageComponent } from './apps-page.component'
import { TabMenuModule } from 'primeng/tabmenu'
import { MenuModule } from 'primeng/menu'
import { FormsModule } from '@angular/forms'
import { TableModule } from 'primeng/table'
import { Bind9AppTabComponent } from '../bind9-app-tab/bind9-app-tab.component'
import { KeaAppTabComponent } from '../kea-app-tab/kea-app-tab.component'
import { LocaltimePipe } from '../localtime.pipe'
import { TooltipModule } from 'primeng/tooltip'
import { TabPanel, TabViewModule } from 'primeng/tabview'
import { HaStatusComponent } from '../ha-status/ha-status.component'
import { PanelModule } from 'primeng/panel'
import { MessageModule } from 'primeng/message'
import { ActivatedRoute, Router, RouterModule, convertToParamMap } from '@angular/router'
import { ServicesService } from '../backend'
import { MessageService } from 'primeng/api'
import { RouterTestingModule } from '@angular/router/testing'
import { HttpClientTestingModule } from '@angular/common/http/testing'
import { of } from 'rxjs'

class App {
    id: number
    name: string
}

describe('AppsPageComponent', () => {
    let component: AppsPageComponent
    let fixture: ComponentFixture<AppsPageComponent>

    beforeEach(async(() => {
        TestBed.configureTestingModule({
            providers: [ServicesService, MessageService],
            imports: [
                HttpClientTestingModule,
                TabMenuModule,
                MenuModule,
                FormsModule,
                TableModule,
                TooltipModule,
                TabViewModule,
                PanelModule,
                MessageModule,
                RouterModule,
                RouterTestingModule.withRoutes([{ path: 'apps/:appType/all', component: AppsPageComponent }]),
            ],
            declarations: [
                AppsPageComponent,
                Bind9AppTabComponent,
                KeaAppTabComponent,
                LocaltimePipe,
                HaStatusComponent,
            ],
        }).compileComponents()
    }))

    beforeEach(() => {
        fixture = TestBed.createComponent(AppsPageComponent)
        component = fixture.componentInstance
        component.appType = 'bind9'
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should render good app tab title and link', () => {
        const app = new App()
        app.id = 1
        app.name = 'test-app'

        component.appType = 'bind9'

        component.addAppTab(app)
        expect(component.tabs.length).toEqual(2)
        expect(component.tabs[1].hasOwnProperty('label')).toBeTrue()
        expect(component.tabs[1].hasOwnProperty('routerLink')).toBeTrue()

        expect(component.tabs[1].label).toBe('test-app')
        expect(component.tabs[1].routerLink).toBe('/apps/bind9/1')
    })
})
