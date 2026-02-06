import { By } from '@angular/platform-browser'
import { ComponentFixture, TestBed, fakeAsync, tick, waitForAsync, flush } from '@angular/core/testing'

import { provideRouter } from '@angular/router'
import { ServicesService } from '../backend'
import { ConfirmationService, MessageService } from 'primeng/api'
import { provideHttpClientTesting } from '@angular/common/http/testing'
import { BreadcrumbsComponent } from '../breadcrumbs/breadcrumbs.component'
import { provideNoopAnimations } from '@angular/platform-browser/animations'
import { ConfirmDialog } from 'primeng/confirmdialog'
import { of, throwError } from 'rxjs'
import { provideHttpClient, withInterceptorsFromDi } from '@angular/common/http'
import { DaemonsPageComponent } from './daemons-page.component'

describe('DaemonsPageComponent', () => {
    let component: DaemonsPageComponent
    let fixture: ComponentFixture<DaemonsPageComponent>
    let api: ServicesService
    let msgSrv: MessageService

    beforeEach(waitForAsync(() => {
        TestBed.configureTestingModule({
            providers: [
                ConfirmationService,
                MessageService,
                provideNoopAnimations(),
                provideHttpClient(withInterceptorsFromDi()),
                provideHttpClientTesting(),
                provideRouter([{ path: 'daemons', component: DaemonsPageComponent }]),
            ],
        }).compileComponents()
    }))

    beforeEach(() => {
        fixture = TestBed.createComponent(DaemonsPageComponent)
        component = fixture.componentInstance
        api = fixture.debugElement.injector.get(ServicesService)
        msgSrv = fixture.debugElement.injector.get(MessageService)
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should have breadcrumbs', () => {
        const breadcrumbsElement = fixture.debugElement.query(By.directive(BreadcrumbsComponent))
        expect(breadcrumbsElement).not.toBeNull()
        const breadcrumbsComponent = breadcrumbsElement.componentInstance as BreadcrumbsComponent
        expect(breadcrumbsComponent).not.toBeNull()
        expect(breadcrumbsComponent.items).toHaveSize(2)
        expect(breadcrumbsComponent.items[0].label).toEqual('Services')
        expect(breadcrumbsComponent.items[1].label).toEqual('Daemons')
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
        confirmDialog.onAccept()
        tick()
        fixture.detectChanges()

        expect(api.deleteKeaDaemonConfigHashes).toHaveBeenCalled()
        expect(msgSrv.add).toHaveBeenCalled()
        flush()
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
        confirmDialog.onAccept()
        tick()
        fixture.detectChanges()

        expect(api.deleteKeaDaemonConfigHashes).toHaveBeenCalled()
        expect(msgSrv.add).toHaveBeenCalled()
        flush()
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
        confirmDialog.onReject()
        tick()
        fixture.detectChanges()

        expect(api.deleteKeaDaemonConfigHashes).not.toHaveBeenCalled()
        expect(msgSrv.add).not.toHaveBeenCalled()
        flush()
    }))

    it('should have enabled or disabled button in filtering toolbar according to privileges', () => {
        expect(component.toolbarButtons.length).toBeGreaterThan(0)
        // at first, it should be disabled
        expect(component.toolbarButtons[0].disabled).toBeTrue()
        // it should react on privilege change
        component.canResyncConfig.set(true)
        fixture.detectChanges()
        expect(component.toolbarButtons[0].disabled).toBeFalse()
    })
})
