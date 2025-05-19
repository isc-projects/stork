import { ManagedAccessDirective } from './managed-access.directive'
import { Component } from '@angular/core'
import { Button } from 'primeng/button'
import { ComponentFixture, TestBed } from '@angular/core/testing'
import { AuthService } from './auth.service'
import { provideHttpClient, withInterceptorsFromDi } from '@angular/common/http'
import { provideHttpClientTesting } from '@angular/common/http/testing'
import { MessageService } from 'primeng/api'
import { By } from '@angular/platform-browser'

@Component({
    standalone: true,
    template: `<p-button
            label="Create"
            appManagedAccess="subnet"
            accessType="create"
            (onClick)="wasCreated = true"
            (hasAccess)="hasCreateAccess = $event"
        />
        <p-button
            label="Read"
            appManagedAccess="subnet"
            accessType="read"
            (onClick)="wasRead = true"
            (hasAccess)="hasReadAccess = $event"
        />
        <p-button label="Update" appManagedAccess="subnet" accessType="update" (onClick)="wasUpdated = true" />
        <p-button label="Delete" appManagedAccess="subnet" accessType="delete" (onClick)="wasDeleted = true" />
        <div appManagedAccess="subnet" accessType="create">This is subnet creation form.</div>
        <div appManagedAccess="subnet" accessType="delete" [hideOnNoAccess]="true">
            This is subnet removal component.
        </div> `,
    imports: [ManagedAccessDirective, Button],
})
class TestHostComponent {
    wasCreated = false
    wasRead = false
    wasUpdated = false
    wasDeleted = false
    hasCreateAccess = false
    hasReadAccess = false
}

describe('ManagedAccessDirective', () => {
    let hostComponent: TestHostComponent
    let fixture: ComponentFixture<TestHostComponent>
    let authService: AuthService
    let hasPrivilegeSpy: jasmine.Spy
    let createBtn: any
    let readBtn: any
    let updateBtn: any
    let deleteBtn: any
    let createDiv: any
    let deleteDiv: any

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
        expect(fixture.debugElement.children.length).toBeGreaterThanOrEqual(6)
        createBtn = fixture.debugElement.children[0]
        readBtn = fixture.debugElement.children[1]
        updateBtn = fixture.debugElement.children[2]
        deleteBtn = fixture.debugElement.children[3]
        createDiv = fixture.debugElement.children[4]
        deleteDiv = fixture.debugElement.children[5]

        fixture.detectChanges()
    })

    it('should create', () => {
        expect(hostComponent).toBeTruthy()
        expect(createBtn).toBeTruthy()
        expect(readBtn).toBeTruthy()
        expect(updateBtn).toBeTruthy()
        expect(deleteBtn).toBeTruthy()
        expect(createDiv).toBeTruthy()
        expect(deleteDiv).toBeTruthy()
    })

    it('should display no content when no access and hide on no access flag was used', () => {
        expect(deleteDiv).toBeTruthy()
        expect(deleteDiv.nativeElement.innerText).toBeFalsy()
    })

    it('should display feedback message about lack of privileges', () => {
        expect(createDiv).toBeTruthy()
        expect(createDiv.nativeElement.innerText).toContain("You don't have create privileges")
    })

    it('should emit if user has access', () => {
        expect(hostComponent.hasReadAccess).toBeTrue()
        expect(hostComponent.hasCreateAccess).toBeFalse()
    })

    it('should disable elements when no access', () => {
        expect(createBtn.query(By.css('button'))).toBeTruthy()
        expect(updateBtn.query(By.css('button'))).toBeTruthy()
        expect(deleteBtn.query(By.css('button'))).toBeTruthy()
        createBtn.query(By.css('button')).nativeElement.click()
        updateBtn.query(By.css('button')).nativeElement.click()
        deleteBtn.query(By.css('button')).nativeElement.click()
        fixture.detectChanges()
        expect(hostComponent.wasCreated).toBeFalse()
        expect(hostComponent.wasUpdated).toBeFalse()
        expect(hostComponent.wasDeleted).toBeFalse()
    })

    it('should not disable elements when has access', () => {
        expect(readBtn.query(By.css('button'))).toBeTruthy()
        readBtn.query(By.css('button')).nativeElement.click()
        fixture.detectChanges()
        expect(hostComponent.wasRead).toBeTrue()
    })
})
