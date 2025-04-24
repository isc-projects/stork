import { ComponentFixture, TestBed } from '@angular/core/testing'

import { ManagedAccessComponent } from './managed-access.component'
import { provideHttpClient, withInterceptorsFromDi } from '@angular/common/http'
import { provideHttpClientTesting } from '@angular/common/http/testing'
import { MessageService } from 'primeng/api'
import { Component, ViewChild } from '@angular/core'
import { StorkTemplateDirective } from '../stork-template.directive'
import { AuthService } from '../auth.service'
import { By } from '@angular/platform-browser'

describe('ManagedAccessComponent', () => {
    let hostComponent: TestHostComponent
    let fixture: ComponentFixture<TestHostComponent>
    let authService: AuthService

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            declarations: [],
            imports: [TestHostComponent, ManagedAccessComponent],
            providers: [provideHttpClient(withInterceptorsFromDi()), provideHttpClientTesting(), MessageService],
        }).compileComponents()

        fixture = TestBed.createComponent(TestHostComponent)
        hostComponent = fixture.componentInstance
        authService = fixture.debugElement.injector.get(AuthService)
        const hasPrivilegeSpy = spyOn(authService, 'hasPrivilege')
        hasPrivilegeSpy.withArgs('edit-machine-authorization', 'write').and.returnValue(true)
        hasPrivilegeSpy.withArgs('get-access-point-key', 'read').and.returnValue(true)

        fixture.detectChanges()
    })

    it('should create', () => {
        expect(hostComponent).toBeTruthy()
        expect(hostComponent.first).toBeTruthy()
        expect(hostComponent.second).toBeTruthy()
    })

    it('should display content when no stork template was used', () => {
        expect(hostComponent.first.templates.length).toBe(0)
        const headingDe = fixture.debugElement.query(By.css('h1'))
        expect(headingDe).toBeTruthy()
        expect(headingDe.nativeElement.textContent).toContain('Edit machine authorization')
    })
})

@Component({
    standalone: true,
    template: `
        <app-managed-access #first key="edit-machine-authorization"
            ><h1>Edit machine authorization.</h1></app-managed-access
        >
        <app-managed-access #second key="get-access-point-key" accessType="read" [hideOnNoAccess]="false">
            <ng-template appTemplate="hasAccess">Full access</ng-template>
        </app-managed-access>
    `,
    imports: [ManagedAccessComponent, StorkTemplateDirective],
})
class TestHostComponent {
    @ViewChild('first') first: ManagedAccessComponent
    @ViewChild('second') second: ManagedAccessComponent
}
