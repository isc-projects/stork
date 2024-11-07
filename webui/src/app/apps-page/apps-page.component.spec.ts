import { By } from '@angular/platform-browser'
import { ComponentFixture, TestBed, fakeAsync, tick, waitForAsync } from '@angular/core/testing'

import { AppsPageComponent } from './apps-page.component'
import { TabMenuModule } from 'primeng/tabmenu'
import { MenuModule } from 'primeng/menu'
import { FormsModule } from '@angular/forms'
import { TableModule } from 'primeng/table'
import { Bind9AppTabComponent } from '../bind9-app-tab/bind9-app-tab.component'
import { KeaAppTabComponent } from '../kea-app-tab/kea-app-tab.component'
import { TooltipModule } from 'primeng/tooltip'
import { TabViewModule } from 'primeng/tabview'
import { HaStatusComponent } from '../ha-status/ha-status.component'
import { PanelModule } from 'primeng/panel'
import { MessageModule } from 'primeng/message'
import { RouterModule } from '@angular/router'
import { ServicesService } from '../backend'
import { ConfirmationService, MessageService } from 'primeng/api'
import { RouterTestingModule } from '@angular/router/testing'
import { HttpClientTestingModule } from '@angular/common/http/testing'
import { BreadcrumbsComponent } from '../breadcrumbs/breadcrumbs.component'
import { BreadcrumbModule } from 'primeng/breadcrumb'
import { HelpTipComponent } from '../help-tip/help-tip.component'
import { OverlayPanelModule } from 'primeng/overlaypanel'
import { NoopAnimationsModule } from '@angular/platform-browser/animations'
import { LocaltimePipe } from '../pipes/localtime.pipe'
import { ProgressSpinnerModule } from 'primeng/progressspinner'
import { ConfirmDialog, ConfirmDialogModule } from 'primeng/confirmdialog'
import { of, throwError } from 'rxjs'

class App {
    id: number
    name: string
}

describe('AppsPageComponent', () => {
    let component: AppsPageComponent
    let fixture: ComponentFixture<AppsPageComponent>
    let api: ServicesService
    let msgSrv: MessageService

    beforeEach(waitForAsync(() => {
        TestBed.configureTestingModule({
            providers: [ConfirmationService, ServicesService, MessageService],
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
                ProgressSpinnerModule,
                ConfirmDialogModule,
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
        api = fixture.debugElement.injector.get(ServicesService)
        msgSrv = fixture.debugElement.injector.get(MessageService)
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

    it('should request synchronization configurations from Kea', fakeAsync(() => {
        component.onSyncKeaConfigs()
        fixture.detectChanges()

        const dialog = fixture.debugElement.query(By.directive(ConfirmDialog))
        expect(dialog).not.toBeNull()
        expect(dialog.nativeElement.innerText).toContain('This operation instructs')

        const success: any = {}
        spyOn(api, 'deleteKeaDaemonConfigHashes').and.returnValue(of(success))
        spyOn(msgSrv, 'add')
        const confirmDialog = dialog.componentInstance as ConfirmDialog
        expect(confirmDialog).not.toBeNull()
        confirmDialog.accept()
        tick()
        fixture.detectChanges()

        expect(api.deleteKeaDaemonConfigHashes).toHaveBeenCalled()
        expect(msgSrv.add).toHaveBeenCalled()
    }))

    it('should report an error while requesting synchronization configurations from Kea', fakeAsync(() => {
        component.onSyncKeaConfigs()
        fixture.detectChanges()

        const dialog = fixture.debugElement.query(By.directive(ConfirmDialog))
        expect(dialog).not.toBeNull()
        expect(dialog.nativeElement.innerText).toContain('This operation instructs')

        // Simulate an error so we can also test that the error message is shown.
        spyOn(api, 'deleteKeaDaemonConfigHashes').and.returnValue(throwError({ status: 404 }))
        spyOn(msgSrv, 'add')
        const confirmDialog = dialog.componentInstance as ConfirmDialog
        expect(confirmDialog).not.toBeNull()
        confirmDialog.accept()
        tick()
        fixture.detectChanges()

        expect(api.deleteKeaDaemonConfigHashes).toHaveBeenCalled()
        expect(msgSrv.add).toHaveBeenCalled()
    }))

    it('should cancel synchronizing configurations from Kea', fakeAsync(() => {
        component.onSyncKeaConfigs()
        fixture.detectChanges()

        const dialog = fixture.debugElement.query(By.directive(ConfirmDialog))
        expect(dialog).not.toBeNull()
        expect(dialog.nativeElement.innerText).toContain('This operation instructs')

        spyOn(api, 'deleteKeaDaemonConfigHashes')
        spyOn(msgSrv, 'add')
        const confirmDialog = dialog.componentInstance as ConfirmDialog
        expect(confirmDialog).not.toBeNull()
        confirmDialog.reject()
        tick()
        fixture.detectChanges()

        expect(api.deleteKeaDaemonConfigHashes).not.toHaveBeenCalled()
        expect(msgSrv.add).not.toHaveBeenCalled()
    }))
})
