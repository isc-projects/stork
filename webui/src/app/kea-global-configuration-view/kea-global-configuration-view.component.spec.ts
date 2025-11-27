import { ComponentFixture, TestBed } from '@angular/core/testing'

import { KeaGlobalConfigurationViewComponent } from './kea-global-configuration-view.component'
import { provideNoopAnimations } from '@angular/platform-browser/animations'
import { By } from '@angular/platform-browser'
import { provideHttpClient, withInterceptorsFromDi } from '@angular/common/http'
import { provideHttpClientTesting } from '@angular/common/http/testing'
import { MessageService } from 'primeng/api'
import { AuthService } from '../auth.service'

describe('KeaGlobalConfigurationViewComponent', () => {
    let component: KeaGlobalConfigurationViewComponent
    let fixture: ComponentFixture<KeaGlobalConfigurationViewComponent>
    let authService: AuthService

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            providers: [
                provideHttpClient(withInterceptorsFromDi()),
                provideHttpClientTesting(),
                provideNoopAnimations(),
                MessageService,
            ],
        }).compileComponents()

        fixture = TestBed.createComponent(KeaGlobalConfigurationViewComponent)
        component = fixture.componentInstance
        authService = fixture.debugElement.injector.get(AuthService)
        spyOn(authService, 'superAdmin').and.returnValue(true)
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should display configuration parameters view', () => {
        const fieldset = fixture.debugElement.query(By.css("[legend='Global DHCP Parameters']"))
        expect(fieldset).toBeTruthy()
        expect(fieldset.nativeElement.innerText).toContain('No parameters configured')
    })

    it('should display an edit button', () => {
        const button = fixture.debugElement.query(By.css('[label=Edit]'))
        expect(button).toBeTruthy()
    })

    it('should emit an event upon edit', () => {
        spyOn(component.editBegin, 'emit')
        component.onEditBegin()
        expect(component.editBegin.emit).toHaveBeenCalled()
    })

    it('should display the DHCP options view', () => {
        const fieldset = fixture.debugElement.query(By.css("[legend='Global DHCP Options']"))
        expect(fieldset).toBeTruthy()
        expect(fieldset.nativeElement.innerText).toContain('No options configured')
    })
})
