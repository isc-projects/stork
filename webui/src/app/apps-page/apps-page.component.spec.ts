import { By } from '@angular/platform-browser'
import { ComponentFixture, TestBed, waitForAsync } from '@angular/core/testing'

import { AppsPageComponent } from './apps-page.component'
import { TabMenuModule } from 'primeng/tabmenu'
import { MenuModule } from 'primeng/menu'
import { FormsModule } from '@angular/forms'
import { TableModule } from 'primeng/table'
import { Bind9AppTabComponent } from '../bind9-app-tab/bind9-app-tab.component'
import { KeaAppTabComponent } from '../kea-app-tab/kea-app-tab.component'
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
import { BreadcrumbsComponent } from '../breadcrumbs/breadcrumbs.component'
import { BreadcrumbModule } from 'primeng/breadcrumb'
import { HelpTipComponent } from '../help-tip/help-tip.component'
import { OverlayPanelModule } from 'primeng/overlaypanel'
import { NoopAnimationsModule } from '@angular/platform-browser/animations'
import { LocaltimePipe } from '../pipes/localtime.pipe'

class App {
    id: number
    name: string
}

describe('AppsPageComponent', () => {
    let component: AppsPageComponent
    let fixture: ComponentFixture<AppsPageComponent>

    beforeEach(waitForAsync(() => {
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
                BreadcrumbModule,
                OverlayPanelModule,
                NoopAnimationsModule,
            ],
            declarations: [
                AppsPageComponent,
                Bind9AppTabComponent,
                KeaAppTabComponent,
                LocaltimePipe,
                HaStatusComponent,
                BreadcrumbsComponent,
                HelpTipComponent,
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

    it('should change app tab label after rename', () => {
        const app = new App()
        app.id = 1
        app.name = 'kea@@machine1'

        component.appType = 'kea'

        // Open a tab presenting our test app.
        component.addAppTab(app)
        component.switchToTab(1)
        expect(component.tabs.length).toEqual(2)
        expect(component.tabs[1].hasOwnProperty('label')).toBeTrue()
        expect(component.tabs[1].label).toBe('kea@@machine1')

        // Generate notification that the app was renamed.
        const event = 'kea@@machine2'
        component.onRenameApp(event)

        // The notification should cause the app tab label to
        // be changed to the new name.
        expect(component.tabs.length).toEqual(2)
        expect(component.tabs[1].hasOwnProperty('label')).toBeTrue()
        expect(component.tabs[1].label).toBe('kea@@machine2')
    })

    it('should have breadcrumbs', () => {
        const breadcrumbsElement = fixture.debugElement.query(By.directive(BreadcrumbsComponent))
        expect(breadcrumbsElement).not.toBeNull()
        const breadcrumbsComponent = breadcrumbsElement.componentInstance as BreadcrumbsComponent
        expect(breadcrumbsComponent).not.toBeNull()
        expect(breadcrumbsComponent.items).toHaveSize(2)
        expect(breadcrumbsComponent.items[0].label).toEqual('Services')
        expect(breadcrumbsComponent.items[1].label).toEqual('Kea Apps')
    })
})
