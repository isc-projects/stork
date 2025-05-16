import { ManagedAccessDirective } from './managed-access.directive'
import { Component } from '@angular/core'
import { Button } from 'primeng/button'
import { ComponentFixture, TestBed } from '@angular/core/testing'
import { AuthService } from './auth.service'
import { provideHttpClient, withInterceptorsFromDi } from '@angular/common/http'
import { provideHttpClientTesting } from '@angular/common/http/testing'
import { MessageService } from 'primeng/api'

@Component({
    standalone: true,
    template: `<p-button label="Create" appManagedAccess="subnet" accessType="create" />
        <p-button label="Read" appManagedAccess="subnet" accessType="read" />
        <p-button label="Update" appManagedAccess="subnet" accessType="update" />
        <p-button label="Delete" appManagedAccess="subnet" accessType="delete" /> `,
    imports: [ManagedAccessDirective, Button],
})
class TestHostComponent {}

describe('ManagedAccessDirective', () => {
    let hostComponent: TestHostComponent
    let fixture: ComponentFixture<TestHostComponent>
    let authService: AuthService
    let hasPrivilegeSpy: jasmine.Spy
    let firstComponent: any
    let secondComponent: any
    let thirdComponent: any
    let fourthComponent: any

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            declarations: [],
            imports: [TestHostComponent],
            providers: [provideHttpClient(withInterceptorsFromDi()), provideHttpClientTesting(), MessageService],
        }).compileComponents()

        fixture = TestBed.createComponent(TestHostComponent)
        hostComponent = fixture.componentInstance
        authService = fixture.debugElement.injector.get(AuthService)
        hasPrivilegeSpy = spyOn(authService, 'hasPrivilege')
        hasPrivilegeSpy.withArgs('subnet', 'update').and.returnValue(false)
        hasPrivilegeSpy.withArgs('subnet', 'read').and.returnValue(true)
        hasPrivilegeSpy.withArgs('subnet', 'create').and.returnValue(false)
        hasPrivilegeSpy.withArgs('subnet', 'delete').and.returnValue(false)

        expect(fixture.debugElement.children).toBeTruthy()
        expect(fixture.debugElement.children.length).toBeGreaterThanOrEqual(4)
        firstComponent = fixture.debugElement.children[0].componentInstance as Button
        secondComponent = fixture.debugElement.children[1].componentInstance as Button
        thirdComponent = fixture.debugElement.children[2].componentInstance as Button
        fourthComponent = fixture.debugElement.children[3].componentInstance as Button

        fixture.detectChanges()
    })

    it('should create', () => {
        expect(hostComponent).toBeTruthy()
        expect(firstComponent).toBeTruthy()
        expect(secondComponent).toBeTruthy()
        expect(thirdComponent).toBeTruthy()
        expect(fourthComponent).toBeTruthy()
    })

    // it('should set access type', () => {
    //     expect(firstComponent.accessType).toBe('write')
    //     expect(secondComponent.accessType).toBe('read')
    //     expect(thirdComponent.accessType).toBe('write')
    // })
    //
    // it('should display content when no stork template was used', () => {
    //     expect(firstComponent.templates.length).toBe(0)
    //     expect(firstComponent.hasAccess).toBeTrue()
    //     const headingDe = fixture.debugElement.query(By.css('h1'))
    //     expect(headingDe).toBeTruthy()
    //     expect(headingDe.nativeElement.textContent).toContain('Edit machine authorization')
    // })
    //
    // it('should display hasAccess stork template content', () => {
    //     expect(secondComponent.templates.length).toBe(1)
    //     expect(secondComponent.hasAccess).toBeTrue()
    //     const headingDe = fixture.debugElement.query(By.css('h2'))
    //     expect(headingDe).toBeTruthy()
    //     expect(headingDe.nativeElement.textContent).toContain('Full access')
    // })
    //
    // it('should display no content when no access', () => {
    //     hasPrivilegeSpy.withArgs('machine-authorization', 'write').and.returnValue(false)
    //     firstComponent.ngOnInit()
    //     fixture.detectChanges()
    //     const headingDe = fixture.debugElement.query(By.css('h1'))
    //     expect(headingDe).toBeFalsy()
    // })
    //
    // it('should display default limited content when no access', () => {
    //     hasPrivilegeSpy.withArgs('app-access-point-key', 'read').and.returnValue(false)
    //     secondComponent.ngOnInit()
    //     fixture.detectChanges()
    //     const headingDe = fixture.debugElement.query(By.css('h2'))
    //     expect(headingDe).toBeFalsy()
    //     expect(fixture.debugElement.nativeElement.innerHTML).toContain("You don't have read privileges")
    //     // Default template for limited access has pi-ban icon included.
    //     expect(fixture.debugElement.query(By.css('.pi-ban'))).toBeTruthy()
    // })
    //
    // it('should emit if user has access', () => {
    //     expect(hostComponent.secondHasAccess).toBeTrue()
    //     hasPrivilegeSpy.withArgs('app-access-point-key', 'read').and.returnValue(false)
    //     secondComponent.ngOnInit()
    //     fixture.detectChanges()
    //     expect(hostComponent.secondHasAccess).toBeFalse()
    // })
    //
    // it('should display stork template content when has access and template name does not match hasAccess name', () => {
    //     // Third component uses appTemplate="foobar" stork template name.
    //     expect(thirdComponent.templates.length).toBe(2)
    //     expect(thirdComponent.hasAccess).toBeTrue()
    //     const headingDe = fixture.debugElement.query(By.css('h3'))
    //     expect(headingDe).toBeTruthy()
    //     expect(headingDe.nativeElement.textContent).toContain('Write access')
    // })
    //
    // it('should display limited content when noAccess template was provided even if hideOnNoAccess is true', () => {
    //     hasPrivilegeSpy.withArgs('edit-host-reservation', 'write').and.returnValue(false)
    //     thirdComponent.ngOnInit()
    //     fixture.detectChanges()
    //     expect(thirdComponent.hideOnNoAccess).toBeTrue()
    //     const hasAccessHeadingDe = fixture.debugElement.query(By.css('h3'))
    //     expect(hasAccessHeadingDe).toBeFalsy()
    //     const headingDe = fixture.debugElement.query(By.css('h4'))
    //     expect(headingDe).toBeTruthy()
    //     expect(fixture.debugElement.nativeElement.innerHTML).toContain('Limited access')
    // })
    //
    // it('should display dedicated noAccess template', () => {
    //     hasPrivilegeSpy.withArgs('edit-host-reservation', 'write').and.returnValue(false)
    //     thirdComponent.hideOnNoAccess = false
    //     thirdComponent.ngOnInit()
    //     fixture.detectChanges()
    //     expect(thirdComponent.hideOnNoAccess).toBeFalse()
    //     const hasAccessHeadingDe = fixture.debugElement.query(By.css('h3'))
    //     expect(hasAccessHeadingDe).toBeFalsy()
    //     const headingDe = fixture.debugElement.query(By.css('h4'))
    //     expect(headingDe).toBeTruthy()
    //     expect(fixture.debugElement.nativeElement.innerHTML).toContain('Limited access')
    // })
})
